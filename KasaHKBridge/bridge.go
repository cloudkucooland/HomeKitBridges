package kasahkbridge

import (
	"github.com/brutella/hap/accessory"
	"github.com/brutella/hap/characteristic"
	// "github.com/brutella/hap/log"
	"github.com/brutella/hap/service"
)

// Bridge is used by the startup to build the generic bridge type on which all other devices hang
func Bridge() *accessory.A {
	root := accessory.NewBridge(accessory.Info{
		Name:         "KasaHomekitBridge",
		SerialNumber: "1101",
		Manufacturer: "cloudkucooland",
		Model:        "KasaHomekitbridge",
		Firmware:     "0.0.1",
	})
	root.A.Id = 1

	// root.A.AddS(NewKHKBsvc().S)

	return root.A
}

type KHKBsvc struct {
	*service.S

	Name     *characteristic.Name
	PollRate *characteristic.Int
}

func NewKHKBsvc() *KHKBsvc {
	svc := KHKBsvc{}
	svc.S = service.New("FF")

	svc.Name = characteristic.NewName()
	svc.AddC(svc.Name.C)
	svc.Name.SetValue("Kasa Homekit Bridge Config")

	svc.PollRate = characteristic.NewInt("FF")
	svc.AddC(svc.PollRate.C)
	svc.PollRate.SetMinValue(10)
	svc.PollRate.SetMaxValue(600)
	svc.PollRate.SetValue(60)

	return &svc
}
