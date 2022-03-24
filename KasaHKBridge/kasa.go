package kasahkbridge

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/brutella/hap/accessory"
	// "github.com/brutella/hap/characteristic"
	"github.com/brutella/hap/log"
	// "github.com/brutella/hap/service"

	"github.com/cloudkucooland/go-kasa"
)

var kasas map[string]kasaDevice
var refresh chan bool
var bufsize int = 2048
var packetconn *net.UDPConn
var broadcasts []net.IP
var pollInterval time.Duration = 30

type kasaDevice interface {
	getA() *accessory.A
	getName() string
	update(kasa.KasaDevice, net.IP)
	getLastUpdate() time.Time
	unreachable()
}

// Listener is the go process that listens for UDP responses from the Kasa devices
func Listener(ctx context.Context) {
	var err error
	packetconn, err = net.ListenUDP("udp", &net.UDPAddr{IP: nil, Port: 0})
	if err != nil {
		log.Info.Println(err.Error())
		return
	}
	defer packetconn.Close()

	done := make(chan bool)
	buffer := make([]byte, bufsize)

	go func() {
		for {
			n, addr, err := packetconn.ReadFromUDP(buffer)
			if err != nil {
				log.Info.Println(err.Error())
				done <- true
				return
			}

			d := kasa.Unscramble(buffer[:n])
			s := string(d)
			// log.Info.Println(s)

			if s == `{"system":{"set_relay_state":{"err_code":0}}}` {
				continue
			}

			if s == `{"smartlife.iot.dimmer":{"set_brightness":{"err_code":0}}}` {
				continue
			}

			if !strings.Contains(s, `"get_sysinfo"`) {
				log.Info.Printf("unknown message from %s: %s", addr.IP, s)
				continue
			}

			var kd kasa.KasaDevice
			if err = json.Unmarshal(d, &kd); err != nil {
				log.Info.Printf("unmarshal failed: %s\n", err.Error())
				continue
			}

			k, ok := kasas[kd.GetSysinfo.Sysinfo.DeviceID]

			// make the device, store it, trigger a refresh
			if !ok {
				switch kd.GetSysinfo.Sysinfo.Model {
				case "HS103(US)":
					kasas[kd.GetSysinfo.Sysinfo.DeviceID] = NewHS103(kd, addr.IP)
					refresh <- true
				case "HS200(US)", "HS210(US)":
					kasas[kd.GetSysinfo.Sysinfo.DeviceID] = NewHS200(kd, addr.IP)
					refresh <- true
				case "HS220(US)":
					kasas[kd.GetSysinfo.Sysinfo.DeviceID] = NewHS220(kd, addr.IP)
					refresh <- true
				case "KP115(US)":
					kasas[kd.GetSysinfo.Sysinfo.DeviceID] = NewKP115(kd, addr.IP)
					refresh <- true
				case "KP303(US)":
					kasas[kd.GetSysinfo.Sysinfo.DeviceID] = NewKP303(kd, addr.IP)
					refresh <- true
				default:
					log.Info.Printf("unknown device type (%s) %s\n", addr.IP.String(), d)
				}
			} else {
				k.update(kd, addr.IP)
			}
		}
	}()

	select {
	case <-ctx.Done():
		log.Info.Println("shutting down listener")
	case <-done:
		log.Info.Println("shutting down listener due to error")
	}
}

// Startup
func Startup(ctx context.Context, r chan bool) error {
	kasas = make(map[string]kasaDevice)
	refresh = r

	kasa.SetLogger(log.Info)

	broadcasts, _ = kasa.BroadcastAddresses()

	// wait for the Listener to get going -- use a proper sync...
	time.Sleep(1 * time.Second)

	to, _ := context.WithTimeout(context.Background(), 3*time.Second)
	discover()

FIRST:
	for {
		select {
		case <-to.Done():
			break FIRST
		case <-refresh:
			// drain the buffer during initial discovery
		}
	}

	log.Info.Printf("Initial discovery complete, found %d devices\n", len(kasas))

	// start the routine poller
	go poller(ctx)

	return nil
}

// Devices returns the accessories ready for HAP to start a hap.Server
func Devices() []*accessory.A {
	var a []*accessory.A

	for _, k := range kasas {
		a = append(a, k.getA())
	}

	return a
}

func poller(ctx context.Context) {
	t := time.Tick(pollInterval * time.Second)

	for {
		discover()

		n := time.Now()
		b := n.Add(0 - (5 * pollInterval * time.Second))
		for _, k := range kasas {
			if k.getLastUpdate().Before(b) {
				k.unreachable()
			}
		}

		select {
		case <-ctx.Done():
			log.Info.Printf("poller: contexted canceled")
			return
		case <-t:
			// log.Info.Printf("poller: tick")
		}
	}
}

// discover, relay and brightness should be fast, skip the overhead of kasa.New()...
func discover() {
	payload := kasa.Scramble(kasa.CmdGetSysinfo)

	for _, b := range broadcasts {
		_, err := packetconn.WriteToUDP(payload, &net.UDPAddr{IP: b, Port: 9999})
		if err != nil {
			log.Info.Printf("discovery failed: %s\n", err.Error())
			return
		}
	}
}

func setRelayState(ip net.IP, newstate bool) error {
	state := 0
	if newstate {
		state = 1
	}
	cmd := fmt.Sprintf(kasa.CmdSetRelayState, state)
	payload := kasa.Scramble(cmd)

	if _, err := packetconn.WriteToUDP(payload, &net.UDPAddr{IP: ip, Port: 9999}); err != nil {
		log.Info.Printf("set relay state failed: %s\n", err.Error())
		return err
	}

	return nil
}

func setBrightness(ip net.IP, brightness int) error {
	cmd := fmt.Sprintf(kasa.CmdSetBrightness, brightness)
	payload := kasa.Scramble(cmd)

	if _, err := packetconn.WriteToUDP(payload, &net.UDPAddr{IP: ip, Port: 9999}); err != nil {
		log.Info.Printf("set brightness failed: %s\n", err.Error())
		return err
	}

	return nil
}

// this doesn't need to be fast...
func setCountdown(ip net.IP, target bool, dur int) error {
	k, _ := kasa.NewDevice(ip.String())

	// remove any existing countdowns
	if err := k.ClearCountdownRules(); err != nil {
		log.Info.Printf(err.Error())
		return err
	}

	// add our new countdown
	if err := k.AddCountdownRule(dur, target, "added from kasahkb"); err != nil {
		log.Info.Printf(err.Error())
		return err
	}

	return nil
}

func setChildRelayState(ip net.IP, parent, child string, newstate bool) error {
	state := 0
	if newstate {
		state = 1
	}

	idx := fmt.Sprintf("%s%s", parent, child)

	cmd := fmt.Sprintf(kasa.CmdSetRelayStateChild, idx, state)
	payload := kasa.Scramble(cmd)

	if _, err := packetconn.WriteToUDP(payload, &net.UDPAddr{IP: ip, Port: 9999}); err != nil {
		log.Info.Printf("set child relay failed: %s\n", err.Error())
		return err
	}

	return nil
}

func setChildRelayAlias(ip net.IP, index string, alias string) error {
	k, err := kasa.NewDevice(ip.String())
	if err != nil {
		log.Info.Printf(err.Error())
		return err
	}
	if err := k.SetChildAlias(index, alias); err != nil {
		log.Info.Printf(err.Error())
		return err
	}
	return nil
}
