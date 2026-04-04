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
	StatusFault  *characteristic.StatusFault
	ip           net.IP
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
	g.StatusFault.SetValue(characteristic.StatusFaultGeneralFault)

	// try conecting using a TCP connection to see if it is really down or just dropping UDP
	k, err := newKasaIP(g.ip)
	if err != nil {
		log.Info.Println(err.Error())
		return
	}
	if _, err := k.GetWIFIStatus(); err != nil {
		log.Info.Println(err.Error())
		return
	}
}

func (g *generic) configure(k kasa.Sysinfo, ip net.IP) accessory.Info {
	g.Sysinfo = k
	g.lastUpdate = time.Now()
	g.ip = ip

	g.RSSI = NewRSSI()
	g.StatusActive = characteristic.NewStatusActive()
	g.StatusActive.SetValue(true)
	g.StatusFault = characteristic.NewStatusFault()
	g.StatusFault.SetValue(characteristic.StatusFaultNoFault)

	info := accessory.Info{
		Name:         k.Alias,
		SerialNumber: k.DeviceID,           // deprecated
		Manufacturer: "TP-Link Kasa Smart", // deprecated
		Model:        k.Model,              // deprecated
		Firmware:     k.SWVersion,          // deprecated
	}

	return info
}

// convert 12 chars of the deviceId into a uint64 for the ID, g.A must exist first, so can't be part of g.configure
func (g *generic) setID() {
	mac, err := hex.DecodeString(g.Sysinfo.DeviceID[:12])
	if err != nil {
		log.Info.Printf("weird kasa DeviceID: %s", err.Error())
		return
	}
	var ID uint64
	for k, v := range mac {
		ID += uint64(v) << (12 - k) * 8
	}
	g.A.Id = ID

	g.Info.Name.Permissions = []string{characteristic.PermissionRead, characteristic.PermissionWrite}

	// doesn't ever send -- Apple removed this from HomeKit ~2019
	g.Info.Name.OnValueRemoteUpdate(func(newname string) {
		log.Info.Printf("[%s] renamed to %s", g.Sysinfo.Alias, newname)
		// rename it on the device...
	})
}

func (g *generic) genericUpdate(k kasa.KasaDevice, newip net.IP) {
	// if it was not responding, but is now...
	if !g.StatusActive.Value() {
		log.Info.Printf("[%s] responding again", g.Sysinfo.Alias)
		g.StatusActive.SetValue(true)
		g.StatusFault.SetValue(characteristic.StatusFaultNoFault)
	}

	// netip.IP.Compare() exists but net.IP.Compare() does not
	if g.ip.String() != newip.String() {
		log.Info.Printf("updating ip address: [%s] -> [%s] (%s)", g.ip, newip, k.GetSysinfo.Sysinfo.Alias)
		g.ip = newip
	}

	if g.Sysinfo.Alias != k.GetSysinfo.Sysinfo.Alias {
		log.Info.Printf("renaming: [%s] -> [%s]", g.Sysinfo.Alias, k.GetSysinfo.Sysinfo.Alias)
		g.Sysinfo.Alias = k.GetSysinfo.Sysinfo.Alias
		// HomeKit now ignores this
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

func (g *generic) incomingEmeterData(e kasa.EmeterRealtime) {
	log.Info.Printf("emeter update from non-emeter device: %s %+v", g.ip, e)
}

func (g *generic) incomingDimmerData(e kasa.Dimmer) {
	log.Info.Printf("dimmer update from non-dimmer device: %s %+v", g.ip, e)
}

func (g *generic) getIPstring() string {
	return g.ip.String()
}

func (g *generic) getAlias() string {
	return g.Sysinfo.Alias
}

func intToState(i uint) string {
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
