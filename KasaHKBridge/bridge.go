package kasahkbridge

import (
	"github.com/brutella/hap/accessory"
	// "github.com/brutella/hap/characteristic"
	// "github.com/brutella/hap/log"
	// "github.com/brutella/hap/service"
)

var root *accessory.Bridge

// Bridge is used by the startup to build the generic bridge type on which all other devices hang
func Bridge() *accessory.A {
	root = accessory.NewBridge(accessory.Info{
		Name:         "KasaHomekitBridge",
		SerialNumber: "1101",
		Manufacturer: "cloudkucooland",
		Model:        "KasaHomekitbridge",
		Firmware:     "0.0.1",
	})
	root.A.Id = 1

	return root.A
}
