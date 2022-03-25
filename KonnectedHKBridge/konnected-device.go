package konnectedkhbridge

import (
	"fmt"

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
			acc.A.AddS(p.S)
			log.Info.Printf("Konnected Pin: %d: %s (buzzer)", v.Pin, v.Name)
			p.Beeper.OnValueRemoteUpdate(func(on bool) {
				log.Info.Printf("beeping: %t", on)
				// doBeep()
			})
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
				p.(*KonnectedMotionSensor).MotionDetected.SetValue(v.State == 1)
			case *KonnectedBuzzer:
				// p.(*KonnectedBuzzer).Beeper.SetValue(v.State == 1)
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

	return &acc
}

type KonnectedSvc struct {
	*service.S

	SecuritySystemCurrentState *characteristic.SecuritySystemCurrentState
	SecuritySystemTargetState  *characteristic.SecuritySystemTargetState
}

func NewKonnectedSvc() *KonnectedSvc {
	svc := KonnectedSvc{}
	svc.S = service.New(service.TypeSecuritySystem)

	svc.SecuritySystemCurrentState = characteristic.NewSecuritySystemCurrentState()
	svc.AddC(svc.SecuritySystemCurrentState.C)

	svc.SecuritySystemTargetState = characteristic.NewSecuritySystemTargetState()
	svc.AddC(svc.SecuritySystemTargetState.C)

	return &svc
}

type KonnectedContactSensor struct {
	*service.S

	ContactSensorState *characteristic.ContactSensorState
	Name               *characteristic.Name
}

func NewKonnectedContactSensor(name string) *KonnectedContactSensor {
	svc := KonnectedContactSensor{}
	svc.S = service.New(service.TypeContactSensor)

	svc.ContactSensorState = characteristic.NewContactSensorState()
	svc.AddC(svc.ContactSensorState.C)

	svc.Name = characteristic.NewName()
	svc.Name.SetValue(name)
	svc.AddC(svc.Name.C)

	return &svc
}

type KonnectedMotionSensor struct {
	*service.S

	MotionDetected *characteristic.MotionDetected
	Name           *characteristic.Name
}

func NewKonnectedMotionSensor(name string) *KonnectedMotionSensor {
	svc := KonnectedMotionSensor{}
	svc.S = service.New(service.TypeMotionSensor)

	svc.MotionDetected = characteristic.NewMotionDetected()
	svc.AddC(svc.MotionDetected.C)

	svc.Name = characteristic.NewName()
	svc.Name.SetValue(name)
	svc.AddC(svc.Name.C)

	return &svc
}

type KonnectedBuzzer struct {
	*service.S

	Name   *characteristic.Name
	Beeper *beeper
}

func NewKonnectedBuzzer(name string) *KonnectedBuzzer {
	svc := KonnectedBuzzer{}
	svc.S = service.New("EE") // custom

	svc.Name = characteristic.NewName()
	svc.Name.SetValue(name)
	svc.AddC(svc.Name.C)

	svc.Beeper = NewBeeper()
	svc.Beeper.Description = "Buzzer"
	svc.Beeper.SetValue(false)

	return &svc
}

type beeper struct {
	*characteristic.Bool
}

func NewBeeper() *beeper {
	c := characteristic.NewBool("EE1")
	c.Format = characteristic.FormatBool

	c.Permissions = []string{characteristic.PermissionRead, characteristic.PermissionWrite, characteristic.PermissionEvents}
	c.SetValue(false)

	return &beeper{c}
}
