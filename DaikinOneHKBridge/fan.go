package dhkb

import (
	// "github.com/brutella/hap/accessory"
	"github.com/brutella/hap/characteristic"
	"github.com/brutella/hap/service"
)

type daikinFan struct {
	*service.S

	Active        *characteristic.Active
	CurrentState  *characteristic.CurrentFanState
	TargetState   *characteristic.TargetFanState
	RotationSpeed *characteristic.RotationSpeed
}

func newDaikinFan() *daikinFan {
	s := daikinFan{}
	s.S = service.New(service.TypeFanV2)

	s.Active = characteristic.NewActive()
	s.AddC(s.Active.C)

	s.CurrentState = characteristic.NewCurrentFanState()
	s.AddC(s.CurrentState.C)

	s.TargetState = characteristic.NewTargetFanState()
	s.AddC(s.TargetState.C)

	s.RotationSpeed = characteristic.NewRotationSpeed()
	s.RotationSpeed.SetStepValue(33)
	s.AddC(s.RotationSpeed.C)

	return &s
}
