package kasahkbridge

import (
	"net"

	"github.com/brutella/hap/accessory"
	"github.com/brutella/hap/characteristic"
	"github.com/brutella/hap/log"
	"github.com/brutella/hap/service"

	"github.com/cloudkucooland/go-kasa"
)

// KP115 is a single outlet with energy monitoring
type KP115 struct {
	*generic

	Outlet *KP115Svc
}

func NewKP115(k kasa.KasaDevice, ip net.IP) *KP115 {
	acc := KP115{}
	acc.generic = &generic{}

	info := acc.configure(k.GetSysinfo.Sysinfo, ip)
	acc.A = accessory.New(info, accessory.TypeOutlet)
	acc.finalize()

	acc.Outlet = NewKP115Svc()
	acc.AddS(acc.Outlet.S)

	// set intial state
	acc.Outlet.On.SetValue(k.GetSysinfo.Sysinfo.RelayState > 0)
	acc.Outlet.OutletInUse.SetValue(k.GetSysinfo.Sysinfo.RelayState > 0)
	pm := kpm2hpm(k.GetSysinfo.Sysinfo.ActiveMode)
	acc.Outlet.ProgramMode.SetValue(pm)

	acc.Outlet.On.OnValueRemoteUpdate(func(newstate bool) {
		log.Info.Printf("setting [%s] to [%t] from KP115 handler", acc.Sysinfo.Alias, newstate)
		if err := setRelayState(acc.ip, newstate); err != nil {
			log.Info.Println(err.Error())
			return
		}
		acc.Outlet.OutletInUse.SetValue(newstate)
	})

	acc.Outlet.SetDuration.OnValueRemoteUpdate(func(when int) {
		log.Info.Printf("setting duration [%s] (%s) to [%d] from HS220 handler", acc.Sysinfo.Alias, acc.ip, when)
		if err := setCountdown(acc.ip, !acc.Outlet.On.Value(), when); err != nil {
			log.Info.Println(err.Error())
			return
		}
		acc.Outlet.ProgramMode.SetValue(characteristic.ProgramModeProgramScheduled)
	})

	acc.Outlet.AddC(acc.reachable.C)
	acc.reachable.SetValue(true)

	return &acc
}

type KP115Svc struct {
	*service.S

	On          *characteristic.On
	OutletInUse *characteristic.OutletInUse

	ProgramMode       *characteristic.ProgramMode
	SetDuration       *characteristic.SetDuration
	RemainingDuration *characteristic.RemainingDuration
}

func NewKP115Svc() *KP115Svc {
	svc := KP115Svc{}
	svc.S = service.New(service.TypeOutlet)

	svc.On = characteristic.NewOn()
	svc.AddC(svc.On.C)

	svc.OutletInUse = characteristic.NewOutletInUse()
	svc.AddC(svc.OutletInUse.C)

	svc.ProgramMode = characteristic.NewProgramMode()
	svc.AddC(svc.ProgramMode.C)
	svc.ProgramMode.SetValue(characteristic.ProgramModeNoProgramScheduled)

	svc.SetDuration = characteristic.NewSetDuration()
	svc.AddC(svc.SetDuration.C)

	svc.RemainingDuration = characteristic.NewRemainingDuration()
	svc.AddC(svc.RemainingDuration.C)
	svc.RemainingDuration.SetValue(0)

	return &svc
}

func (h *KP115) update(k kasa.KasaDevice, ip net.IP) {
	h.genericUpdate(k, ip)

	if h.Outlet.On.Value() != (k.GetSysinfo.Sysinfo.RelayState > 0) {
		log.Info.Printf("updating HomeKit: [%s]:[%s] relay %d\n", ip.String(), k.GetSysinfo.Sysinfo.Alias, k.GetSysinfo.Sysinfo.RelayState)
		h.Outlet.On.SetValue(k.GetSysinfo.Sysinfo.RelayState > 0)
		h.Outlet.OutletInUse.SetValue(k.GetSysinfo.Sysinfo.RelayState > 0)
	}

	if h.Outlet.ProgramMode.Value() != kpm2hpm(k.GetSysinfo.Sysinfo.ActiveMode) {
		log.Info.Printf("updating HomeKit: [%s]:[%s] ProgramMode %s\n", ip.String(), k.GetSysinfo.Sysinfo.Alias, k.GetSysinfo.Sysinfo.ActiveMode)
		h.Outlet.ProgramMode.SetValue(kpm2hpm(k.GetSysinfo.Sysinfo.ActiveMode))
	}

	// SetDuration is write-only, no need to update it here
	// Process RemainingDuration ?
}
