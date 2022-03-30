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
	ip             string
	password       string
	pins           map[uint8]interface{}
	SecuritySystem *KonnectedSvc
	Buzzer         *KonnectedBuzzer
	Trigger        *KonnectedTrigger
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
			acc.pins[v.Pin] = p
			acc.Buzzer = p
				p.Switch.On.OnValueRemoteUpdate(func(on bool) {
					log.Info.Printf("beeping: %t", on)
                    if !on {
                        return
                    }
					go func() {
						acc.beep()
						time.Sleep(1 * time.Second)
						p.Switch.On.SetValue(!on)
					}()
				})
			log.Info.Printf("Konnected Pin: %d: %s (buzzer)", v.Pin, v.Name)
		case "unused": // not used
		default:
			log.Info.Println("unknown KonnectedZone type: %+v", v)
		}
	}

	// set initial state
	for _, v := range details.Sensors {
		if p, ok := acc.pins[v.Pin]; ok {
			switch p.(type) {
			case *KonnectedContactSensor:
				p.(*KonnectedContactSensor).ContactSensorState.SetValue(int(v.State))
			case *KonnectedMotionSensor:
				p.(*KonnectedMotionSensor).MotionDetected.SetValue(false)
			case *KonnectedBuzzer:
				p.(*KonnectedBuzzer).Switch.On.SetValue(false)
			default:
				log.Info.Println("unknown konnected device type")
			}
		}
	}

	// add handler for setting the system state
	acc.SecuritySystem.SecuritySystemTargetState.OnValueRemoteUpdate(func(newval int) {
		log.Info.Printf("HC requested system state change to %d", newval)
		triggered := true
		if acc.SecuritySystem.SecuritySystemCurrentState.Value() !=
			characteristic.SecuritySystemCurrentStateAlarmTriggered {
			triggered = false
		}
		switch newval {
		case characteristic.SecuritySystemCurrentStateStayArm:
			if triggered {
				log.Info.Println("not changing while in triggered state")
				return
			}
		case characteristic.SecuritySystemCurrentStateAwayArm:
			if triggered {
				log.Info.Println("not changing while in triggered state")
				return
			}
		case characteristic.SecuritySystemCurrentStateNightArm:
			if triggered {
				log.Info.Println("not changing while in triggered state")
				return
			}
		case characteristic.SecuritySystemCurrentStateDisarmed:
			if triggered {
				log.Info.Println("shutting off alarm")
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

	acc.Trigger = NewKonnectedTrigger()
	acc.Trigger.ExternalTrigger.OnValueRemoteUpdate(func(newval bool) {
		log.Info.Println("externally triggered alarm")
	})
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
		Name:         "Buzzer",
		SerialNumber: "0001",
		Manufacturer: "Konnected HKB",
		Model:        "Buzzer",
		Firmware:     "0001",
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

type KonnectedTrigger struct {
	*service.S

	// Name            *characteristic.Name
	ExternalTrigger *etrigger
}

func NewKonnectedTrigger() *KonnectedTrigger {
	s := KonnectedTrigger{}
	s.S = service.New("EF") // custom

	// s.Name = characteristic.NewName()
	// s.Name.SetValue("External Trigger")
	// s.AddC(s.Name.C)

	// allow external events to fire the alarm
	s.ExternalTrigger = NewExternalAlarmTrigger()
	s.ExternalTrigger.Description = "Trigger"
	s.ExternalTrigger.SetValue(false)
	s.AddC(s.ExternalTrigger.C)

	return &s
}

type etrigger struct {
	*characteristic.Bool
}

func NewExternalAlarmTrigger() *etrigger {
	c := characteristic.NewBool("EF1")
	c.Format = characteristic.FormatBool

	c.Permissions = []string{characteristic.PermissionRead, characteristic.PermissionWrite, characteristic.PermissionEvents}

	return &etrigger{c}
}
