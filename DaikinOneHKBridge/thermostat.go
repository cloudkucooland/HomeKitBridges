package main

import (
	"github.com/brutella/hap/characteristic"
	"github.com/brutella/hap/service"
)

type daikinThermostat struct {
	*service.S

	CurrentHeatingCoolingState  *characteristic.CurrentHeatingCoolingState
	TargetHeatingCoolingState   *characteristic.TargetHeatingCoolingState
	CurrentTemperature          *characteristic.CurrentTemperature
	TargetTemperature           *characteristic.TargetTemperature
	TemperatureDisplayUnits     *characteristic.TemperatureDisplayUnits
	CurrentRelativeHumidity     *characteristic.CurrentRelativeHumidity
	TargetRelativeHumidity      *characteristic.TargetRelativeHumidity
	CoolingThresholdTemperature *characteristic.CoolingThresholdTemperature
	HeatingThresholdTemperature *characteristic.HeatingThresholdTemperature
}

func newDaikinThermostat() *daikinThermostat {
	s := daikinThermostat{}
	s.S = service.New(service.TypeThermostat)

	s.CurrentHeatingCoolingState = characteristic.NewCurrentHeatingCoolingState()
	s.AddC(s.CurrentHeatingCoolingState.C)

	s.TargetHeatingCoolingState = characteristic.NewTargetHeatingCoolingState()
	s.AddC(s.TargetHeatingCoolingState.C)

	s.CurrentTemperature = characteristic.NewCurrentTemperature()
	s.AddC(s.CurrentTemperature.C)

	s.TargetTemperature = characteristic.NewTargetTemperature()
	s.AddC(s.TargetTemperature.C)

	s.TemperatureDisplayUnits = characteristic.NewTemperatureDisplayUnits()
	s.AddC(s.TemperatureDisplayUnits.C)

	s.CurrentRelativeHumidity = characteristic.NewCurrentRelativeHumidity()
	s.AddC(s.CurrentRelativeHumidity.C)

	s.TargetRelativeHumidity = characteristic.NewTargetRelativeHumidity()
	s.AddC(s.TargetRelativeHumidity.C)

	s.CoolingThresholdTemperature = characteristic.NewCoolingThresholdTemperature()
	s.AddC(s.CoolingThresholdTemperature.C)

	s.HeatingThresholdTemperature = characteristic.NewHeatingThresholdTemperature()
	s.AddC(s.HeatingThresholdTemperature.C)

	return &s
}
