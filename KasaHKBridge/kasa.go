package kasahkbridge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/brutella/hap/accessory"
	"github.com/brutella/hap/log"

	"github.com/cloudkucooland/go-kasa"
)

// globals are OK since there will only ever be one bridge
var kasas map[string]kasaDevice
var kasasMu sync.RWMutex
var bufsize int = 2048
var packetconn *net.UDPConn
var broadcasts []net.IP

var pollInterval time.Duration = 30 * time.Second

// avoid allocations in the main loops -- could pre-scramble these
var relaySuccess = []byte(`{"system":{"set_relay_state":{"err_code":0}}}`)
var emeterSuccess = []byte(`{"smartlife.iot.dimmer":{"set_brightness":{"err_code":0}}}`)
var sysinfoPreamble = []byte(`"get_sysinfo"`)
var emeterPreamble = []byte(`{"emeter":{"get_realtime":{`)

const CHANGE_SLEEP_DURATION = (100 * time.Millisecond)

var discoverCmd = kasa.Scramble(kasa.CmdGetSysinfo)

type kasaDevice interface {
	getA() *accessory.A
	update(kasa.KasaDevice, net.IP)
	incomingEmeterData(kasa.EmeterRealtime)
	getLastUpdate() time.Time
	unreachable()
	getIPstring() string
	getAlias() string
	sysinfo() kasa.Sysinfo
}

type factoryFunc func(kasa.KasaDevice, net.IP) kasaDevice

// wrapper since New* returns a poninter but a pointer to an interface is useless
func wrap[T kasaDevice](fn func(kasa.KasaDevice, net.IP) T) factoryFunc {
	return func(k kasa.KasaDevice, ip net.IP) kasaDevice {
		return fn(k, ip)
	}
}

var deviceFactories = map[string]func(kasa.KasaDevice, net.IP) kasaDevice{
	"HS103(US)": wrap(NewHS103),
	"HS200(US)": wrap(NewHS200),
	"HS210(US)": wrap(NewHS200),
	"HS220(US)": wrap(NewHS220),
	"KP115(US)": wrap(NewKP115),
	"KP303(US)": wrap(NewKP303),
	"HS300(US)": wrap(NewHS300),
}

// Listener is the go process that listens for UDP responses from the Kasa devices
func Listener(ctx context.Context, refresh chan bool) {
	var err error
	packetconn, err = net.ListenUDP("udp", &net.UDPAddr{IP: nil, Port: 0})
	if err != nil {
		log.Info.Println(err.Error())
		return
	}
	defer packetconn.Close()

	buffer := make([]byte, bufsize)

	for {
		select {
		case <-ctx.Done():
			log.Info.Println("shutting down listener")
			return
		default:
		}

		_ = packetconn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		n, addr, err := packetconn.ReadFromUDP(buffer)
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			}
			log.Info.Println(err.Error())
			return
		}

		d := kasa.Unscramble(buffer[:n])

		// ignore success messages
		if bytes.Equal(d, relaySuccess) || bytes.Equal(d, emeterSuccess) {
			continue
		}

		if !(bytes.Contains(d, sysinfoPreamble) || bytes.HasPrefix(d, emeterPreamble)) {
			log.Info.Printf("unknown message from %s: %s", addr.IP.String(), string(d))
			continue
		}

		var kd kasa.KasaDevice
		if err = json.Unmarshal(d, &kd); err != nil {
			log.Info.Printf("unmarshal failed: %s", err.Error())
			continue
		}

		if err := kd.GetSysinfo.Sysinfo.OK(); err != nil {
			log.Info.Println(err)
		}

		if bytes.HasPrefix(d, emeterPreamble) {
			updateEmeter(kd.Emeter.Realtime, addr.IP.String())
			continue
		}

		kasasMu.RLock()
		k, ok := kasas[kd.GetSysinfo.Sysinfo.DeviceID]
		kasasMu.RUnlock()

		// potential for race, but exceedingly unlikely since this only hit during
		// initialization except in VERY rare cases of a new device being brought online
		if !ok {
			if factory, kOk := deviceFactories[kd.GetSysinfo.Sysinfo.Model]; kOk {
				kasasMu.Lock()
				kasas[kd.GetSysinfo.Sysinfo.DeviceID] = factory(kd, addr.IP)
				kasasMu.Unlock()
				refresh <- true // blocking is OK during initialization
			} else {
				log.Info.Printf("unknown device type (%s)", kd.GetSysinfo.Sysinfo.Model)
			}
		} else {
			k.update(kd, addr.IP)
		}
	}
}

