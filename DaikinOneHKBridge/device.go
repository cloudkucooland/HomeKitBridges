package dhkb

import (
	"context"

	"github.com/cloudkucooland/go-daikin"

	"github.com/brutella/hap/accessory"
	"github.com/brutella/hap/characteristic"
	"github.com/brutella/hap/log"
)

type DaikinAccessory struct {
	*accessory.A
	Thermostat *daikinThermostat
	Fan        *daikinFan
	Filter     *daikinFilter
	Device     *daikin.Device
}

func NewDaikinOne(client *daikin.Client, device *daikin.Device) *DaikinAccessory {
	a := DaikinAccessory{
		Device: device,
	}

	info := accessory.Info{
		Name:         device.Name,
		SerialNumber: device.ID,
		Manufacturer: "Daikin",
		Model:        device.Model,
		Firmware:     device.FirmwareVersion,
	}

	a.A = accessory.New(info, accessory.TypeThermostat)

	a.Thermostat = newDaikinThermostat()
	a.AddS(a.Thermostat.S)

	a.Fan = newDaikinFan()
	a.AddS(a.Fan.S)

	a.Filter = newDaikinFilter()
	a.AddS(a.Filter.S)

	status, err := device.GetInfo(context.Background())
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
		targetTemp = status.HeatSetpoint
	case daikin.ModeCool:
		hapMode = characteristic.CurrentHeatingCoolingStateCool
		targetTemp = status.CoolSetpoint
	case daikin.ModeAuto:
		// CurrentHeatingCoolingState doesn't have Auto, only Target does.
		hapMode = characteristic.CurrentHeatingCoolingStateOff
		targetTemp = status.CoolSetpoint - 2.5
	case daikin.ModeOff:
		hapMode = characteristic.CurrentHeatingCoolingStateOff
		targetTemp = status.CoolSetpoint
	}
	a.Thermostat.CurrentHeatingCoolingState.SetValue(hapMode)
	a.Thermostat.TargetHeatingCoolingState.SetValue(hapMode)
	a.Thermostat.TargetTemperature.SetValue(targetTemp)
	a.Thermostat.CurrentTemperature.SetValue(status.IndoorTemp)
	a.Thermostat.CurrentRelativeHumidity.SetValue(float64(status.IndoorHumidity))
	a.Thermostat.TargetRelativeHumidity.SetValue(float64(status.DehumSetpoint))
	a.Thermostat.CoolingThresholdTemperature.SetValue(status.CoolSetpoint)
	a.Thermostat.HeatingThresholdTemperature.SetValue(status.HeatSetpoint)

	a.Thermostat.TemperatureDisplayUnits.SetValue(0)
	a.Thermostat.TemperatureDisplayUnits.OnValueRemoteUpdate(func(s int) {
		log.Info.Printf("setting display units to %d from handler (unsupported)", s)
	})

	// add handler for setting the system state
	a.Thermostat.TargetHeatingCoolingState.OnValueRemoteUpdate(func(s int) {
		log.Info.Printf("setting target state to %d from handler", s)
		mode := daikin.ModeOff
		heat := a.Thermostat.HeatingThresholdTemperature.Value()
		cool := a.Thermostat.CoolingThresholdTemperature.Value()

		// convert homekit value to values the daikin wants
		switch s {
		case characteristic.TargetHeatingCoolingStateOff:
			mode = daikin.ModeOff
		case characteristic.TargetHeatingCoolingStateCool:
			mode = daikin.ModeCool
			cool = a.Thermostat.TargetTemperature.Value()
			heat = cool - 5
		case characteristic.TargetHeatingCoolingStateHeat:
			mode = daikin.ModeHeat
			heat = a.Thermostat.TargetTemperature.Value()
			cool = heat + 5
		case characteristic.TargetHeatingCoolingStateAuto:
			mode = daikin.ModeAuto
			target := a.Thermostat.TargetTemperature.Value()
			heat = target - 2.5
			cool = target + 2.5
		}

		if err := a.Device.SetTemps(context.Background(), mode, heat, cool); err != nil {
			log.Info.Println(err.Error())
		}
	})

	a.Thermostat.TargetTemperature.OnValueRemoteUpdate(func(s float64) {
		log.Info.Printf("setting target temperature to %f from handler", s)
		mode := daikin.SystemMode(a.Thermostat.TargetHeatingCoolingState.Value())
		heat := a.Thermostat.HeatingThresholdTemperature.Value()
		cool := a.Thermostat.CoolingThresholdTemperature.Value()

		switch mode {
		case daikin.ModeOff:
			heat = 20
			cool = 25
		case daikin.ModeCool:
			heat = s - 5
			cool = s
		case daikin.ModeHeat:
			heat = s
			cool = s + 5
		default:
			heat = s - 2.5
			cool = s + 2.5
		}

		if err := a.Device.SetTemps(context.Background(), mode, heat, cool); err != nil {
			log.Info.Println(err.Error())
		}
	})

	a.Thermostat.TargetRelativeHumidity.OnValueRemoteUpdate(func(s float64) {
		log.Info.Printf("setting target relative humidity to %f from handler", s)
		if err := a.Device.SetDehumidifySetpoint(context.Background(), int(s)); err != nil {
			log.Info.Println(err.Error())
		}
	})

	if status.FanCirculate > 0 {
		a.Fan.Active.SetValue(characteristic.ActiveActive)
	} else {
		a.Fan.Active.SetValue(characteristic.ActiveInactive)
	}

	// never go to Off, just return to the schededule
	a.Fan.Active.OnValueRemoteUpdate(func(s int) {
		log.Info.Printf("setting fan state to %d from handler", s)
		circulate := 0
		if s == characteristic.ActiveActive {
			circulate = 1
		}
		if err := a.Device.SetFan(context.Background(), int(status.Fan), circulate); err != nil {
			log.Info.Println(err.Error())
		}
	})

	a.Fan.TargetState.SetValue(characteristic.TargetFanStateAuto)
	a.Fan.TargetState.OnValueRemoteUpdate(func(s int) {
		log.Info.Printf("setting target fan state to %d from handler", s)
		mode := 0 // Auto
		if s == characteristic.TargetFanStateManual {
			mode = 1 // On
		}
		if err := a.Device.SetFan(context.Background(), mode, 1); err != nil {
			log.Info.Println(err.Error())
		}
	})

	a.Fan.RotationSpeed.OnValueRemoteUpdate(func(s float64) {
		log.Info.Printf("setting fan rotational speed to %f from handler (unsupported)", s)
	})

	if status.ActiveError != "" {
		a.Filter.FilterChangeIndication.SetValue(1)
	}

	return &a
}

