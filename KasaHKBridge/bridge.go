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
		Firmware:     "0.0.5",
	})
	root.A.Id = 1

	// create the settings service
	settings := settingsService{}
	settings.S = service.New("E8800000-0000-1000-8000-0026BB765291") // custom

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
		        // write to datastore
	            // restart the poller (this is the hard part)
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
	c := characteristic.NewInt("E8802000-0000-1000-8000-0026BB765291")
	c.Format = characteristic.FormatUInt32
	c.Permissions = []string{characteristic.PermissionRead, characteristic.PermissionWrite, characteristic.PermissionEvents}
	c.Description = "Poll Rate"
	c.Unit = "seconds"
	c.SetMinValue(10)
	c.SetMaxValue(3600)
	c.SetValue(60)

	return &pollRate{c}
}