// Startup
func Startup(ctx context.Context, refresh chan bool, path string) error {
	kasas = make(map[string]kasaDevice)

	kasa.SetLogger(log.Info)
	loadCache(path)

	if err := SetBroadcasts(); err != nil {
		return err
	}

	log.Info.Println("Starting initial discovery (3 seconds)")

	timeout, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	discover()

FIRST:
	for {
		select {
		case <-timeout.Done():
			break FIRST
		case <-refresh:
			// drain refresh messages that happened before the discovery timeout
		}
	}

	log.Info.Printf("Initial discovery complete, found %d devices", len(kasas))
	cancel()

	// start the routine poller
	go poller(ctx)

	return nil
}

// Devices returns the accessories ready for HAP to start a hap.Server
func Devices() []*accessory.A {
	var a []*accessory.A

	kasasMu.RLock()
	for _, k := range kasas {
		a = append(a, k.getA())
	}
	kasasMu.RUnlock()

	return a
}

func SetBroadcasts() error {
	var err error
	log.Debug.Printf("updating broadcasts")
	broadcasts, err = kasa.BroadcastAddresses()
	return err
}

func poller(ctx context.Context) {
	t := time.NewTicker(pollInterval)
	defer t.Stop()

	for {
		discover()

		n := time.Now()
		b := n.Add(0 - (5 * pollInterval))

		kasasMu.RLock()
		for _, k := range kasas {
			if k.getLastUpdate().Before(b) {
				k.unreachable()
			}
		}
		kasasMu.RUnlock()

		select {
		case <-ctx.Done():
			log.Info.Printf("poller: contexted canceled")
			return
		case <-t.C:
			// log.Debug.Printf("poller: tick")
		}
	}
}

func discover() {
	for _, b := range broadcasts {
		if _, err := packetconn.WriteToUDP(discoverCmd, &net.UDPAddr{IP: b, Port: 9999}); err != nil {
			log.Info.Printf("discovery failed for %s: %s", b.String(), err.Error())
			continue
		}
	}
}

func setCountdown(ip net.IP, target bool, dur int) error {
	k, _ := newKasaIP(ip)

	// remove any existing countdowns
	if err := k.ClearCountdownRules(); err != nil {
		log.Info.Println(err.Error())
		return err
	}

	// add our new countdown
	if err := k.AddCountdownRule(dur, target, "added from kasahkb"); err != nil {
		log.Info.Println(err.Error())
		return err
	}

	return nil
}

func getEmeterUDP(ip net.IP) error {
	payload := kasa.Scramble(kasa.CmdGetEmeter)

	if _, err := packetconn.WriteToUDP(payload, &net.UDPAddr{IP: ip, Port: 9999}); err != nil {
		log.Info.Printf("get emeter failed: %s", err.Error())
		return err
	}

	return nil
}

func getEmeterChildUDP(ip net.IP, parent, child string) error {
	full := fmt.Sprintf("%s%s", parent, child)

	cmd := fmt.Sprintf(kasa.CmdGetEmeterChild, full)
	payload := kasa.Scramble(cmd)

	if _, err := packetconn.WriteToUDP(payload, &net.UDPAddr{IP: ip, Port: 9999}); err != nil {
		log.Info.Printf("get emeter child failed: %s", err.Error())
		return err
	}

	return nil
}

func updateEmeter(em kasa.EmeterRealtime, ip string) error {
	// this is an acceptable O(n) loop given typical install sizes
	kasasMu.RLock()
	for _, device := range kasas {
		if device.getIPstring() == ip {
			device.incomingEmeterData(em)
		}
	}
	kasasMu.RUnlock()

	return nil
}

func newKasaIP(ip net.IP) (*kasa.Device, error) {
	d := kasa.Device{
		IP: ip,
		OverrideUDP: func(ctx context.Context, cmd string) error {
			if _, err := packetconn.WriteToUDP(kasa.Scramble(cmd), &net.UDPAddr{IP: ip, Port: 9999}); err != nil {
				log.Info.Printf("udp write failed: %s", err.Error())
				return err
			}
			// so we don't overwhelm the breakers when 50 deviced get flipped at once
			time.Sleep(CHANGE_SLEEP_DURATION)
			return nil
		},
	}
	return &d, nil
}