func Update(a *DaikinAccessory) error {
	status, err := a.Device.GetInfo(context.Background())
	if err != nil {
		return err
	}

	a.Thermostat.CurrentTemperature.SetValue(status.IndoorTemp)
	a.Thermostat.CurrentRelativeHumidity.SetValue(float64(status.IndoorHumidity))
	a.Thermostat.TargetRelativeHumidity.SetValue(float64(status.DehumSetpoint))
	a.Thermostat.CoolingThresholdTemperature.SetValue(status.CoolSetpoint)
	a.Thermostat.HeatingThresholdTemperature.SetValue(status.HeatSetpoint)

	hapMode := characteristic.CurrentHeatingCoolingStateOff
	var targetTemp float64 = a.Thermostat.TargetTemperature.Value()
	switch status.Mode {
	case daikin.ModeHeat:
		hapMode = characteristic.CurrentHeatingCoolingStateHeat
		targetTemp = status.HeatSetpoint
	case daikin.ModeCool:
		hapMode = characteristic.CurrentHeatingCoolingStateCool
		targetTemp = status.CoolSetpoint
	case daikin.ModeAuto:
		hapMode = characteristic.TargetHeatingCoolingStateAuto
		targetTemp = status.CoolSetpoint - 2.5
	case daikin.ModeOff:
		hapMode = characteristic.TargetHeatingCoolingStateOff
		targetTemp = status.CoolSetpoint
	}
	a.Thermostat.TargetHeatingCoolingState.SetValue(hapMode)
	a.Thermostat.TargetTemperature.SetValue(targetTemp)

	if status.FanCirculate > 0 {
		a.Fan.Active.SetValue(characteristic.ActiveActive)
	} else {
		a.Fan.Active.SetValue(characteristic.ActiveInactive)
	}

	if status.ActiveError != "" {
		a.Filter.FilterChangeIndication.SetValue(1)
	}

	return nil
}
