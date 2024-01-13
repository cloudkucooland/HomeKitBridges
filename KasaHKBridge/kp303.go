package kasahkbridge

import (
	"net"
	"strconv"

	"github.com/brutella/hap/accessory"
	"github.com/brutella/hap/characteristic"
	"github.com/brutella/hap/log"
	"github.com/brutella/hap/service"

	"github.com/cloudkucooland/go-kasa"
)

type KP303 struct {
	*generic

	Outlets []*service.Outlet
}

func NewKP303(k kasa.KasaDevice, ip net.IP) *KP303 {
	acc := KP303{}
	acc.generic = &generic{}

	info := acc.configure(k.GetSysinfo.Sysinfo, ip)
	acc.A = accessory.New(info, accessory.TypeOutlet)
	acc.setID()

	os := int(acc.Sysinfo.NumChildren)
	acc.Outlets = make([]*service.Outlet, os, os+1)

	for i := 0; i < os; i++ {
		o := service.NewOutlet()
		acc.AddS(o.S)
		o.On.SetValue(acc.Sysinfo.Children[i].RelayState > 0)
		o.OutletInUse.SetValue(acc.Sysinfo.Children[i].RelayState > 0)

		n := characteristic.NewName()
		n.SetValue(acc.Sysinfo.Children[i].Alias)
		o.AddC(n.C)

		if dx, err := strconv.Atoi(acc.Sysinfo.Children[i].ID); err != nil {
			log.Info.Println(err.Error())
		} else {
			id := characteristic.NewIdentifier()
			id.SetValue(dx)
			o.AddC(id.C)
		}

		ai := characteristic.NewAccessoryIdentifier()
		ai.SetValue(acc.Sysinfo.Children[i].ID)
		o.AddC(ai.C)

		idx := i // local scope
		o.On.OnValueRemoteUpdate(func(newstate bool) {
			log.Info.Printf("setting [%s][%d] (%s) to [%t] from KP303 handler", acc.Sysinfo.Alias, idx, acc.Sysinfo.Children[idx].ID, newstate)
			if err := setChildRelayState(acc.ip, acc.Sysinfo.DeviceID, acc.Sysinfo.Children[idx].ID, newstate); err != nil {
				log.Info.Println(err.Error())
				return
			}
			o.OutletInUse.SetValue(newstate)
		})
		acc.Outlets[i] = o
	}

	return &acc
}

func (h *KP303) update(k kasa.KasaDevice, ip net.IP) {
	h.genericUpdate(k, ip)

	for i := 0; i < len(h.Outlets); i++ {
		if h.Outlets[i].On.Value() != (k.GetSysinfo.Sysinfo.Children[i].RelayState > 0) {
			log.Info.Printf("updating HomeKit: [%s][%d] relay %d", k.GetSysinfo.Sysinfo.Alias, i, k.GetSysinfo.Sysinfo.Children[i].RelayState)
			h.Outlets[i].On.SetValue(k.GetSysinfo.Sysinfo.Children[i].RelayState > 0)
			h.Outlets[i].OutletInUse.SetValue(k.GetSysinfo.Sysinfo.Children[i].RelayState > 0)
		}
	}
}
