package kasahkbridge

import (
	"net"

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
	acc.finalize()

	os := int(acc.Sysinfo.NumChildren)
	acc.Outlets = make([]*hs300outletSvc, os, os+1)

	for i := 0; i < os; i++ {
		o := NewHS300OutletSvc()
		acc.AddS(o.S)
		o.On.SetValue(acc.Sysinfo.Children[i].RelayState > 0)
		o.OutletInUse.SetValue(acc.Sysinfo.Children[i].RelayState > 0)

		n := characteristic.NewName()
		n.SetValue(acc.Sysinfo.Children[i].Alias)
		o.AddC(n.C)

		idx := i // local scope
		n.OnValueRemoteUpdate(func(newname string) {
			log.Info.Print("setting alias to [%s]", newname)
			if err := setChildRelayAlias(acc.ip, acc.Sysinfo.DeviceID, acc.Sysinfo.Children[idx].ID, newname); err != nil {
				log.Info.Println(err.Error())
				return
			}
		})

		o.On.OnValueRemoteUpdate(func(newstate bool) {
			log.Info.Printf("setting [%s][%d] (%s) to [%t] from HS300 handler", acc.Sysinfo.Alias, idx, acc.Sysinfo.Children[idx].ID, newstate)
			if err := setChildRelayState(acc.ip, acc.Sysinfo.DeviceID, acc.Sysinfo.Children[idx].ID, newstate); err != nil {
				log.Info.Println(err.Error())
				return
			}
			o.OutletInUse.SetValue(newstate)
		})

		acc.Outlets[i] = o
	}

	// acc.AddS(acc.KasaStatus.S)

	return &acc
}

type hs300outletSvc struct {
	*service.S

	On          *characteristic.On
	OutletInUse *characteristic.OutletInUse

	Volt *volt
	Watt *watt
	Amp  *amp
}

func NewHS300OutletSvc() *hs300outletSvc {
	svc := hs300outletSvc{}
	svc.S = service.New(service.TypeOutlet)

	svc.On = characteristic.NewOn()
	svc.AddC(svc.On.C)

	svc.OutletInUse = characteristic.NewOutletInUse()
	svc.AddC(svc.OutletInUse.C)

	svc.Volt = NewVolt()
	svc.AddC(svc.Volt.C)
	svc.Volt.SetValue(120)

	svc.Watt = NewWatt()
	svc.AddC(svc.Watt.C)
	svc.Watt.SetValue(1)

	svc.Amp = NewAmp()
	svc.AddC(svc.Amp.C)
	svc.Amp.SetValue(1)

	return &svc
}

func (h *HS300) update(k kasa.KasaDevice, ip net.IP) {
	h.genericUpdate(k, ip)

	for i := 0; i < len(h.Outlets); i++ {
		if h.Outlets[i].On.Value() != (k.GetSysinfo.Sysinfo.Children[i].RelayState > 0) {
			log.Info.Printf("updating HomeKit: [%s][%d] relay %d", k.GetSysinfo.Sysinfo.Alias, i, k.GetSysinfo.Sysinfo.Children[i].RelayState)
			h.Outlets[i].On.SetValue(k.GetSysinfo.Sysinfo.Children[i].RelayState > 0)
			h.Outlets[i].OutletInUse.SetValue(k.GetSysinfo.Sysinfo.Children[i].RelayState > 0)
		}

		// request emeter data for each outlet
		err := getEmeterChild(h.ip, h.Sysinfo.DeviceID, h.Sysinfo.Children[i].ID)
		if err != nil {
			return
		}
	}
}

func (h *HS300) updateEmeter(e kasa.EmeterRealtime) {
	if int(e.Slot) >= len(h.Outlets) {
		log.Info.Println("slot out of bounds: %s", e.Slot)
	}

	h.Outlets[e.Slot].Volt.SetValue(int(e.VoltageMV / 1000))
	h.Outlets[e.Slot].Watt.SetValue(int(e.PowerMW / 1000))
	h.Outlets[e.Slot].Amp.SetValue(int(e.CurrentMA))
}
