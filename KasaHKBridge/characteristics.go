package kasahkbridge

import (
	"github.com/brutella/hap/characteristic"
)

type volt struct {
	*characteristic.Int
}

func NewVolt() *volt {
	c := characteristic.NewInt("10A")
	c.Format = characteristic.FormatUInt32
	c.Permissions = []string{characteristic.PermissionRead, characteristic.PermissionEvents}
	c.Description = "Voltage"
	c.SetValue(120)

	return &volt{c}
}

type watt struct {
	*characteristic.Int
}

func NewWatt() *watt {
	c := characteristic.NewInt("10D")
	c.Format = characteristic.FormatUInt32
	c.Permissions = []string{characteristic.PermissionRead, characteristic.PermissionEvents}
	c.Description = "Watts"
	c.SetValue(0)

	return &watt{c}
}

type amp struct {
	*characteristic.Int
}

func NewAmp() *amp {
	c := characteristic.NewInt("10B") // 126
	c.Format = characteristic.FormatUInt32
	c.Permissions = []string{characteristic.PermissionRead, characteristic.PermissionEvents}
	c.Description = "Current MA"
	c.SetValue(0)

	return &amp{c}
}

// https://github.com/plasticrake/homebridge-tplink-smarthome/blob/master/src/characteristics/amperes.ts
// volt E863F10A-079E-48FF-8F27-9C2605A29F52
// amps E863F126-079E-48FF-8F27-9C2605A29F52
// watt E863F10D-079E-48FF-8F27-9C2605A29F52
// kwh  E863F10C-079E-48FF-8F27-9C2605A29F52

// {"smartlife.iot.dimmer":{"get_dimmer_parameters":{"fadeOnTime":1000,"fadeOffTime":1000,"gentleOnTime":3000,"gentleOffTime":10000,"rampRate":30,"minThreshold":23,"bulb_type":1,"err_code":0}}}

type fadeOnTime struct {
	*characteristic.Int
}

func NewFadeOnTime() *fadeOnTime {
	c := characteristic.NewInt("110")
	c.Format = characteristic.FormatUInt32
	c.Permissions = []string{characteristic.PermissionRead, characteristic.PermissionEvents}
	c.Description = "Fade On Time"
	c.SetValue(0)

	return &fadeOnTime{c}
}

type fadeOffTime struct {
	*characteristic.Int
}

func NewFadeOffTime() *fadeOffTime {
	c := characteristic.NewInt("111")
	c.Format = characteristic.FormatUInt32
	c.Permissions = []string{characteristic.PermissionRead, characteristic.PermissionEvents}
	c.Description = "Fade Off Time"
	c.SetValue(0)

	return &fadeOffTime{c}
}

type gentleOnTime struct {
	*characteristic.Int
}

func NewGentleOnTime() *gentleOnTime {
	c := characteristic.NewInt("112")
	c.Format = characteristic.FormatUInt32
	c.Permissions = []string{characteristic.PermissionRead, characteristic.PermissionEvents}
	c.Description = "Gentle On Time"
	c.SetValue(0)

	return &gentleOnTime{c}
}

type gentleOffTime struct {
	*characteristic.Int
}

func NewGentleOffTime() *gentleOffTime {
	c := characteristic.NewInt("113")
	c.Format = characteristic.FormatUInt32
	c.Permissions = []string{characteristic.PermissionRead, characteristic.PermissionEvents}
	c.Description = "Gentle Off Time"
	c.SetValue(0)

	return &gentleOffTime{c}
}

type rampRate struct {
	*characteristic.Int
}

func NewRampRate() *rampRate {
	c := characteristic.NewInt("114")
	c.Format = characteristic.FormatUInt32
	c.Permissions = []string{characteristic.PermissionRead, characteristic.PermissionEvents}
	c.Description = "Ramp Rate"
	c.SetValue(0)

	return &rampRate{c}
}

type minThreshold struct {
	*characteristic.Int
}

func NewMinThreshold() *minThreshold {
	c := characteristic.NewInt("115")
	c.Format = characteristic.FormatUInt32
	c.Permissions = []string{characteristic.PermissionRead, characteristic.PermissionEvents}
	c.Description = "Min Threshold"
	c.SetValue(0)

	return &minThreshold{c}
}
