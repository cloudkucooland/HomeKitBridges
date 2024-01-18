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
	acc.setID()

	acc.Outlet = NewKP115Svc()
	acc.AddS(acc.Outlet.S)

	// set intial state
	acc.Outlet.On.SetValue(k.GetSysinfo.Sysinfo.RelayState > 0)
	acc.Outlet.OutletInUse.SetValue(k.GetSysinfo.Sysinfo.RelayState > 0)
	pm := kpm2hpm(k.GetSysinfo.Sysinfo.ActiveMode)
	acc.Outlet.ProgramMode.SetValue(pm)

	acc.Outlet.On.OnValueRemoteUpdate(func(newstate bool) {
		log.Info.Printf("[%s] %s", acc.Sysinfo.Alias, boolToState(newstate))
		if err := setRelayState(acc.ip, newstate); err != nil {
			log.Info.Println(err.Error())
			return
		}
		acc.Outlet.OutletInUse.SetValue(newstate)
	})

	acc.Outlet.SetDuration.OnValueRemoteUpdate(func(when int) {
		log.Info.Printf("setting duration [%s] to [%d]", acc.Sysinfo.Alias, when)
		if err := setCountdown(acc.ip, !acc.Outlet.On.Value(), when); err != nil {
			log.Info.Println(err.Error())
			return
		}
		acc.Outlet.ProgramMode.SetValue(characteristic.ProgramModeProgramScheduled)
		acc.Outlet.RemainingDuration.SetValue(when)
	})

	return &acc
}

type KP115Svc struct {
	*service.S

	On          *characteristic.On
	OutletInUse *characteristic.OutletInUse

	ProgramMode       *characteristic.ProgramMode
	SetDuration       *characteristic.SetDuration
	RemainingDuration *characteristic.RemainingDuration
	Volt              *volt
	Watt              *watt
	Amp               *amp
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

	svc.Volt = NewVolt()
	svc.AddC(svc.Volt.C)
	svc.Volt.SetValue(120)

	svc.Watt = NewWatt()
	svc.AddC(svc.Watt.C)
	svc.Watt.SetValue(1)

	svc.Amp = NewAmp()
	svc.AddC(svc.Amp.C)
	svc.Amp.SetValue(1)

	svc.S.Primary = true

	return &svc
}

func (h *KP115) update(k kasa.KasaDevice, ip net.IP) {
	h.genericUpdate(k, ip)

	if h.Outlet.On.Value() != (k.GetSysinfo.Sysinfo.RelayState > 0) {
		log.Info.Printf("[%s] %s", k.GetSysinfo.Sysinfo.Alias, intToState(k.GetSysinfo.Sysinfo.RelayState))
		h.Outlet.On.SetValue(k.GetSysinfo.Sysinfo.RelayState > 0)
		h.Outlet.OutletInUse.SetValue(k.GetSysinfo.Sysinfo.RelayState > 0)
	}

	kd, err := kasa.NewDevice(ip.String())
	if err != nil {
		log.Info.Printf(err.Error())
		return
	}

	if h.Outlet.ProgramMode.Value() != kpm2hpm(k.GetSysinfo.Sysinfo.ActiveMode) {
		log.Info.Printf("updating HomeKit: [%s] ProgramMode %s", k.GetSysinfo.Sysinfo.Alias, k.GetSysinfo.Sysinfo.ActiveMode)
		h.Outlet.ProgramMode.SetValue(kpm2hpm(k.GetSysinfo.Sysinfo.ActiveMode))
		if k.GetSysinfo.Sysinfo.ActiveMode == "none" {
			_ = kd.ClearCountdownRules()
		}
	}

	if k.GetSysinfo.Sysinfo.ActiveMode == "count_down" {
		rules, _ := kd.GetCountdownRules()
		for _, rule := range *rules {
			log.Info.Printf("%+v", rule)
			if rule.Enable > 0 {
				log.Info.Printf("updating HomeKit: [%s] RemainingDuration %d", k.GetSysinfo.Sysinfo.Alias, rule.Remaining)
				h.Outlet.RemainingDuration.SetValue(int(rule.Remaining))
			}
		}
	} else {
		h.Outlet.RemainingDuration.SetValue(0)
	}

	// request emeter data over UDP
	if err := getEmeter(h.ip); err != nil {
		return
	}
}

func (h *KP115) updateEmeter(e kasa.EmeterRealtime) {
	if e.Slot > 0 {
		log.Info.Printf("slot out of bounds: %d", e.Slot)
	}

	h.Outlet.Volt.SetValue(int(e.VoltageMV / 1000))
	h.Outlet.Watt.SetValue(int(e.PowerMW / 1000))
	h.Outlet.Amp.SetValue(int(e.CurrentMA))
}
