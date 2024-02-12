package konnectedkhbridge

import (
	"fmt"
	"time"

	"github.com/brutella/hap/accessory"
	"github.com/brutella/hap/characteristic"
	"github.com/brutella/hap/log"
	"github.com/brutella/hap/service"
)

type Konnected struct {
	*accessory.A
	pins           map[uint8]interface{}
	SecuritySystem *KonnectedSvc
	Buzzer         *KonnectedBuzzer
	// Trigger        *KonnectedTrigger
	ip       string
	password string
}

func NewKonnected(details *system, d *Device) *Konnected {
	acc := Konnected{}

	info := accessory.Info{
		Name:         "konnected", // Name,
		SerialNumber: details.Mac,
		Manufacturer: "Konnected.io",
		Model:        details.Hardware,
		Firmware:     details.Software,
	}

	acc.A = accessory.New(info, accessory.TypeSecuritySystem)
	acc.ip = fmt.Sprintf("%s:%d", details.IP, details.Port)
	acc.password = d.Password

	acc.SecuritySystem = NewKonnectedSvc()
	acc.AddS(acc.SecuritySystem.S)
	acc.SecuritySystem.SecuritySystemCurrentState.SetValue(3) // default to Off
	acc.SecuritySystem.SecuritySystemTargetState.SetValue(3)  // default to Off

	alarmType := characteristic.NewSecuritySystemAlarmType()
	alarmType.SetValue(1)
	acc.SecuritySystem.AddC(alarmType.C)

	// convert zones from config to pins
	acc.pins = make(map[uint8]interface{})
	for _, v := range d.Zones {
		switch v.Type {
		case "motion":
			p := NewKonnectedMotionSensor(v.Name)
			acc.pins[v.Pin] = p
			acc.A.AddS(p.S)
			log.Info.Printf("Konnected Pin: %d: %s (motion)", v.Pin, v.Name)
		case "door":
			p := NewKonnectedContactSensor(v.Name)
			acc.pins[v.Pin] = p
			acc.A.AddS(p.S)
			log.Info.Printf("Konnected Pin: %d: %s (contact)", v.Pin, v.Name)
		case "buzzer":
			p := NewKonnectedBuzzer(v.Name)
			p.Switch.On.OnValueRemoteUpdate(func(on bool) {
				log.Info.Printf("beeping: %t", on)
				if !on {
					return
				}
				go func() {
					acc.beep()
					time.Sleep(5 * time.Second)
					p.Switch.On.SetValue(false)
				}()
			})
			acc.pins[v.Pin] = p
			acc.Buzzer = p
			log.Info.Printf("Konnected Pin: %d: %s (buzzer)", v.Pin, v.Name)
		case "unused": // not used
		default:
			log.Info.Printf("unknown KonnectedZone type: %+v", v)
		}
	}

	// set initial state
	for _, v := range details.Sensors {
		if p, ok := acc.pins[v.Pin]; ok {
			switch p := p.(type) {
			case *KonnectedContactSensor:
				p.ContactSensorState.SetValue(int(v.State))
			case *KonnectedMotionSensor:
				p.MotionDetected.SetValue(false)
			case *KonnectedBuzzer:
				// p.Switch.On.SetValue(false)
			default:
				log.Info.Println("unknown konnected device type")
			}
		}
	}

	// add handler for setting the system state
	acc.SecuritySystem.SecuritySystemTargetState.OnValueRemoteUpdate(func(newval int) {
		triggered := true
		if acc.SecuritySystem.SecuritySystemCurrentState.Value() !=
			characteristic.SecuritySystemCurrentStateAlarmTriggered {
			triggered = false
		}
		switch newval {
		case characteristic.SecuritySystemCurrentStateStayArm:
			log.Info.Println("system state changed to 'stay'")
			if triggered {
				log.Info.Println("not changing while triggered")
				return
			}
		case characteristic.SecuritySystemCurrentStateAwayArm:
			log.Info.Println("system state changed to 'away'")
			if triggered {
				log.Info.Println("not changing while triggered")
				return
			}
		case characteristic.SecuritySystemCurrentStateNightArm:
			log.Info.Println("system state changed to 'night'")
			if triggered {
				log.Info.Println("not changing while triggered")
				return
			}
		case characteristic.SecuritySystemCurrentStateDisarmed:
			log.Info.Println("system state changed to 'disarmed'")
			if triggered {
				log.Info.Println("stopping alarm")
				acc.cancelAlarm()
			}
		default:
			log.Info.Printf("unknown security system state: %d", newval)
			return
		}
		acc.SecuritySystem.SecuritySystemCurrentState.SetValue(newval)
		// let a triggered alarm continue to ring until cancelAlarm takes care of it
		if !triggered {
			acc.beep()
		}
	})

	/* acc.Trigger = NewKonnectedTrigger()
	acc.Trigger.Trigger.Trip.OnValueRemoteUpdate(func(targetState bool) {
		if targetState {
			log.Info.Println("externally triggered alarm")
			switch acc.SecuritySystem.SecuritySystemCurrentState.Value() {
			case characteristic.SecuritySystemCurrentStateAwayArm:
				acc.countdownAlarm()
			case characteristic.SecuritySystemCurrentStateNightArm,
				characteristic.SecuritySystemCurrentStateStayArm:
				acc.instantAlarm()
			default:
				acc.instantAlarm()
			}

		} else {
			triggered := true
			if acc.SecuritySystem.SecuritySystemCurrentState.Value() !=
				characteristic.SecuritySystemCurrentStateAlarmTriggered {
				triggered = false
			}
			if triggered {
				log.Info.Println("clearing external alarm")
				acc.cancelAlarm()
			} else {
				log.Info.Println("not triggered, nothing to clear")
			}
		}
	}) */
	return &acc
}

