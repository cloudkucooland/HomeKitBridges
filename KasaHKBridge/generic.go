package kasahkbridge

import (
	"encoding/hex"
	"net"
	"time"

	"github.com/brutella/hap/accessory"
	"github.com/brutella/hap/characteristic"
	"github.com/brutella/hap/log"
	"github.com/brutella/hap/service"

	"github.com/cloudkucooland/go-kasa"
)

// included in all device types
type generic struct {
	lastUpdate time.Time // last time the device responded (move to Status)
	*accessory.A

	KasaStatus *statusSvc   // a service for displaying status info
	ip         net.IP       // would probably be better to use string
	Sysinfo    kasa.Sysinfo // contents of the last response from the device
}

func (g *generic) getA() *accessory.A {
	return g.A
}

func (g *generic) getLastUpdate() time.Time {
	return g.lastUpdate
}

func (g *generic) unreachable() {
	// figure out how to tell homekit that this device has gone away....
	// probably under the bridge accessory
}

func (g *generic) configure(k kasa.Sysinfo, ip net.IP) accessory.Info {
	g.Sysinfo = k
	g.lastUpdate = time.Now()
	g.ip = ip
	g.KasaStatus = NewStatusSvc()

	info := accessory.Info{
		Name:         k.Alias,
		SerialNumber: k.DeviceID,
		Manufacturer: "TP-Link Kasa Smart",
		Model:        k.Model,
		Firmware:     k.SWVersion,
	}

	return info
}

func id(g *generic) uint64 {
	var ID uint64

	// convert 12 chars of the deviceId into a uint64 for the ID
	mac, err := hex.DecodeString(g.Sysinfo.DeviceID[:12])
	if err != nil {
		log.Info.Printf("weird kasa devid: %s", err.Error())
	}
	for k, v := range mac {
		ID += uint64(v) << (12 - k) * 8
	}
	return ID
}

func (g *generic) finalize() {
	// set the ID so the device remains consistent in homekit across reboots
	// i := id(g)
	g.A.Id = id(g)
}

func (g *generic) genericUpdate(k kasa.KasaDevice, ip net.IP) {
	if g.ip.String() != ip.String() {
		log.Info.Printf("updating ip address: [%s] -> [%s] (%s)\n", g.ip, ip, k.GetSysinfo.Sysinfo.Alias)
		g.ip = ip
	}

	if g.Sysinfo.Alias != k.GetSysinfo.Sysinfo.Alias {
		log.Info.Printf("renaming: [%s] -> [%s]\n", g.Sysinfo.Alias, k.GetSysinfo.Sysinfo.Alias)
		g.Sysinfo.Alias = k.GetSysinfo.Sysinfo.Alias
		g.Info.Name.SetValue(k.GetSysinfo.Sysinfo.Alias)
	}

	// log.Info.Printf("[%s] RSSI: [%d]", g.Sysinfo.Alias, k.GetSysinfo.Sysinfo.RSSI)
	g.KasaStatus.RSSI.SetValue(int(k.GetSysinfo.Sysinfo.RSSI))
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

type statusSvc struct {
	*service.S

	Name *characteristic.Name
	RSSI *rssi
}

func NewStatusSvc() *statusSvc {
	svc := statusSvc{}
	svc.S = service.New("E8800001")
	svc.S.Primary = false
	svc.S.Hidden = true

	svc.Name = characteristic.NewName()
	svc.Name.SetValue("Kasa Status")
	svc.S.AddC(svc.Name.C)

	svc.RSSI = NewRSSI()
	svc.S.AddC(svc.RSSI.C)

	return &svc
}

func (g *generic) updateEmeter(e kasa.EmeterRealtime) {
	log.Info.Printf("emeter update from non-emeter device: %s %+v", g.ip, e)
}

func (g *generic) getIP() net.IP {
	return g.ip
}
