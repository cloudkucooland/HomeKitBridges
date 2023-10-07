package main

import (
	"fmt"

	"github.com/redgoose/daikin-skyport"

	"github.com/brutella/hap/accessory"
	"github.com/brutella/hap/characteristic"
	"github.com/brutella/hap/log"
)

type daikinAccessory struct {
	*accessory.A
	Thermostat *daikinThermostat
	Fan        *daikinFan
	Filter     *daikinFilter
}

func newDaikinOne(da *daikin.Daikin, details *daikin.Device) *daikinAccessory {
	a := daikinAccessory{}

	info := accessory.Info{
		Name:         details.Name,
		SerialNumber: details.Id,
		Manufacturer: "Daikin",
		Model:        details.Model,
		Firmware:     details.FirmwareVersion,
	}

	a.A = accessory.New(info, accessory.TypeThermostat)

	a.Thermostat = newDaikinThermostat()
	a.AddS(a.Thermostat.S)

	a.Fan = newDaikinFan()
	a.AddS(a.Fan.S)

	a.Filter = newDaikinFilter()
	a.AddS(a.Filter.S)

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
		targetTemp = float64(status.HspActive)
	case daikin.ModeCool:
		hapMode = characteristic.CurrentHeatingCoolingStateCool
		targetTemp = float64(status.CspActive)
	case daikin.ModeAuto:
		// hapMode = characteristic.CurrentHeatingCoolingStateAuto
		targetTemp = float64(status.CspActive) - 2.5
	case daikin.ModeOff:
		hapMode = characteristic.CurrentHeatingCoolingStateOff
		targetTemp = float64(status.CspActive)
	}
	a.Thermostat.CurrentHeatingCoolingState.SetValue(hapMode)
	a.Thermostat.TargetHeatingCoolingState.SetValue(hapMode)
	a.Thermostat.TargetTemperature.SetValue(targetTemp)
	a.Thermostat.CurrentTemperature.SetValue(float64(status.TempIndoor))
	a.Thermostat.CurrentRelativeHumidity.SetValue(float64(status.HumIndoor))
	a.Thermostat.TargetRelativeHumidity.SetValue(float64(status.DehumSP))
	a.Thermostat.CoolingThresholdTemperature.SetValue(float64(status.CspActive))
	a.Thermostat.HeatingThresholdTemperature.SetValue(float64(status.HspActive))

	a.Thermostat.TemperatureDisplayUnits.SetValue(status.Units)
	a.Thermostat.TemperatureDisplayUnits.OnValueRemoteUpdate(func(s int) {
		log.Info.Printf("setting display units to %d from handler", s)
		// Homekit and Daikin flip these...
		x := 1
		if s == 1 {
			x = 0
		}
		json := fmt.Sprintf(`{"units": %d}`, x)
		if err := da.UpdateDeviceRaw(a.Info.SerialNumber.Value(), json); err != nil {
			log.Info.Println(err.Error())
		}
	})

	// add handler for setting the system state
	a.Thermostat.TargetHeatingCoolingState.OnValueRemoteUpdate(func(s int) {
		log.Info.Printf("setting target state to %d from handler", s)
		setPoint := daikin.SetTempParams{}

		// convert homekit value to values the daikin wants
		switch s {
		case characteristic.CurrentHeatingCoolingStateOff:
			if err := da.SetMode(a.Info.SerialNumber.Value(), daikin.ModeOff); err != nil {
				log.Info.Println(err.Error())
				return
			}
			setPoint.HeatSetpoint = float32(20)
			setPoint.CoolSetpoint = float32(25)
		case characteristic.CurrentHeatingCoolingStateCool:
			if err := da.SetMode(a.Info.SerialNumber.Value(), daikin.ModeCool); err != nil {
				log.Info.Println(err.Error())
				return
			}
			setPoint.HeatSetpoint = float32(a.Thermostat.TargetTemperature.Value() - 5)
			setPoint.CoolSetpoint = float32(a.Thermostat.TargetTemperature.Value())
		case characteristic.CurrentHeatingCoolingStateHeat:
			if err := da.SetMode(a.Info.SerialNumber.Value(), daikin.ModeHeat); err != nil {
				log.Info.Println(err.Error())
				return
			}
			setPoint.HeatSetpoint = float32(a.Thermostat.TargetTemperature.Value())
			setPoint.CoolSetpoint = float32(a.Thermostat.TargetTemperature.Value() + 5)
		default: // characteristic.CurrentHeatingCoolingStateAuto:
			if err := da.SetMode(a.Info.SerialNumber.Value(), daikin.ModeAuto); err != nil {
				log.Info.Println(err.Error())
				return
			}
			setPoint.HeatSetpoint = float32(a.Thermostat.TargetTemperature.Value() - 2.5)
			setPoint.CoolSetpoint = float32(a.Thermostat.TargetTemperature.Value() + 2.5)
		}

		log.Info.Printf("%+v", setPoint)
		if err := da.SetTemp(a.Info.SerialNumber.Value(), setPoint); err != nil {
			log.Info.Println(err.Error())
			return
		}
	})

	a.Thermostat.TargetTemperature.OnValueRemoteUpdate(func(s float64) {
		log.Info.Printf("setting target temperature to %f from handler", s)
		setPoint := daikin.SetTempParams{}

		// convert homekit valuesto values the daikin wants
		switch a.Thermostat.CurrentHeatingCoolingState.Value() {
		case characteristic.CurrentHeatingCoolingStateOff:
			setPoint.HeatSetpoint = float32(20)
			setPoint.CoolSetpoint = float32(25)
		case characteristic.CurrentHeatingCoolingStateCool:
			setPoint.HeatSetpoint = float32(s - 5)
			setPoint.CoolSetpoint = float32(s)
		case characteristic.CurrentHeatingCoolingStateHeat:
			setPoint.HeatSetpoint = float32(s)
			setPoint.CoolSetpoint = float32(s + 5)
		default:
			setPoint.HeatSetpoint = float32(a.Thermostat.TargetTemperature.Value() - 2.5)
			setPoint.CoolSetpoint = float32(a.Thermostat.TargetTemperature.Value() + 2.5)
		}

		// log.Info.Printf("%+v", setPoint)
		if err := da.SetTemp(a.Info.SerialNumber.Value(), setPoint); err != nil {
			log.Info.Println(err.Error())
		}
	})

	a.Thermostat.TargetRelativeHumidity.OnValueRemoteUpdate(func(s float64) {
		log.Info.Printf("setting target relative humidity to %f from handler", s)
		json := fmt.Sprintf(`{"dehumSP": %f}`, s)
		if err := da.UpdateDeviceRaw(a.Info.SerialNumber.Value(), json); err != nil {
			log.Info.Println(err.Error())
		}
	})

	switch status.FanCirculate {
	case daikin.FanCirculateOff:
		a.Fan.Active.SetValue(characteristic.CurrentFanStateInactive)
	case daikin.FanCirculateOn:
		a.Fan.Active.SetValue(characteristic.CurrentFanStateBlowingAir)
	case daikin.FanCirculateSched:
		a.Fan.Active.SetValue(characteristic.CurrentFanStateIdle)
	}

	// never go to Off, just return to the schededule
	a.Fan.Active.OnValueRemoteUpdate(func(s int) {
		switch s {
		case characteristic.CurrentFanStateBlowingAir:
			log.Info.Printf("setting fan state to On from handler")
			da.SetFanMode(a.Info.SerialNumber.Value(), daikin.FanCirculateOn)
		case characteristic.CurrentFanStateIdle:
			log.Info.Printf("setting fan state to Idle from handler")
			da.SetFanMode(a.Info.SerialNumber.Value(), daikin.FanCirculateSched)
		case characteristic.CurrentFanStateInactive:
			log.Info.Printf("setting fan state to Inactive from handler")
			da.SetFanMode(a.Info.SerialNumber.Value(), daikin.FanCirculateSched)
		default:
			log.Info.Printf("setting fan state to Default from handler")
			da.SetFanMode(a.Info.SerialNumber.Value(), daikin.FanCirculateSched)
		}
	})

	a.Fan.TargetState.SetValue(characteristic.TargetFanStateAuto)
	a.Fan.TargetState.OnValueRemoteUpdate(func(s int) {
		switch s {
		case characteristic.TargetFanStateManual:
			log.Info.Printf("setting target fan state Manual from handler")
			da.SetFanMode(a.Info.SerialNumber.Value(), daikin.FanCirculateOn)
		default:
			log.Info.Printf("setting target fan state Automatic from handler")
			da.SetFanMode(a.Info.SerialNumber.Value(), daikin.FanCirculateSched)
		}
	})

	switch status.FanCirculateSpeed {
	case daikin.FanCirculateSpeedLow:
		a.Fan.RotationSpeed.SetValue(33)
	case daikin.FanCirculateSpeedMed:
		a.Fan.RotationSpeed.SetValue(66)
	case daikin.FanCirculateSpeedHigh:
		a.Fan.RotationSpeed.SetValue(99)
	}

	a.Fan.RotationSpeed.OnValueRemoteUpdate(func(s float64) {
		log.Info.Printf("setting fan rotational speed to %f from handler", s)
		rs := daikin.FanCirculateSpeedLow
		if s > 33 {
			rs = daikin.FanCirculateSpeedMed
		}
		if s > 66 {
			rs = daikin.FanCirculateSpeedHigh
		}
		da.SetFanSpeed(a.Info.SerialNumber.Value(), rs)
	})

	if status.AlertMediaAirFilterActive {
		a.Filter.FilterChangeIndication.SetValue(1)
	}

	return &a
}

