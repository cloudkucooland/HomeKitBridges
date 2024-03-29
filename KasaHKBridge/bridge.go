package kasahkbridge

import (
	// "time"

	"github.com/brutella/hap/accessory"
	"github.com/brutella/hap/characteristic"
	// "github.com/brutella/hap/log"
	"github.com/brutella/hap/service"
)

var root *accessory.Bridge

// Bridge is used by the startup to build the generic bridge type on which all other devices hang
func Bridge() *accessory.A {
	root = accessory.NewBridge(accessory.Info{
		Name:         "Kasa-Homekit Bridge",
		SerialNumber: "1101",
		Manufacturer: "cloudkucooland",
		Model:        "kasa-homekit",
		Firmware:     "0.0.3",
	})
	root.A.Id = 1

	// create the settings service
	settings := settingsService{}
	settings.S = service.New("E880") // custom

	// doesn't seem to work
	settings.Name = characteristic.NewName()
	settings.Name.SetValue("Settings")
	settings.S.AddC(settings.Name.C)

	settings.PollRate = newPollRate()
	settings.S.AddC(settings.PollRate.C)
	// causes a hang
	/* settings.PollRate.OnValueRemoteUpdate(func(newstate int) {
			log.Info.Printf("setting poll rate: %d", newstate)
			pollInterval = time.Second * time.Duration(newstate)
	        // write to config file or other data store
		}) */

	root.A.AddS(settings.S)

	return root.A
}

// add bridge-wide tunable parameters here
type settingsService struct {
	*service.S

	Name     *characteristic.Name
	PollRate *pollRate
}

type pollRate struct {
	*characteristic.Int
}

func newPollRate() *pollRate {
	c := characteristic.NewInt("E8802")
	c.Format = characteristic.FormatUInt32
	c.Permissions = []string{characteristic.PermissionRead, characteristic.PermissionWrite}
	c.Description = "Poll Rate"
	c.SetMinValue(10)
	c.SetMaxValue(3600)
	c.SetValue(60)

	return &pollRate{c}
}
