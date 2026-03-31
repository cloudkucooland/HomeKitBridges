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
	acc.setID()

	acc.Lightbulb = NewHS220Svc(ip)
	acc.AddS(acc.Lightbulb.S)
	acc.Lightbulb.AddC(acc.generic.StatusActive.C)

	acc.Lightbulb.On.SetValue(k.GetSysinfo.Sysinfo.RelayState > 0)
	acc.Lightbulb.Brightness.SetValue(int(k.GetSysinfo.Sysinfo.Brightness))
	pm := kpm2hpm(k.GetSysinfo.Sysinfo.ActiveMode)
	acc.Lightbulb.ProgramMode.SetValue(pm)

	acc.Lightbulb.On.OnValueRemoteUpdate(func(newstate bool) {
		log.Info.Printf("[%s] %s", acc.Sysinfo.Alias, boolToState(newstate))
		k, _ := newKasaIP(acc.ip)
		if err := k.SetRelayState(newstate); err != nil {
			log.Info.Println(err.Error())
			return
		}
	})

	acc.Lightbulb.Brightness.OnValueRemoteUpdate(func(newstate int) {
		if newstate == 0 {
			return
		}
		log.Info.Printf("[%s] %d%%", acc.Sysinfo.Alias, newstate)
		k, _ := newKasaIP(acc.ip)
		if err := k.SetBrightness(newstate); err != nil {
			log.Info.Println(err.Error())
			return
		}
	})

	acc.Lightbulb.SetDuration.OnValueRemoteUpdate(func(when int) {
		log.Info.Printf("setting duration [%s] to [%d]", acc.Sysinfo.Alias, when)
		if err := setCountdown(acc.ip, !acc.Lightbulb.On.Value(), when); err != nil {
			log.Info.Println(err.Error())
			return
		}
		acc.Lightbulb.ProgramMode.SetValue(characteristic.ProgramModeProgramScheduled)
		acc.Lightbulb.RemainingDuration.SetValue(when)
	})

	acc.Lightbulb.FadeOnTime.OnValueRemoteUpdate(func(when int) {
		log.Info.Printf("setting fade on time [%s] to [%d]", acc.Sysinfo.Alias, when)
		kd, _ := newKasaIP(acc.ip)
		if err := kd.SetFadeOnTime(when); err != nil {
			log.Info.Println(err.Error())
			return
		}
	})

	acc.Lightbulb.FadeOffTime.OnValueRemoteUpdate(func(when int) {
		log.Info.Printf("setting fade off time [%s] to [%d]", acc.Sysinfo.Alias, when)
		kd, _ := newKasaIP(acc.ip)
		if err := kd.SetFadeOffTime(when); err != nil {
			log.Info.Println(err.Error())
			return
		}
	})

	acc.Lightbulb.GentleOnTime.OnValueRemoteUpdate(func(when int) {
		log.Info.Printf("setting gentle on time [%s] to [%d]", acc.Sysinfo.Alias, when)
		kd, _ := newKasaIP(acc.ip)
		if err := kd.SetGentleOnTime(when); err != nil {
			log.Info.Println(err.Error())
			return
		}
	})

	acc.Lightbulb.GentleOffTime.OnValueRemoteUpdate(func(when int) {
		log.Info.Printf("setting gentle off time [%s] to [%d]", acc.Sysinfo.Alias, when)
		kd, _ := newKasaIP(acc.ip)
		if err := kd.SetGentleOffTime(when); err != nil {
			log.Info.Println(err.Error())
			return
		}
	})

	return &acc
}

type HS220Svc struct {
	*service.S

	On         *characteristic.On
	Brightness *characteristic.Brightness

	ProgramMode       *characteristic.ProgramMode
	SetDuration       *characteristic.SetDuration
	RemainingDuration *characteristic.RemainingDuration

	FadeOnTime    *fadeOnTime
	FadeOffTime   *fadeOffTime
	GentleOnTime  *gentleOnTime
	GentleOffTime *gentleOffTime
	RampRate      *rampRate
	MinThreshold  *minThreshold
}

func NewHS220Svc(ip net.IP) *HS220Svc {
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

	svc.FadeOnTime = NewFadeOnTime()
	svc.AddC(svc.FadeOnTime.C)

	svc.FadeOffTime = NewFadeOffTime()
	svc.AddC(svc.FadeOffTime.C)

	svc.GentleOnTime = NewGentleOnTime()
	svc.AddC(svc.GentleOnTime.C)

	svc.GentleOffTime = NewGentleOffTime()
	svc.AddC(svc.GentleOffTime.C)

	svc.RampRate = NewRampRate()
	svc.AddC(svc.RampRate.C)

	svc.MinThreshold = NewMinThreshold()
	svc.AddC(svc.MinThreshold.C)

	svc.S.Primary = true

	getDimmerParametersUDP(ip)
	return &svc
}