func update(a *daikinAccessory, d *daikin.Daikin) error {
	status, err := d.GetDeviceInfo(a.Info.SerialNumber.Value())
	if err != nil {
		return err
	}

	// log.Info.Printf("%+v", status)

	a.Thermostat.CurrentTemperature.SetValue(float64(status.TempIndoor))
	a.Thermostat.CurrentRelativeHumidity.SetValue(float64(status.HumIndoor))
	a.Thermostat.TargetRelativeHumidity.SetValue(float64(status.DehumSP))
	a.Thermostat.CoolingThresholdTemperature.SetValue(float64(status.CspActive))
	a.Thermostat.HeatingThresholdTemperature.SetValue(float64(status.HspActive))

	hapMode := characteristic.CurrentHeatingCoolingStateOff
	var targetTemp float64 = a.Thermostat.TargetTemperature.Value()
	switch status.Mode {
	case daikin.ModeHeat:
		hapMode = characteristic.CurrentHeatingCoolingStateHeat
		targetTemp = float64(status.HspActive)
	case daikin.ModeCool:
		hapMode = characteristic.CurrentHeatingCoolingStateCool
		targetTemp = float64(status.CspActive)
	case daikin.ModeAuto:
		// hapMode = characteristic.CurrentHeatingCoolingStateAuto
		targetTemp = float64(status.CspActive) - 2.5
	case daikin.ModeOff:
		hapMode = characteristic.CurrentHeatingCoolingStateOff
		targetTemp = float64(status.CspActive)
	}
	a.Thermostat.TargetHeatingCoolingState.SetValue(hapMode)
	a.Thermostat.TargetTemperature.SetValue(targetTemp)

	switch status.FanCirculate {
	case daikin.FanCirculateOff:
		a.Fan.Active.SetValue(characteristic.CurrentFanStateInactive)
	case daikin.FanCirculateOn:
		a.Fan.Active.SetValue(characteristic.CurrentFanStateBlowingAir)
	case daikin.FanCirculateSched:
		a.Fan.Active.SetValue(characteristic.CurrentFanStateIdle)
	}

	switch status.FanCirculateSpeed {
	case daikin.FanCirculateSpeedLow:
		a.Fan.RotationSpeed.SetValue(33)
	case daikin.FanCirculateSpeedMed:
		a.Fan.RotationSpeed.SetValue(66)
	case daikin.FanCirculateSpeedHigh:
		a.Fan.RotationSpeed.SetValue(99)
	}

	if status.AlertMediaAirFilterActive {
		a.Filter.FilterChangeIndication.SetValue(1)
	}

	// % remaining
	var filtLifeRemaining float64 = (100.0 - ((float64(status.AlertMediaAirFilterDays) / float64(status.AlertMediaAirFilterDaysLimit)) * 100.0))
	a.Filter.FilterLifeLevel.SetValue(filtLifeRemaining)

	return nil
}
