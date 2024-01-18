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

	acc.Lightbulb.On.SetValue(k.GetSysinfo.Sysinfo.RelayState > 0)
	acc.Lightbulb.Brightness.SetValue(int(k.GetSysinfo.Sysinfo.Brightness))
	pm := kpm2hpm(k.GetSysinfo.Sysinfo.ActiveMode)
	acc.Lightbulb.ProgramMode.SetValue(pm)

	acc.Lightbulb.On.OnValueRemoteUpdate(func(newstate bool) {
		log.Info.Printf("[%s] %s", acc.Sysinfo.Alias, boolToState(newstate))
		if err := setRelayState(acc.ip, newstate); err != nil {
			log.Info.Println(err.Error())
			return
		}
	})

	acc.Lightbulb.Brightness.OnValueRemoteUpdate(func(newstate int) {
		if newstate == 0 {
			/* log.Info.Printf("[%s] %s", acc.Sysinfo.Alias, boolToState(false))
					    if err := setRelayState(acc.ip, false); err != nil {
						    log.Info.Println(err.Error())
			            } */
			return
		}
		log.Info.Printf("[%s] %d%%", acc.Sysinfo.Alias, newstate)
		if err := setBrightness(acc.ip, newstate); err != nil {
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
		kd, _ := kasa.NewDevice(acc.ip.String())
		if err := kd.SetFadeOnTime(when); err != nil {
			log.Info.Println(err.Error())
			return
		}
	})

	acc.Lightbulb.FadeOffTime.OnValueRemoteUpdate(func(when int) {
		log.Info.Printf("setting fade off time [%s] to [%d]", acc.Sysinfo.Alias, when)
		kd, _ := kasa.NewDevice(acc.ip.String())
		if err := kd.SetFadeOffTime(when); err != nil {
			log.Info.Println(err.Error())
			return
		}
	})

	acc.Lightbulb.GentleOnTime.OnValueRemoteUpdate(func(when int) {
		log.Info.Printf("setting gentle on time [%s] to [%d]", acc.Sysinfo.Alias, when)
		kd, _ := kasa.NewDevice(acc.ip.String())
		if err := kd.SetGentleOnTime(when); err != nil {
			log.Info.Println(err.Error())
			return
		}
	})

	acc.Lightbulb.GentleOffTime.OnValueRemoteUpdate(func(when int) {
		log.Info.Printf("setting gentle off time [%s] to [%d]", acc.Sysinfo.Alias, when)
		kd, _ := kasa.NewDevice(acc.ip.String())
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
	svc.AddC(svc.RampRate.C) // nope

	svc.MinThreshold = NewMinThreshold()
	svc.AddC(svc.MinThreshold.C) // nope

	k, _ := kasa.NewDevice(ip.String())
	dimmer, err := k.GetDimmerParameters()
	if err != nil {
		return &svc
	}

	svc.FadeOnTime.SetValue(int(dimmer.FadeOnTime))
	svc.FadeOffTime.SetValue(int(dimmer.FadeOffTime))
	svc.GentleOnTime.SetValue(int(dimmer.GentleOnTime))
	svc.GentleOffTime.SetValue(int(dimmer.GentleOffTime))
	svc.RampRate.SetValue(int(dimmer.RampRate))
	svc.MinThreshold.SetValue(int(dimmer.MinThreshold))

	svc.S.Primary = true

	return &svc
}

func (h *HS220) update(k kasa.KasaDevice, ip net.IP) {
	h.genericUpdate(k, ip)
	d, _ := kasa.NewDevice(h.ip.String())

	if h.Lightbulb.On.Value() != (k.GetSysinfo.Sysinfo.RelayState > 0) {
		log.Info.Printf("[%s] %s", k.GetSysinfo.Sysinfo.Alias, intToState(k.GetSysinfo.Sysinfo.RelayState))
		h.Lightbulb.On.SetValue(k.GetSysinfo.Sysinfo.RelayState > 0)
	}

	if h.Lightbulb.Brightness.Value() != int(k.GetSysinfo.Sysinfo.Brightness) {
		log.Info.Printf("updating HomeKit: [%s] brightness %d", k.GetSysinfo.Sysinfo.Alias, int(k.GetSysinfo.Sysinfo.Brightness))
		h.Lightbulb.Brightness.SetValue(int(k.GetSysinfo.Sysinfo.Brightness))
	}

	if h.Lightbulb.ProgramMode.Value() != kpm2hpm(k.GetSysinfo.Sysinfo.ActiveMode) {
		log.Info.Printf("updating HomeKit: [%s] ProgramMode %s", k.GetSysinfo.Sysinfo.Alias, k.GetSysinfo.Sysinfo.ActiveMode)
		h.Lightbulb.ProgramMode.SetValue(kpm2hpm(k.GetSysinfo.Sysinfo.ActiveMode))
		if k.GetSysinfo.Sysinfo.ActiveMode == "none" {
			// d, _ := kasa.NewDevice(h.ip.String())
			_ = d.ClearCountdownRules()
		}
	}

	if k.GetSysinfo.Sysinfo.ActiveMode == "count_down" {
		rules, _ := d.GetCountdownRules()
		for _, rule := range *rules {
			if rule.Enable > 0 {
				log.Info.Printf("updating HomeKit: [%s] RemainingDuration %d", k.GetSysinfo.Sysinfo.Alias, rule.Remaining)
				h.Lightbulb.RemainingDuration.SetValue(int(rule.Remaining))
			}
		}
	} else {
		h.Lightbulb.RemainingDuration.SetValue(0)
	}

	// request dimmer parameters on broadcast
}

// TODO add update on dimmer parameters
