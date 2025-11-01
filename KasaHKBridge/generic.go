package kasahkbridge

import (
	"encoding/hex"
	"net"
	"time"

	"github.com/brutella/hap/accessory"
	"github.com/brutella/hap/characteristic"
	"github.com/brutella/hap/log"

	"github.com/cloudkucooland/go-kasa"
)

// included in all device types
type generic struct {
	*accessory.A
	lastUpdate   time.Time // last time the device responded
	RSSI         *rssi
	StatusActive *characteristic.StatusActive
	ip           net.IP       // would probably be better to use string
	Sysinfo      kasa.Sysinfo // contents of the last response from the device
}

func (g *generic) getA() *accessory.A {
	return g.A
}

func (g *generic) getLastUpdate() time.Time {
	return g.lastUpdate
}

func (g *generic) sysinfo() kasa.Sysinfo {
	return g.Sysinfo
}

func (g *generic) unreachable() {
	if !g.StatusActive.Value() {
		return
	}

	log.Info.Printf("[%s] has not responded", g.Sysinfo.Alias)
	g.StatusActive.SetValue(false)

	// try conecting using a TCP connection to see if it is really down or just dropping UDP
	k, _ := kasa.NewDevice(g.ip.String())
	if _, err := k.GetWIFIStatus(); err != nil {
		log.Info.Println(err.Error())
		return
	}

	/* log.Info.Printf("[%s] sending reboot", g.Sysinfo.Alias)
	if err := k.Reboot(); err != nil {
		log.Info.Printf(err.Error())
	} */
}

func (g *generic) configure(k kasa.Sysinfo, ip net.IP) accessory.Info {
	g.Sysinfo = k
	g.lastUpdate = time.Now()
	g.ip = ip

	g.RSSI = NewRSSI()
	g.StatusActive = characteristic.NewStatusActive()
	g.StatusActive.SetValue(true)

	info := accessory.Info{
		Name:         k.Alias,
		SerialNumber: k.DeviceID,
		Manufacturer: "TP-Link Kasa Smart",
		Model:        k.Model,
		Firmware:     k.SWVersion,
	}

	return info
}

// convert 12 chars of the deviceId into a uint64 for the ID, g.A must exist first, so can't be part of g.configure
func (g *generic) setID() {
	mac, err := hex.DecodeString(g.Sysinfo.DeviceID[:12])
	if err != nil {
		log.Info.Printf("weird kasa devid: %s", err.Error())
		return
	}
	var ID uint64
	for k, v := range mac {
		ID += uint64(v) << (12 - k) * 8
	}
	g.A.Id = ID

	// doesn't ever send
	g.Info.Name.OnValueRemoteUpdate(func(newname string) {
		log.Info.Printf("[%s] renamed to %s", g.Sysinfo.Alias, newname)
		// rename it on the device...
	})
}

func (g *generic) genericUpdate(k kasa.KasaDevice, ip net.IP) {
	// if it was not responding, but is now...
	if !g.StatusActive.Value() {
		log.Info.Printf("[%s] respnding again", g.Sysinfo.Alias)
		g.StatusActive.SetValue(true)
	}

	if g.ip.String() != ip.String() {
		log.Info.Printf("updating ip address: [%s] -> [%s] (%s)", g.ip, ip, k.GetSysinfo.Sysinfo.Alias)
		g.ip = ip
	}

	if g.Sysinfo.Alias != k.GetSysinfo.Sysinfo.Alias {
		log.Info.Printf("renaming: [%s] -> [%s]", g.Sysinfo.Alias, k.GetSysinfo.Sysinfo.Alias)
		g.Sysinfo.Alias = k.GetSysinfo.Sysinfo.Alias
		g.Info.Name.SetValue(k.GetSysinfo.Sysinfo.Alias)
	}

	g.RSSI.SetValue(int(k.GetSysinfo.Sysinfo.RSSI))
	if k.GetSysinfo.Sysinfo.RSSI < -95 {
		log.Info.Printf("[%s] weak WIFI signal: [%d]", g.Sysinfo.Alias, k.GetSysinfo.Sysinfo.RSSI)
	}
	g.lastUpdate = time.Now()
}

// kasa program mode to hap program mode
func kpm2hpm(kasaMode string) int {
	i := characteristic.ProgramModeNoProgramScheduled

	switch kasaMode {
	case "add_rule":
		i = characteristic.ProgramModeProgramScheduled
	case "count_down":
		i = characteristic.ProgramModeProgramScheduledManualMode
	case "none", "delete_all_rules":
		i = characteristic.ProgramModeNoProgramScheduled
	default:
		i = characteristic.ProgramModeNoProgramScheduled
	}
	return i
}

func (g *generic) updateEmeter(e kasa.EmeterRealtime) {
	log.Info.Printf("emeter update from non-emeter device: %s %+v", g.ip, e)
}

func (g *generic) getIP() net.IP {
	return g.ip
}

func (g *generic) getAlias() string {
	return g.Sysinfo.Alias
}

func intToState(i uint8) string {
	if i == 1 {
		return "On"
	}
	return "Off"
}

func boolToState(b bool) string {
	if b {
		return "On"
	}
	return "Off"
}