type KonnectedSvc struct {
	*service.S

	SecuritySystemCurrentState *characteristic.SecuritySystemCurrentState
	SecuritySystemTargetState  *characteristic.SecuritySystemTargetState
}

func NewKonnectedSvc() *KonnectedSvc {
	s := KonnectedSvc{}
	s.S = service.New(service.TypeSecuritySystem)

	s.SecuritySystemCurrentState = characteristic.NewSecuritySystemCurrentState()
	s.AddC(s.SecuritySystemCurrentState.C)

	s.SecuritySystemTargetState = characteristic.NewSecuritySystemTargetState()
	s.AddC(s.SecuritySystemTargetState.C)

	return &s
}

type KonnectedContactSensor struct {
	*service.S

	ContactSensorState *characteristic.ContactSensorState
	Name               *characteristic.Name
}

func NewKonnectedContactSensor(name string) *KonnectedContactSensor {
	s := KonnectedContactSensor{}
	s.S = service.New(service.TypeContactSensor)

	s.ContactSensorState = characteristic.NewContactSensorState()
	s.AddC(s.ContactSensorState.C)

	s.Name = characteristic.NewName()
	s.Name.SetValue(name)
	s.AddC(s.Name.C)

	return &s
}

type KonnectedMotionSensor struct {
	*service.S

	MotionDetected *characteristic.MotionDetected
	Name           *characteristic.Name
}

func NewKonnectedMotionSensor(name string) *KonnectedMotionSensor {
	s := KonnectedMotionSensor{}
	s.S = service.New(service.TypeMotionSensor)

	s.MotionDetected = characteristic.NewMotionDetected()
	s.AddC(s.MotionDetected.C)

	s.Name = characteristic.NewName()
	s.Name.SetValue(name)
	s.AddC(s.Name.C)

	return &s
}

type KonnectedBuzzer struct {
	*accessory.A
	Switch *KonnectedBuzzerSvc
}

func NewKonnectedBuzzer(name string) *KonnectedBuzzer {
	a := KonnectedBuzzer{}
	a.A = accessory.New(accessory.Info{
		Name:         name,
		SerialNumber: "0002",
		Manufacturer: "Konnected HKB",
		Model:        "Buzzer",
		Firmware:     "0002",
	}, accessory.TypeSwitch)
	a.Switch = NewKonnectedBuzzerSvc(name)
	a.AddS(a.Switch.S)

	return &a
}

type KonnectedBuzzerSvc struct {
	*service.S

	On   *characteristic.On
	Name *characteristic.Name
}

func NewKonnectedBuzzerSvc(name string) *KonnectedBuzzerSvc {
	s := KonnectedBuzzerSvc{}
	s.S = service.New(service.TypeSwitch)

	s.On = characteristic.NewOn()
	s.AddC(s.On.C)

	s.Name = characteristic.NewName()
	s.Name.SetValue(name)
	s.AddC(s.Name.C)

	return &s
}

/* type KonnectedTrigger struct {
	*accessory.A

	Trigger *KonnectedTriggerSvc
}

func NewKonnectedTrigger() *KonnectedTrigger {
	a := KonnectedTrigger{}

	a.A = accessory.New(accessory.Info{
		Name:         "Trigger",
		SerialNumber: "0003",
		Manufacturer: "Konnected HKB",
		Model:        "Trigger",
		Firmware:     "0003",
	}, accessory.TypeSwitch)

	a.Trigger = NewKonnectedTriggerSvc()
	a.AddS(a.Trigger.S)

	return &a
}

type KonnectedTriggerSvc struct {
	*service.S

	Trip *characteristic.On
}

func NewKonnectedTriggerSvc() *KonnectedTriggerSvc {
	s := KonnectedTriggerSvc{}
	s.S = service.New(service.TypeSwitch)

	s.Trip = characteristic.NewOn()
	s.AddC(s.Trip.C)

	return &s
} */
