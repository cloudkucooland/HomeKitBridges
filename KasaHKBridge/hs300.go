package kasahkbridge

import (
	"fmt"
	"net"
	"strconv"

	"github.com/brutella/hap/accessory"
	"github.com/brutella/hap/characteristic"
	"github.com/brutella/hap/log"
	"github.com/brutella/hap/service"

	"github.com/cloudkucooland/go-kasa"
)

type HS300 struct {
	*generic

	Outlets []*hs300outletSvc
}

func NewHS300(k kasa.KasaDevice, ip net.IP) *HS300 {
	acc := HS300{}
	acc.generic = &generic{}

	info := acc.configure(k.GetSysinfo.Sysinfo, ip)
	acc.A = accessory.New(info, accessory.TypeOutlet)
	acc.setID()

	outlets := int(acc.Sysinfo.NumChildren)
	acc.Outlets = make([]*hs300outletSvc, outlets, outlets+1)

	for i := 0; i < outlets; i++ {
		idx := i // local scope

		o := newHS300OutletSvc()
		o.On.SetValue(acc.Sysinfo.Children[idx].RelayState > 0)
		o.OutletInUse.SetValue(acc.Sysinfo.Children[idx].RelayState > 0)
		o.Name.SetValue(acc.Sysinfo.Children[idx].Alias)
		id := fmt.Sprintf("%s%s", acc.Sysinfo.DeviceID[32:], acc.Sysinfo.Children[idx].ID)
		o.AccIdentifier.SetValue(id)
		if dx, err := strconv.ParseInt(id, 16, 64); err != nil {
			log.Info.Println(err.Error())
		} else {
			o.ID.SetValue(int(dx))
			o.Id = uint64(dx)
		}

		o.On.OnValueRemoteUpdate(func(newstate bool) {
			log.Info.Printf("[%s][%d] %s", acc.Sysinfo.Alias, idx, boolToState(newstate))
			full := fmt.Sprintf("%s%s", acc.Sysinfo.DeviceID, acc.Sysinfo.Children[idx].ID)
			k, _ := newKasaIP(acc.ip)
			if err := k.SetRelayStateChild(full, newstate); err != nil {
				log.Info.Println(err.Error())
				return
			}
			o.OutletInUse.SetValue(newstate)
		})

		o.Name.OnValueRemoteUpdate(func(newname string) {
			log.Info.Printf("[%s][%d] new name %s", acc.Sysinfo.Alias, idx, newname)
			full := fmt.Sprintf("%s%s", acc.Sysinfo.DeviceID, acc.Sysinfo.Children[idx].ID)
			k, _ := newKasaIP(acc.ip)
			if err := k.SetChildAlias(full, newname); err != nil {
				log.Info.Println(err.Error())
				return
			}
		})

		acc.AddS(o.S)
		acc.Outlets[idx] = o
	}

	return &acc
}

type hs300outletSvc struct {
	*service.S

	On            *characteristic.On
	OutletInUse   *characteristic.OutletInUse
	Name          *characteristic.Name
	ID            *characteristic.Identifier
	AccIdentifier *characteristic.AccessoryIdentifier

	Volt *volt
	Watt *watt
	Amp  *amp

	StatusFault *characteristic.StatusFault
}

func newHS300OutletSvc() *hs300outletSvc {
	svc := hs300outletSvc{}
	svc.S = service.New(service.TypeOutlet)

	svc.On = characteristic.NewOn()
	svc.AddC(svc.On.C)

	svc.OutletInUse = characteristic.NewOutletInUse()
	svc.AddC(svc.OutletInUse.C)

	// doesn't work (anymore, did in older versions of HAP)
	svc.Name = characteristic.NewName()
	svc.Name.Permissions = []string{characteristic.PermissionRead, characteristic.PermissionWrite}
	svc.AddC(svc.Name.C)

	svc.ID = characteristic.NewIdentifier()
	svc.AddC(svc.ID.C)
	svc.ID.SetValue(0)

	svc.AccIdentifier = characteristic.NewAccessoryIdentifier()
	svc.AddC(svc.AccIdentifier.C)
	svc.AccIdentifier.SetValue("0")

	svc.Volt = NewVolt()
	svc.AddC(svc.Volt.C)
	svc.Volt.SetValue(120)

	svc.Watt = NewWatt()
	svc.AddC(svc.Watt.C)
	svc.Watt.SetValue(0)

	svc.Amp = NewAmp()
	svc.AddC(svc.Amp.C)
	svc.Amp.SetValue(0)

	svc.StatusFault = characteristic.NewStatusFault()
	svc.AddC(svc.StatusFault.C)
	svc.StatusFault.SetValue(characteristic.StatusFaultNoFault)

	return &svc
}

