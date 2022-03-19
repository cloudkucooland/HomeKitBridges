package ohkb

import (
	"context"
	"fmt"
	"strconv"
	// "sync"
	// "time"

	"github.com/cloudkucooland/go-onkyo"

	"github.com/brutella/hap/accessory"
	"github.com/brutella/hap/characteristic"
	"github.com/brutella/hap/log"
)

func DiscoverOnkyo(ctx context.Context, ip string) (*OnkyoReceiver, error) {
	dev, err := eiscp.NewReceiver(ip, true)
	if err != nil {
		log.Info.Printf(err.Error())
		return nil, err
	}

	// we don't ever care about cover art, and can make the first pull fail
	dev.SetNetworkJacketArt(false)
	deets, err := dev.GetDetails()
	if err != nil {
		log.Info.Printf("unable to pull for details: %s", err.Error())
		return nil, err
	}

	info := accessory.Info{
		Manufacturer: deets.Device.Brand,
		Model:        deets.Device.Model,
		SerialNumber: deets.Device.DeviceSerial,
		Firmware:     deets.Device.FirmwareVersion,
		Name:         fmt.Sprintf("%s (%s)", deets.Device.Model, deets.Device.ZoneList.Zone[0].Name),
	}

	a := NewOnkyoReceiver(info, dev)

	// add to HC for GUI
	log.Info.Printf("Discovered %s %s", info.Name, info.Model)

	a.Amp = dev
	a.Television.ConfiguredName.SetValue(info.Name)
	a.AddInputs(deets)
	a.AddZones(deets)

	// set initial power state
	power, err := dev.GetPower()
	if err != nil {
		log.Info.Println(err.Error())
	}
	_, err = dev.GetVolume()
	if err != nil {
		log.Info.Println(err.Error())
	}
	if _, err := dev.GetTempData(); err != nil {
		log.Info.Println(err.Error())
	}

	source, err := dev.GetSource()
	if err != nil {
		log.Info.Println(err.Error())
	} else {
		i, _ := strconv.ParseInt(string(source), 16, 32)
		a.Television.ActiveIdentifier.SetValue(int(i))
		// d.Television.ConfiguredName.SetValue(fmt.Sprintf("%s:%s", a.Info.Name, d.Sources[int(i)]))
	}
	/// NPS does not respond if powered off or not set to SLI network
	a.Television.CurrentMediaState.SetValue(characteristic.CurrentMediaStateUnknown)
	if power && source == eiscp.SrcNetwork {
		a.Amp.GetNetworkPlayStatus()
	}

	// start listener for updates from the onkyo
	go iscpListener(ctx, a)

	return a, nil
}

// we just ask, let the persistentListener process the responses
func (d *OnkyoReceiver) Update() {
	d.Amp.GetTempData()
	d.Amp.GetVolume()
	d.Amp.GetMute()

	power, err := d.Amp.GetPower()
	if err != nil {
		log.Info.Println(err.Error())
		err = nil
	}

	source, err := d.Amp.GetSource()
	if err != nil {
		log.Info.Println(err.Error())
		err = nil
	}

	if power && source == eiscp.SrcNetwork {
		if _, err := d.Amp.GetNetworkPlayStatus(); err != nil {
			log.Info.Println(err.Error())
		}
	}
}
