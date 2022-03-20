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

	return &svc
}

func (h *KP115) update(k kasa.KasaDevice, ip net.IP) {
	h.genericUpdate(k, ip)

	if h.Outlet.On.Value() != (k.GetSysinfo.Sysinfo.RelayState > 0) {
		log.Info.Printf("updating HomeKit: [%s]:[%s] relay %d\n", ip.String(), k.GetSysinfo.Sysinfo.Alias, k.GetSysinfo.Sysinfo.RelayState)
		h.Outlet.On.SetValue(k.GetSysinfo.Sysinfo.RelayState > 0)
		h.Outlet.OutletInUse.SetValue(k.GetSysinfo.Sysinfo.RelayState > 0)
	}

	kd, err := kasa.NewDevice(ip.String())
	if err != nil {
		log.Info.Printf(err.Error())
		return
	}

	if h.Outlet.ProgramMode.Value() != kpm2hpm(k.GetSysinfo.Sysinfo.ActiveMode) {
		log.Info.Printf("updating HomeKit: [%s]:[%s] ProgramMode %s\n", ip.String(), k.GetSysinfo.Sysinfo.Alias, k.GetSysinfo.Sysinfo.ActiveMode)
		h.Outlet.ProgramMode.SetValue(kpm2hpm(k.GetSysinfo.Sysinfo.ActiveMode))
		if k.GetSysinfo.Sysinfo.ActiveMode == "none" {
			_ = kd.ClearCountdownRules()
		}
	}

	em, err := kd.GetEmeter()
	if err != nil {
		log.Info.Printf(err.Error())
		return
	}
	h.Outlet.Volt.SetValue(int(em.VoltageMV / 1000))
	h.Outlet.Watt.SetValue(int(em.PowerMW / 1000))
	h.Outlet.Amp.SetValue(int(em.CurrentMA))

	if k.GetSysinfo.Sysinfo.ActiveMode == "count_down" {
		rules, _ := kd.GetCountdownRules()
		for _, rule := range *rules {
			log.Info.Printf("%+v", rule)
			if rule.Enable > 0 {
				log.Info.Printf("updating HomeKit: [%s]:[%s] RemainingDuration %d\n", ip.String(), k.GetSysinfo.Sysinfo.Alias, rule.Remaining)
				h.Outlet.RemainingDuration.SetValue(int(rule.Remaining))
			}
		}
	} else {
		h.Outlet.RemainingDuration.SetValue(0)
	}
}

// move this to custom characteristic file
type volt struct {
	*characteristic.Int
}

func NewVolt() *volt {
	c := characteristic.NewInt("10A")
	c.Format = characteristic.FormatUInt32
	c.Permissions = []string{characteristic.PermissionRead, characteristic.PermissionEvents}
	c.Description = "Voltage"
	c.SetValue(120)

	return &volt{c}
}

type watt struct {
	*characteristic.Int
}

func NewWatt() *watt {
	c := characteristic.NewInt("10D")
	c.Format = characteristic.FormatUInt32
	c.Permissions = []string{characteristic.PermissionRead, characteristic.PermissionEvents}
	c.Description = "Watts"
	c.SetValue(0)

	return &watt{c}
}

type amp struct {
	*characteristic.Int
}

func NewAmp() *amp {
	c := characteristic.NewInt("10B") // 126
	c.Format = characteristic.FormatUInt32
	c.Permissions = []string{characteristic.PermissionRead, characteristic.PermissionEvents}
	c.Description = "Current MA"
	c.SetValue(0)

	return &amp{c}
}

// https://github.com/plasticrake/homebridge-tplink-smarthome/blob/master/src/characteristics/amperes.ts
// volt E863F10A-079E-48FF-8F27-9C2605A29F52
// amps E863F126-079E-48FF-8F27-9C2605A29F52
// watt E863F10D-079E-48FF-8F27-9C2605A29F52
// kwh  E863F10C-079E-48FF-8F27-9C2605A29F52
