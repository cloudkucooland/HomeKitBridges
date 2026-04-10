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

	OutletMap map[string]*hs300outletSvc
}

func NewHS300(k kasa.KasaDevice, ip net.IP) *HS300 {
	acc := HS300{}
	acc.generic = &generic{}

	info := acc.configure(k.GetSysinfo.Sysinfo, ip)
	acc.A = accessory.New(info, accessory.TypeOutlet)
	acc.setID()

	acc.OutletMap = make(map[string]*hs300outletSvc)

	for i := uint(0); i < acc.Sysinfo.NumChildren; i++ {
		idx := i // local scope

		o := newHS300OutletSvc()
		o.slot = idx
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
		acc.OutletMap[acc.Sysinfo.Children[idx].ID] = o
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

	slot uint
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

	for id, outlet := range h.OutletMap {
		data, err := getChildFromID(k, id)
		if err != nil {
			log.Info.Println(err.Error())
			continue
		}

		if outlet.On.Value() != (data.RelayState > 0) {
			log.Info.Printf("[%s][%s] %s", k.GetSysinfo.Sysinfo.Alias, id, intToState(data.RelayState))
			outlet.On.SetValue(data.RelayState > 0)
			outlet.OutletInUse.SetValue(data.RelayState > 0)
		}

		if outlet.Name.Value() != data.Alias {
			log.Info.Printf("updating HomeKit: [%s][%s] name %s", k.GetSysinfo.Sysinfo.Alias, id, data.Alias)
			outlet.Name.SetValue(data.Alias)
		}

		// request emeter data for each outlet
		if err := getEmeterChildUDP(h.ip, h.Sysinfo.DeviceID, id); err != nil {
			log.Info.Println(err.Error())
		}
	}
}

func getChildFromID(k kasa.KasaDevice, id string) (*kasa.Child, error) {
	for _, j := range k.GetSysinfo.Sysinfo.Children {
		if j.ID == id {
			return &j, nil
		}
	}
	return nil, fmt.Errorf("child not found")
}

func (h *HS300) getOutletFromSlot(slot uint) (*hs300outletSvc, error) {
	for _, j := range h.OutletMap {
		if j.slot == slot {
			return j, nil
		}
	}
	return nil, fmt.Errorf("child not found")
}

func (h *HS300) incomingEmeterData(e kasa.EmeterRealtime) {
	if int(e.Slot) >= len(h.OutletMap) {
		log.Info.Printf("slot out of bounds: %d", e.Slot)
		return
	}

	child, err := h.getOutletFromSlot(e.Slot)
	if err != nil {
		log.Info.Println(err.Error())
		return
	}

	v := int(e.VoltageMV / 1000)
	child.Volt.SetValue(v)
	switch {
	case v > 130:
		log.Info.Printf("[%s][%s] dangerously high voltage: %dV", h.Sysinfo.Alias, child.Name.Value(), v)
		child.StatusFault.SetValue(characteristic.StatusFaultGeneralFault)

		full := fmt.Sprintf("%s%s", h.Sysinfo.DeviceID, h.Sysinfo.Children[e.Slot].ID)
		k, _ := newKasaIP(h.ip)
		if err := k.SetRelayStateChild(full, false); err != nil {
			log.Info.Println(err.Error())
			return
		}
		child.On.SetValue(false)
		child.OutletInUse.SetValue(false)
	case v > 127:
		log.Info.Printf("[%s][%s] high voltage: %dV", h.Sysinfo.Alias, child.Name.Value(), v)
		if child.StatusFault.Value() == characteristic.StatusFaultGeneralFault {
			child.StatusFault.SetValue(characteristic.StatusFaultNoFault)
		}
	case v < 114:
		log.Info.Printf("[%s][%s] low voltage: %dV", h.Sysinfo.Alias, child.Name.Value(), v)
		if child.StatusFault.Value() == characteristic.StatusFaultGeneralFault {
			child.StatusFault.SetValue(characteristic.StatusFaultNoFault)
		}
	case v < 110:
		log.Info.Printf("[%s][%s] dangerously low voltage: %dV", h.Sysinfo.Alias, child.Name.Value(), v)
		child.StatusFault.SetValue(characteristic.StatusFaultGeneralFault)

		full := fmt.Sprintf("%s%s", h.Sysinfo.DeviceID, h.Sysinfo.Children[e.Slot].ID)
		k, _ := newKasaIP(h.ip)
		if err := k.SetRelayStateChild(full, false); err != nil {
			log.Info.Println(err.Error())
			return
		}
		child.On.SetValue(false)
		child.OutletInUse.SetValue(false)
	}

	child.Watt.SetValue(int(e.PowerMW / 1000))
	child.Amp.SetValue(int(e.CurrentMA))
}
