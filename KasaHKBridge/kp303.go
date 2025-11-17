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

	Outlets []*kp303outletSvc
}

func NewKP303(k kasa.KasaDevice, ip net.IP) *KP303 {
	acc := KP303{}
	acc.generic = &generic{}

	info := acc.configure(k.GetSysinfo.Sysinfo, ip)
	acc.A = accessory.New(info, accessory.TypeOutlet)
	acc.setID()

	os := int(acc.Sysinfo.NumChildren)
	acc.Outlets = make([]*kp303outletSvc, os, os+1)

	for i := 0; i < os; i++ {
		idx := i // force local scope - especially for handler

		o := newkp303OutletSvc()
		o.On.SetValue(acc.Sysinfo.Children[idx].RelayState > 0)
		o.OutletInUse.SetValue(acc.Sysinfo.Children[idx].RelayState > 0)

		o.Name.SetValue(acc.Sysinfo.Children[idx].Alias)
		id := fmt.Sprintf("%s%s", acc.Sysinfo.DeviceID[32:], acc.Sysinfo.Children[idx].ID)
		o.AccIdentifier.SetValue(id)
		o.ID.SetValue(idx)
		if dx, err := strconv.ParseInt(id, 16, 64); err != nil {
			log.Info.Println(err.Error())
		} else {
			o.ID.SetValue(int(dx))
		}

		o.On.OnValueRemoteUpdate(func(newstate bool) {
			log.Info.Printf("[%s][%d] %s", acc.Sysinfo.Alias, idx, boolToState(newstate))
			if err := setChildRelayState(acc.ip, acc.Sysinfo.DeviceID, acc.Sysinfo.Children[idx].ID, newstate); err != nil {
				log.Info.Println(err.Error())
				return
			}
			o.OutletInUse.SetValue(newstate)
		})

		o.Name.OnValueRemoteUpdate(func(newname string) {
			log.Info.Printf("[%s][%d] new name %s", acc.Sysinfo.Alias, idx, newname)
			/* if err := setChildRelayName(acc.ip, acc.Sysinfo.DeviceID, acc.Sysinfo.Children[idx].ID, newstate); err != nil {
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

		if h.Outlets[i].Name.Value() != k.GetSysinfo.Sysinfo.Children[i].Alias {
			log.Info.Printf("updating HomeKit: [%s][%d] name %s", k.GetSysinfo.Sysinfo.Alias, i, k.GetSysinfo.Sysinfo.Children[i].Alias)
			h.Outlets[i].Name.SetValue(k.GetSysinfo.Sysinfo.Children[i].Alias)
		}
	}
}

type kp303outletSvc struct {
	*service.S

	On            *characteristic.On
	OutletInUse   *characteristic.OutletInUse
	Name          *characteristic.Name
	ID            *characteristic.Identifier
	AccIdentifier *characteristic.AccessoryIdentifier
}

func newkp303OutletSvc() *kp303outletSvc {
	svc := kp303outletSvc{}
	svc.S = service.New(service.TypeOutlet)

	svc.On = characteristic.NewOn()
	svc.AddC(svc.On.C)

	svc.OutletInUse = characteristic.NewOutletInUse()
	svc.AddC(svc.OutletInUse.C)

	svc.Name = characteristic.NewName()
	svc.Name.Permissions = []string{characteristic.PermissionRead, characteristic.PermissionWrite}
	svc.AddC(svc.Name.C)

	svc.ID = characteristic.NewIdentifier()
	svc.AddC(svc.ID.C)
	svc.ID.SetValue(0)

	svc.AccIdentifier = characteristic.NewAccessoryIdentifier()
	svc.AddC(svc.AccIdentifier.C)
	svc.AccIdentifier.SetValue("0")

	return &svc
}