func (h *HS300) update(k kasa.KasaDevice, ip net.IP) {
	h.genericUpdate(k, ip)

	for i := 0; i < len(h.Outlets); i++ {
		if h.Outlets[i].On.Value() != (k.GetSysinfo.Sysinfo.Children[i].RelayState > 0) {
			log.Info.Printf("[%s][%d] %s", k.GetSysinfo.Sysinfo.Alias, i, intToState(k.GetSysinfo.Sysinfo.Children[i].RelayState))
			h.Outlets[i].On.SetValue(k.GetSysinfo.Sysinfo.Children[i].RelayState > 0)
			h.Outlets[i].OutletInUse.SetValue(k.GetSysinfo.Sysinfo.Children[i].RelayState > 0)
		}

		if h.Outlets[i].Name.Value() != k.GetSysinfo.Sysinfo.Children[i].Alias {
			log.Info.Printf("updating HomeKit: [%s][%d] name %s", k.GetSysinfo.Sysinfo.Alias, i, k.GetSysinfo.Sysinfo.Children[i].Alias)
			h.Outlets[i].Name.SetValue(k.GetSysinfo.Sysinfo.Children[i].Alias)
		}

		// request emeter data for each outlet
		if err := getEmeterChildUDP(h.ip, h.Sysinfo.DeviceID, h.Sysinfo.Children[i].ID); err != nil {
			log.Info.Println(err.Error())
		}

		if h.Outlets[i].On.Value() && h.Outlets[i].Amp.Value() < 10 {
			if h.Outlets[i].StatusFault.Value() == characteristic.StatusFaultNoFault {
				log.Info.Printf("ALERT: [%s] is ON but drawing no current!", k.GetSysinfo.Sysinfo.Children[i].Alias)
				h.Outlets[i].StatusFault.SetValue(characteristic.StatusFaultGeneralFault)
			}
		} else {
			if h.Outlets[i].StatusFault.Value() == characteristic.StatusFaultGeneralFault {
				log.Info.Printf("CLEAR: [%s] is ON and drawing current", k.GetSysinfo.Sysinfo.Children[i].Alias)
				h.Outlets[i].StatusFault.SetValue(characteristic.StatusFaultNoFault)
			}
		}

		v := h.Outlets[i].Volt.Value()
		if v < 114 {
			log.Info.Printf("ALERT: [%s][%s] low voltage: %dV", k.GetSysinfo.Sysinfo.Alias, k.GetSysinfo.Sysinfo.Children[i].Alias, v)
		}
		if v > 127 {
			log.Info.Printf("ALERT: [%s][%s] high voltage: %dV", k.GetSysinfo.Sysinfo.Alias, k.GetSysinfo.Sysinfo.Children[i].Alias, v)
		}
	}
}

func (h *HS300) incomingEmeterData(e kasa.EmeterRealtime) {
	if int(e.Slot) >= len(h.Outlets) {
		log.Info.Printf("slot out of bounds: %d", e.Slot)
	}

	h.Outlets[e.Slot].Volt.SetValue(int(e.VoltageMV / 1000))
	h.Outlets[e.Slot].Watt.SetValue(int(e.PowerMW / 1000))
	h.Outlets[e.Slot].Amp.SetValue(int(e.CurrentMA))
}