func (h *HS220) update(k kasa.KasaDevice, ip net.IP) {
	h.genericUpdate(k, ip)
	d, _ := newKasaIP(ip)

	if h.Lightbulb.On.Value() != (k.GetSysinfo.Sysinfo.RelayState > 0) {
		log.Info.Printf("[%s] %s", k.GetSysinfo.Sysinfo.Alias, intToState(k.GetSysinfo.Sysinfo.RelayState))
		h.Lightbulb.On.SetValue(k.GetSysinfo.Sysinfo.RelayState > 0)
	}

	if h.Lightbulb.Brightness.Value() != int(k.GetSysinfo.Sysinfo.Brightness) {
		log.Info.Printf("[%s] %d%%", k.GetSysinfo.Sysinfo.Alias, int(k.GetSysinfo.Sysinfo.Brightness))
		h.Lightbulb.Brightness.SetValue(int(k.GetSysinfo.Sysinfo.Brightness))
	}

	if h.Lightbulb.ProgramMode.Value() != kpm2hpm(k.GetSysinfo.Sysinfo.ActiveMode) {
		log.Info.Printf("updating HomeKit: [%s] ProgramMode %s", k.GetSysinfo.Sysinfo.Alias, k.GetSysinfo.Sysinfo.ActiveMode)
		h.Lightbulb.ProgramMode.SetValue(kpm2hpm(k.GetSysinfo.Sysinfo.ActiveMode))
		if k.GetSysinfo.Sysinfo.ActiveMode == "none" {
			_ = d.ClearCountdownRules()
		}
	}

	if k.GetSysinfo.Sysinfo.ActiveMode == "count_down" {
		rules, _ := d.GetCountdownRules()
		for _, rule := range rules {
			if rule.Enable > 0 {
				log.Info.Printf("updating HomeKit: [%s] RemainingDuration %d", k.GetSysinfo.Sysinfo.Alias, rule.Remaining)
				h.Lightbulb.RemainingDuration.SetValue(int(rule.Remaining))
			}
		}
	} else {
		h.Lightbulb.RemainingDuration.SetValue(0)
	}

	// almost certainly pointless since these so rarely change, maybe run once a day?
	getDimmerParametersUDP(ip)
}

func (h *HS220) incomingDimmerData(dim kasa.Dimmer) {
	if h.Lightbulb.MinThreshold.Value() != int(dim.Parameters.MinThreshold) {
		log.Info.Printf("updating MinThreshold: [%d] => [%d]", h.Lightbulb.MinThreshold.Value(), dim.Parameters.MinThreshold)
		h.Lightbulb.MinThreshold.SetValue(int(dim.Parameters.MinThreshold))
	}
	if h.Lightbulb.FadeOnTime.Value() != int(dim.Parameters.FadeOnTime) {
		log.Info.Printf("updating FadeOnTime: [%d] => [%d]", h.Lightbulb.FadeOnTime.Value(), dim.Parameters.FadeOnTime)
		h.Lightbulb.FadeOnTime.SetValue(int(dim.Parameters.FadeOnTime))
	}
	if h.Lightbulb.FadeOffTime.Value() != int(dim.Parameters.FadeOffTime) {
		log.Info.Printf("updating FadeOffTime: [%d] => [%d]", h.Lightbulb.FadeOffTime.Value(), dim.Parameters.FadeOffTime)
		h.Lightbulb.FadeOffTime.SetValue(int(dim.Parameters.FadeOffTime))
	}
	if h.Lightbulb.GentleOnTime.Value() != int(dim.Parameters.GentleOnTime) {
		log.Info.Printf("updating GentleOnTime: [%d] => [%d]", h.Lightbulb.GentleOnTime.Value(), dim.Parameters.GentleOnTime)
		h.Lightbulb.GentleOnTime.SetValue(int(dim.Parameters.GentleOnTime))
	}
	if h.Lightbulb.GentleOffTime.Value() != int(dim.Parameters.GentleOffTime) {
		log.Info.Printf("updating GentleOffTime: [%d] => [%d]", h.Lightbulb.GentleOffTime.Value(), dim.Parameters.GentleOffTime)
		h.Lightbulb.GentleOffTime.SetValue(int(dim.Parameters.GentleOffTime))
	}
	if h.Lightbulb.RampRate.Value() != int(dim.Parameters.RampRate) {
		log.Info.Printf("updating RampRate: [%d] => [%d]", h.Lightbulb.RampRate.Value(), dim.Parameters.RampRate)
		h.Lightbulb.RampRate.SetValue(int(dim.Parameters.RampRate))
	}
}
