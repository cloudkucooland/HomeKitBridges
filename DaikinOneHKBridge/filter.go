package dhkb

import (
	// "github.com/brutella/hap/accessory"
	"github.com/brutella/hap/characteristic"
	"github.com/brutella/hap/service"
)

type daikinFilter struct {
	*service.S

	FilterChangeIndication *characteristic.FilterChangeIndication
	FilterLifeLevel        *characteristic.FilterLifeLevel
}

func newDaikinFilter() *daikinFilter {
	s := daikinFilter{}
	s.S = service.New(service.TypeFilterMaintenance)

	s.FilterChangeIndication = characteristic.NewFilterChangeIndication()
	s.AddC(s.FilterChangeIndication.C)

	s.FilterLifeLevel = characteristic.NewFilterLifeLevel()
	s.AddC(s.FilterLifeLevel.C)

	return &s
}
