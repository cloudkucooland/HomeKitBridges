package kasahkbridge

import (
	// "github.com/brutella/hap/accessory"
	"github.com/brutella/hap/characteristic"
	// "github.com/brutella/hap/log"
	"github.com/brutella/hap/service"
)

type bstate struct {
	*service.S

	Reachable           *characteristic.Reachable
	LinkQuality         *characteristic.LinkQuality
	AccessoryIdentifier *characteristic.AccessoryIdentifier
	Category            *characteristic.Category
}

func NewBridgingState() *bstate {
	bs := &bstate{}
	bs.S = service.New("62")

	bs.Reachable = characteristic.NewReachable()
	bs.Reachable.Description = "Reachable"
	bs.S.AddC(bs.Reachable.C)

	bs.LinkQuality = characteristic.NewLinkQuality()
	bs.LinkQuality.Description = "Link Quality"
	bs.S.AddC(bs.LinkQuality.C)

	bs.AccessoryIdentifier = characteristic.NewAccessoryIdentifier()
	bs.AccessoryIdentifier.Description = "AccessoryIdentifier"
	bs.S.AddC(bs.AccessoryIdentifier.C)

	bs.Category = characteristic.NewCategory() // 1-16
	bs.Category.Description = "Category"
	bs.S.AddC(bs.Category.C)

	return bs
}
