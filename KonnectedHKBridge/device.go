package konnectedkhbridge

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/brutella/hap/accessory"
	"github.com/brutella/hap/characteristic"
	"github.com/brutella/hap/log"
	"github.com/brutella/hap/service"
)

type Konnected struct {
	*accessory.A
	pins           map[uint8]PinHandler
	SecuritySystem *KonnectedSvc
	Buzzer         *KonnectedBuzzer
	ip             string
	password       string

	alarmCancel context.CancelFunc
	mu          sync.Mutex
}

func NewKonnected(details *system, d *Device, stateManager *PersistentState) *Konnected {
	acc := Konnected{
		pins: make(map[uint8]PinHandler),
	}

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
	acc.SecuritySystem.SecuritySystemCurrentState.SetValue(stateManager.SecurityMode)
	acc.SecuritySystem.SecuritySystemTargetState.SetValue(stateManager.SecurityMode)

	acc.SecuritySystem.SecuritySystemTargetState.OnValueRemoteUpdate(func(newval int) {
		current := acc.SecuritySystem.SecuritySystemCurrentState.Value()

		if newval == characteristic.SecuritySystemTargetStateDisarm {
			log.Info.Println("System Disarmed: Stopping all alarm logic")
			acc.stopAllAlarmLogic()
			acc.SecuritySystem.SecuritySystemCurrentState.SetValue(characteristic.SecuritySystemCurrentStateDisarmed)
			acc.beep()
			if err := stateManager.UpdateAndSave(newval); err != nil {
				log.Info.Printf("Failed to persist state: %v", err)
			}
			return
		}

		// Prevent state changes if already triggered (unless disarming)
		if current == characteristic.SecuritySystemCurrentStateAlarmTriggered {
			log.Info.Println("Action ignored: System is currently triggered. Disarm first.")
			return
		}

		acc.SecuritySystem.SecuritySystemCurrentState.SetValue(newval)
		acc.beep()

		if err := stateManager.UpdateAndSave(newval); err != nil {
			log.Info.Printf("Failed to persist state: %v", err)
		}
	})

	alarmType := characteristic.NewSecuritySystemAlarmType()
	alarmType.SetValue(1)
	acc.SecuritySystem.AddC(alarmType.C)

	// convert zones from config to pins
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

func (k *Konnected) triggerCountdown() {
	k.mu.Lock()
	defer k.mu.Unlock()

	// If already counting down or triggered, don't start another goroutine
	if k.alarmCancel != nil {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	k.alarmCancel = cancel

	go func() {
		log.Info.Println("Alarm countdown started...")
		k.doBuzz(`"state":1, "momentary":50, "pause":450`)

		select {
		case <-time.After(1 * time.Minute):
			k.instantAlarm()
		case <-ctx.Done():
			log.Info.Println("Countdown cancelled by user disarm.")
		}
	}()
}

func (k *Konnected) stopAllAlarmLogic() {
	k.mu.Lock()
	defer k.mu.Unlock()

	if k.alarmCancel != nil {
		k.alarmCancel()
		k.alarmCancel = nil
	}
	k.doBuzz(`"state": 0`)
}

func (s *KonnectedContactSensor) Handle(state uint8, k *Konnected) {
    log.Info.Printf("Pin Update: %s (pin %d) is %d", s.Name.Value(), s.ContactSensorState.Value(), state)
	s.ContactSensorState.SetValue(int(state))

	if state == 0 {
		k.doorCloseChirps()
		return
	}

	sysState := k.SecuritySystem.SecuritySystemCurrentState.Value()

	switch sysState {
	case characteristic.SecuritySystemCurrentStateAwayArm:
		log.Info.Printf("Entry on pin while Away")
		k.triggerCountdown()

	case characteristic.SecuritySystemCurrentStateNightArm:
		log.Info.Printf("Entry on pin while Night")
		k.instantAlarm()

	case characteristic.SecuritySystemCurrentStateStayArm:
		k.doorOpenChirps()
	}
}

func (s *KonnectedMotionSensor) Handle(state uint8, k *Konnected) {
    log.Info.Printf("Pin Update: %s (pin %d) is %d", s.Name.Value(), s.MotionDetected.Value(), state)
	s.MotionDetected.SetValue(state == 1)

	if state == 0 {
		return
	}

	sysState := k.SecuritySystem.SecuritySystemCurrentState.Value()

	switch sysState {
	case characteristic.SecuritySystemCurrentStateAwayArm:
		log.Info.Printf("Motion detected while Away")
		k.triggerCountdown()

	case characteristic.SecuritySystemCurrentStateNightArm:
		log.Info.Printf("Motion detected while Night")
		k.motionChirps()

	case characteristic.SecuritySystemCurrentStateStayArm:
		// optional
	}
}

func (b *KonnectedBuzzer) Handle(state uint8, k *Konnected) {
	log.Info.Printf("%s: %d", b.Switch.Name.Value(), state)
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
