package kasahkbridge

import (
	"net"

	"github.com/brutella/hap/accessory"
	"github.com/brutella/hap/characteristic"
	"github.com/brutella/hap/log"
	"github.com/brutella/hap/service"

	"github.com/cloudkucooland/go-kasa"
)

// 200 and 210 are the same
type HS200 struct {
	*generic

	Switch *HS200Svc
}

func NewHS200(k kasa.KasaDevice, ip net.IP) *HS200 {
	acc := HS200{}
	acc.generic = &generic{}

	info := acc.configure(k.GetSysinfo.Sysinfo, ip)
	acc.A = accessory.New(info, accessory.TypeSwitch)
	acc.finalize()

	acc.Switch = NewHS200Svc()
	acc.AddS(acc.Switch.S)

	acc.Switch.On.SetValue(k.GetSysinfo.Sysinfo.RelayState > 0)
	pm := kpm2hpm(k.GetSysinfo.Sysinfo.ActiveMode)
	acc.Switch.ProgramMode.SetValue(pm)

	acc.Switch.On.OnValueRemoteUpdate(func(newstate bool) {
		log.Info.Printf("setting [%s] to [%t] from HS200 handler", acc.Sysinfo.Alias, newstate)
		if err := setRelayState(acc.ip, newstate); err != nil {
			log.Info.Println(err.Error())
			return
		}
	})

	acc.Switch.SetDuration.OnValueRemoteUpdate(func(when int) {
		log.Info.Printf("setting duration [%s] (%s) to [%d] from HS220 handler", acc.Sysinfo.Alias, acc.ip, when)
		if err := setCountdown(acc.ip, !acc.Switch.On.Value(), when); err != nil {
			log.Info.Println(err.Error())
			return
		}
		acc.Switch.ProgramMode.SetValue(characteristic.ProgramModeProgramScheduled)
	})

	acc.Switch.AddC(acc.reachable.C)
	acc.reachable.SetValue(true)

	return &acc
}

type HS200Svc struct {
	*service.S

	On *characteristic.On

	ProgramMode       *characteristic.ProgramMode
	SetDuration       *characteristic.SetDuration
	RemainingDuration *characteristic.RemainingDuration
}

func NewHS200Svc() *HS200Svc {
	svc := HS200Svc{}
	svc.S = service.New(service.TypeSwitch)

	svc.On = characteristic.NewOn()
	svc.AddC(svc.On.C)

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

func (h *HS200) update(k kasa.KasaDevice, ip net.IP) {
	h.genericUpdate(k, ip)

	if h.Switch.On.Value() != (k.GetSysinfo.Sysinfo.RelayState > 0) {
		log.Info.Printf("updating HomeKit: [%s]:[%s] relay %d\n", ip.String(), k.GetSysinfo.Sysinfo.Alias, k.GetSysinfo.Sysinfo.RelayState)
		h.Switch.On.SetValue(k.GetSysinfo.Sysinfo.RelayState > 0)
	}

	if h.Switch.ProgramMode.Value() != kpm2hpm(k.GetSysinfo.Sysinfo.ActiveMode) {
		log.Info.Printf("updating HomeKit: [%s]:[%s] ProgramMode %s\n", ip.String(), k.GetSysinfo.Sysinfo.Alias, k.GetSysinfo.Sysinfo.ActiveMode)
		h.Switch.ProgramMode.SetValue(kpm2hpm(k.GetSysinfo.Sysinfo.ActiveMode))
	}

	// SetDuration is write-only, no need to update it here
	// Process RemainingDuration ?
}
