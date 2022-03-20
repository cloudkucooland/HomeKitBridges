package kasahkbridge

import (
	"net"

	"github.com/brutella/hap/accessory"
	"github.com/brutella/hap/characteristic"
	"github.com/brutella/hap/log"
	"github.com/brutella/hap/service"

	"github.com/cloudkucooland/go-kasa"
)

type HS220 struct {
	*generic

	Lightbulb *HS220Svc
}

func NewHS220(k kasa.KasaDevice, ip net.IP) *HS220 {
	acc := HS220{}
	acc.generic = &generic{}

	info := acc.configure(k.GetSysinfo.Sysinfo, ip)
	acc.A = accessory.New(info, accessory.TypeLightbulb)
	acc.finalize()

	acc.Lightbulb = NewHS220Svc()
	acc.AddS(acc.Lightbulb.S)

	acc.Lightbulb.On.SetValue(k.GetSysinfo.Sysinfo.RelayState > 0)
	acc.Lightbulb.Brightness.SetValue(int(k.GetSysinfo.Sysinfo.Brightness))
	pm := kpm2hpm(k.GetSysinfo.Sysinfo.ActiveMode)
	acc.Lightbulb.ProgramMode.SetValue(pm)

	acc.Lightbulb.On.OnValueRemoteUpdate(func(newstate bool) {
		log.Info.Printf("setting [%s] (%s) to [%t] from HS220 handler", acc.Sysinfo.Alias, acc.ip, newstate)
		if err := setRelayState(acc.ip, newstate); err != nil {
			log.Info.Println(err.Error())
			return
		}
	})

	acc.Lightbulb.Brightness.OnValueRemoteUpdate(func(newstate int) {
		log.Info.Printf("setting brightness [%s] (%s) to [%d] from HS220 handler", acc.Sysinfo.Alias, acc.ip, newstate)
		if err := setBrightness(acc.ip, newstate); err != nil {
			log.Info.Println(err.Error())
			return
		}
	})

	acc.Lightbulb.SetDuration.OnValueRemoteUpdate(func(when int) {
		log.Info.Printf("setting duration [%s] (%s) to [%d] from HS220 handler", acc.Sysinfo.Alias, acc.ip, when)
		if err := setCountdown(acc.ip, !acc.Lightbulb.On.Value(), when); err != nil {
			log.Info.Println(err.Error())
			return
		}
		acc.Lightbulb.ProgramMode.SetValue(characteristic.ProgramModeProgramScheduled)
	})

	acc.Lightbulb.AddC(acc.reachable.C)
	acc.reachable.SetValue(true)

	return &acc
}

type HS220Svc struct {
	*service.S

	On         *characteristic.On
	Brightness *characteristic.Brightness

	ProgramMode       *characteristic.ProgramMode
	SetDuration       *characteristic.SetDuration
	RemainingDuration *characteristic.RemainingDuration
}

func NewHS220Svc() *HS220Svc {
	svc := HS220Svc{}
	svc.S = service.New(service.TypeLightbulb)

	svc.On = characteristic.NewOn()
	svc.AddC(svc.On.C)

	svc.Brightness = characteristic.NewBrightness()
	svc.AddC(svc.Brightness.C)

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

func (h *HS220) update(k kasa.KasaDevice, ip net.IP) {
	h.genericUpdate(k, ip)

	if h.Lightbulb.On.Value() != (k.GetSysinfo.Sysinfo.RelayState > 0) {
		log.Info.Printf("updating HomeKit: [%s]:[%s] relay %d\n", ip.String(), k.GetSysinfo.Sysinfo.Alias, k.GetSysinfo.Sysinfo.RelayState)
		h.Lightbulb.On.SetValue(k.GetSysinfo.Sysinfo.RelayState > 0)
	}

	if h.Lightbulb.Brightness.Value() != int(k.GetSysinfo.Sysinfo.Brightness) {
		log.Info.Printf("updating HomeKit: [%s]:[%s] brightness %d\n", ip.String(), k.GetSysinfo.Sysinfo.Alias, int(k.GetSysinfo.Sysinfo.Brightness))
		h.Lightbulb.Brightness.SetValue(int(k.GetSysinfo.Sysinfo.Brightness))
	}

	if h.Lightbulb.ProgramMode.Value() != kpm2hpm(k.GetSysinfo.Sysinfo.ActiveMode) {
		log.Info.Printf("updating HomeKit: [%s]:[%s] ProgramMode %s\n", ip.String(), k.GetSysinfo.Sysinfo.Alias, k.GetSysinfo.Sysinfo.ActiveMode)
		h.Lightbulb.ProgramMode.SetValue(kpm2hpm(k.GetSysinfo.Sysinfo.ActiveMode))
	}

	if k.GetSysinfo.Sysinfo.ActiveMode == "count_down" {
		d, _ := kasa.NewDevice(h.ip.String())
		rules, _ := d.GetCountdownRules()
		for _, rule := range *rules {
			if rule.Enable > 0 {
				log.Info.Printf("updating HomeKit: [%s]:[%s] RemainingDuration %d\n", ip.String(), k.GetSysinfo.Sysinfo.Alias, rule.Remaining)
				h.Lightbulb.RemainingDuration.SetValue(int(rule.Remaining))
			}
		}
	} else {
		h.Lightbulb.RemainingDuration.SetValue(0)
	}
}
