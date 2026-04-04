package kasahkbridge

import (
	"github.com/brutella/hap/characteristic"
)

// use the same values as other bridges
// https://github.com/plasticrake/homebridge-tplink-smarthome/blob/master/src/characteristics/amperes.ts
// volt E863F10A-079E-48FF-8F27-9C2605A29F52
// amps E863F126-079E-48FF-8F27-9C2605A29F52
// watt E863F10D-079E-48FF-8F27-9C2605A29F52
// kwh  E863F10C-079E-48FF-8F27-9C2605A29F52
// k volt amp hour E863F127-079E-48FF-8F27-9C2605A29F52
// volt amps E863F110-079E-48FF-8F27-9C2605A29F52

type volt struct {
	*characteristic.Int
}

func NewVolt() *volt {
	c := characteristic.NewInt("E863F10A-079E-48FF-8F27-9C2605A29F52")
	c.Format = characteristic.FormatUInt32
	c.Permissions = []string{characteristic.PermissionRead, characteristic.PermissionEvents}
	c.Description = "Voltage"
	c.Unit = "volt"
	_ = c.SetValue(120)

	return &volt{c}
}

type watt struct {
	*characteristic.Int
}

func NewWatt() *watt {
	c := characteristic.NewInt("E863F10D-079E-48FF-8F27-9C2605A29F52")
	c.Format = characteristic.FormatUInt32
	c.Permissions = []string{characteristic.PermissionRead, characteristic.PermissionEvents}
	c.Description = "Watts"
	c.Unit = "watt"
	_ = c.SetValue(0)

	return &watt{c}
}

type amp struct {
	*characteristic.Int
}

func NewAmp() *amp {
	c := characteristic.NewInt("E863F126-079E-48FF-8F27-9C2605A29F52")
	c.Format = characteristic.FormatUInt32
	c.Permissions = []string{characteristic.PermissionRead, characteristic.PermissionEvents}
	c.Description = "Current mA"
	c.Unit = "milliampre"
	_ = c.SetValue(0)

	return &amp{c}
}

// custom to us
// fade on       E8700110
// fade off      E8700111
// gentle on     E8700112
// gentle off    E8700113
// ramp rate     E8700114
// min threshold E8700115
// RSSI          E8700116

type fadeOnTime struct {
	*characteristic.Int
}

func NewFadeOnTime() *fadeOnTime {
	c := characteristic.NewInt("E8700110-079E-48FF-8F27-9C2605A29F52")
	c.Format = characteristic.FormatUInt32
	c.Permissions = []string{characteristic.PermissionRead, characteristic.PermissionWrite, characteristic.PermissionEvents}
	c.Description = "Fade On Time"
	c.Unit = "millisecond"
	c.SetMinValue(0)
	c.SetMaxValue(100000)
	_ = c.SetValue(0)

	return &fadeOnTime{c}
}

type fadeOffTime struct {
	*characteristic.Int
}

func NewFadeOffTime() *fadeOffTime {
	c := characteristic.NewInt("E8700111-079E-48FF-8F27-9C2605A29F52")
	c.Format = characteristic.FormatUInt32
	c.Permissions = []string{characteristic.PermissionRead, characteristic.PermissionWrite, characteristic.PermissionEvents}
	c.Description = "Fade Off Time"
	c.Unit = "millisecond"
	c.SetMinValue(0)
	c.SetMaxValue(100000)
	_ = c.SetValue(0)

	return &fadeOffTime{c}
}

type gentleOnTime struct {
	*characteristic.Int
}

func NewGentleOnTime() *gentleOnTime {
	c := characteristic.NewInt("E8700112-079E-48FF-8F27-9C2605A29F52")
	c.Format = characteristic.FormatUInt32
	c.Permissions = []string{characteristic.PermissionRead, characteristic.PermissionWrite, characteristic.PermissionEvents}
	c.Description = "Gentle On Time"
	c.Unit = "millisecond"
	c.SetMinValue(0)
	c.SetMaxValue(100000)
	_ = c.SetValue(0)

	return &gentleOnTime{c}
}

type gentleOffTime struct {
	*characteristic.Int
}

func NewGentleOffTime() *gentleOffTime {
	c := characteristic.NewInt("E8700113-079E-48FF-8F27-9C2605A29F52")
	c.Format = characteristic.FormatUInt32
	c.Permissions = []string{characteristic.PermissionRead, characteristic.PermissionWrite, characteristic.PermissionEvents}
	c.Description = "Gentle Off Time"
	c.Unit = "millisecond"
	c.SetMinValue(0)
	c.SetMaxValue(100000)
	_ = c.SetValue(0)

	return &gentleOffTime{c}
}

type rampRate struct {
	*characteristic.Int
}

func NewRampRate() *rampRate {
	c := characteristic.NewInt("E8700114-079E-48FF-8F27-9C2605A29F52")
	c.Format = characteristic.FormatUInt32
	c.Permissions = []string{characteristic.PermissionRead, characteristic.PermissionWrite, characteristic.PermissionEvents}
	c.Description = "Ramp Rate"
	c.Unit = "millisecond"
	c.SetMinValue(0)
	c.SetMaxValue(1000)
	_ = c.SetValue(0)

	return &rampRate{c}
}

type minThreshold struct {
	*characteristic.Int
}

func NewMinThreshold() *minThreshold {
	c := characteristic.NewInt("E8700115-079E-48FF-8F27-9C2605A29F52")
	c.Format = characteristic.FormatUInt32
	c.Permissions = []string{characteristic.PermissionRead, characteristic.PermissionWrite, characteristic.PermissionEvents}
	c.Description = "Minimum Threshold"
	c.Unit = "percentage"
	c.SetMinValue(0)
	c.SetMaxValue(100)
	_ = c.SetValue(0)

	return &minThreshold{c}
}

type rssi struct {
	*characteristic.Int
}

func NewRSSI() *rssi {
	c := characteristic.NewInt("E8700116-079E-48FF-8F27-9C2605A29F52")
	c.Format = characteristic.FormatInt32
	c.Permissions = []string{characteristic.PermissionRead, characteristic.PermissionEvents}
	c.Description = "Kasa RSSI"
	c.Unit = "decibel"
	c.SetMinValue(-110)
	c.SetMaxValue(0)
	_ = c.SetValue(-50)

	return &rssi{c}
}
