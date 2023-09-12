package main

import (
	"github.com/cloudkucooland/daikin-one/daikin"

	"github.com/brutella/hap/accessory"
	"github.com/brutella/hap/characteristic"
	"github.com/brutella/hap/log"
	"github.com/brutella/hap/service"
)

func newDaikinOne(da *daikin.Daikin, details *daikin.Device) *accessory.Thermostat {
	a := accessory.Thermostat{}

	info := accessory.Info{
		Name:         details.Name,
		SerialNumber: details.Id,
		Manufacturer: "Daikin",
		Model:        details.Model,
		Firmware:     details.FirmwareVersion,
	}

	a.A = accessory.New(info, accessory.TypeThermostat)

	a.Thermostat = service.NewThermostat()
	a.AddS(a.Thermostat.S)

	status, err := da.GetDeviceInfo(details.Id)
	if err != nil {
		log.Info.Println(err.Error())
		return &a
	}

	// set initial state
	hapMode := characteristic.CurrentHeatingCoolingStateOff
	var targetTemp float64 = 22.5
	switch status.Mode {
	case daikin.ModeHeat:
		hapMode = characteristic.CurrentHeatingCoolingStateHeat
		targetTemp = float64(status.HeatSetpoint)
	case daikin.ModeCool:
		hapMode = characteristic.CurrentHeatingCoolingStateCool
		targetTemp = float64(status.CoolSetpoint)
	}
	a.Thermostat.CurrentHeatingCoolingState.SetValue(hapMode)
	a.Thermostat.TargetHeatingCoolingState.SetValue(hapMode)
	a.Thermostat.TargetTemperature.SetValue(targetTemp)
	a.Thermostat.CurrentTemperature.SetValue(float64(status.TempIndoor))

	// a.Thermostat.TemperatureDisplayUnits = status.TemperatureDisplayUnits

	// add handler for setting the system state
	a.Thermostat.TargetHeatingCoolingState.OnValueRemoteUpdate(func(s int) {
		log.Info.Printf("setting target state to %d from handler", s)
		setPoint := daikin.ModeSetpointOptions{}

		// convert homekit value to values the daikin wants
		switch s {
		case characteristic.CurrentHeatingCoolingStateOff:
			setPoint.Mode = daikin.ModeOff
			setPoint.HeatSetpoint = float32(20)
			setPoint.CoolSetpoint = float32(25)
		case characteristic.CurrentHeatingCoolingStateCool:
			setPoint.Mode = daikin.ModeCool
			setPoint.HeatSetpoint = float32(a.Thermostat.TargetTemperature.Value() - 5)
			setPoint.CoolSetpoint = float32(a.Thermostat.TargetTemperature.Value())
		case characteristic.CurrentHeatingCoolingStateHeat:
			setPoint.Mode = daikin.ModeHeat
			setPoint.HeatSetpoint = float32(a.Thermostat.TargetTemperature.Value())
			setPoint.CoolSetpoint = float32(a.Thermostat.TargetTemperature.Value() + 5)
		default: // characteristic.CurrentHeatingCoolingStateAuto:
			setPoint.Mode = daikin.ModeAuto
			setPoint.HeatSetpoint = float32(a.Thermostat.TargetTemperature.Value() - 2.5)
			setPoint.CoolSetpoint = float32(a.Thermostat.TargetTemperature.Value() + 2.5)
		}

		log.Info.Printf("%s ModeSetpointOptions: %+v", a.Info.SerialNumber.Value(), setPoint)
		if err := da.UpdateModeSetpoint(a.Info.SerialNumber.Value(), setPoint); err != nil {
			log.Info.Println(err.Error())
			return
		}
		update(&a, da)
	})

	a.Thermostat.TargetTemperature.OnValueRemoteUpdate(func(s float64) {
		log.Info.Printf("setting target temperature to %f from handler", s)
		setPoint := daikin.ModeSetpointOptions{}

		// convert homekit valuesto values the daikin wants
		switch a.Thermostat.CurrentHeatingCoolingState.Value() {
		case characteristic.CurrentHeatingCoolingStateOff:
			setPoint.Mode = daikin.ModeOff
			setPoint.HeatSetpoint = float32(20)
			setPoint.CoolSetpoint = float32(25)
		case characteristic.CurrentHeatingCoolingStateCool:
			setPoint.Mode = daikin.ModeCool
			setPoint.HeatSetpoint = float32(s - 5)
			setPoint.CoolSetpoint = float32(s)
		case characteristic.CurrentHeatingCoolingStateHeat:
			setPoint.Mode = daikin.ModeHeat
			setPoint.HeatSetpoint = float32(s)
			setPoint.CoolSetpoint = float32(s + 5)
		default:
			setPoint.Mode = daikin.ModeAuto
			setPoint.HeatSetpoint = float32(a.Thermostat.TargetTemperature.Value() - 2.5)
			setPoint.CoolSetpoint = float32(a.Thermostat.TargetTemperature.Value() + 2.5)
		}

		log.Info.Printf("%s ModeSetpointOptions: %+v", a.Info.SerialNumber.Value(), setPoint)
		if err := da.UpdateModeSetpoint(a.Info.SerialNumber.Value(), setPoint); err != nil {
			log.Info.Println(err.Error())
			return
		}
		update(&a, da)
	})

	return &a
}

func update(a *accessory.Thermostat, d *daikin.Daikin) error {
	status, err := d.GetDeviceInfo(a.Info.SerialNumber.Value())
	if err != nil {
		return err
	}

	log.Info.Printf("%+v\n", status)

	a.Thermostat.CurrentTemperature.SetValue(float64(status.TempIndoor))

	hapMode := characteristic.CurrentHeatingCoolingStateOff
	var targetTemp float64 = a.Thermostat.TargetTemperature.Value()
	switch status.Mode {
	case daikin.ModeHeat:
		hapMode = characteristic.CurrentHeatingCoolingStateHeat
		targetTemp = float64(status.HeatSetpoint)
	case daikin.ModeCool:
		hapMode = characteristic.CurrentHeatingCoolingStateCool
		targetTemp = float64(status.CoolSetpoint)
	}
	a.Thermostat.TargetHeatingCoolingState.SetValue(hapMode)
	a.Thermostat.TargetTemperature.SetValue(targetTemp)
	return nil
}
