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
		idx := i // force local scope - especially for handler

		o := service.NewOutlet()
		o.On.SetValue(acc.Sysinfo.Children[idx].RelayState > 0)
		o.OutletInUse.SetValue(acc.Sysinfo.Children[idx].RelayState > 0)

		// name doesn't display correctly
		n := characteristic.NewName()
		n.Permissions = []string{characteristic.PermissionRead, characteristic.PermissionWrite}
		n.SetValue(acc.Sysinfo.Children[idx].Alias)
		o.AddC(n.C)

		// Identifier doesn't seem to much help - but doesn't hurt
		// ServiceLabelIndex seems to help keep the service correct across backup/restore, I think
		id := fmt.Sprintf("%s%s", acc.Sysinfo.DeviceID[32:], acc.Sysinfo.Children[idx].ID)
		if dx, err := strconv.ParseInt(id, 16, 64); err != nil {
			log.Info.Println(err.Error())
		} else {
			id := characteristic.NewIdentifier()
			id.SetValue(int(dx))
			o.AddC(id.C)
		}

		// AccessoryIdentifier doesn't seem to much help - but doesn't hurt
		ai := characteristic.NewAccessoryIdentifier()
		ai.SetValue(id)
		o.AddC(ai.C)

		o.On.OnValueRemoteUpdate(func(newstate bool) {
			log.Info.Printf("[%s][%d] %s", acc.Sysinfo.Alias, idx, boolToState(newstate))
			if err := setChildRelayState(acc.ip, acc.Sysinfo.DeviceID, acc.Sysinfo.Children[idx].ID, newstate); err != nil {
				log.Info.Println(err.Error())
				return
			}
			o.OutletInUse.SetValue(newstate)
		})

		n.OnValueRemoteUpdate(func(newname string) {
			log.Info.Printf("[%s][%d] new name %s", acc.Sysinfo.Alias, idx, newname)
			/* if err := setChildRelayState(acc.ip, acc.Sysinfo.DeviceID, acc.Sysinfo.Children[idx].ID, newstate); err != nil {
				log.Info.Println(err.Error())
				return
			} */
		})

		acc.Outlets[idx] = o
		acc.AddS(o.S)
	}

	return &acc
}

func (h *KP303) update(k kasa.KasaDevice, ip net.IP) {
	h.genericUpdate(k, ip)

	for i := 0; i < len(h.Outlets); i++ {
		if h.Outlets[i].On.Value() != (k.GetSysinfo.Sysinfo.Children[i].RelayState > 0) {
			log.Info.Printf("[%s][%d] %s", k.GetSysinfo.Sysinfo.Alias, i, intToState(k.GetSysinfo.Sysinfo.Children[i].RelayState))
			h.Outlets[i].On.SetValue(k.GetSysinfo.Sysinfo.Children[i].RelayState > 0)
			h.Outlets[i].OutletInUse.SetValue(k.GetSysinfo.Sysinfo.Children[i].RelayState > 0)
		}
	}
}
