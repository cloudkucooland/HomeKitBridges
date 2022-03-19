package ohkb

import (
	"github.com/brutella/hap/characteristic"
	"github.com/brutella/hap/log"
)

func handleRemote(o *OnkyoReceiver, newstate int) {
	d := o.Amp
	switch newstate {
	case characteristic.RemoteKeyRewind:
		if err := d.SetOnly("NTC", "REW"); err != nil {
			log.Info.Println(err)
		}
	case characteristic.RemoteKeyFastForward:
		if err := d.SetOnly("NTC", "FF"); err != nil {
			log.Info.Println(err)
		}
	case characteristic.RemoteKeyExit:
		if err := d.SetOnly("NTC", "RETURN"); err != nil {
			log.Info.Println(err)
		}
	case characteristic.RemoteKeyPlayPause:
		if err := d.SetOnly("NTC", "P/P"); err != nil {
			log.Info.Println(err)
		}
	case characteristic.RemoteKeyInfo:
		if err := d.SetOnly("NTC", "TOP"); err != nil {
			log.Info.Println(err)
		}
	case characteristic.RemoteKeyNextTrack:
		if err := d.SetOnly("NTC", "TRUP"); err != nil {
			log.Info.Println(err)
		}
	case characteristic.RemoteKeyPrevTrack:
		if err := d.SetOnly("NTC", "TRDN"); err != nil {
			log.Info.Println(err)
		}
	case characteristic.RemoteKeyArrowUp:
		if err := d.SetOnly("NTC", "UP"); err != nil {
			log.Info.Println(err)
		}
	case characteristic.RemoteKeyArrowDown:
		if err := d.SetOnly("NTC", "DOWN"); err != nil {
			log.Info.Println(err)
		}
	case characteristic.RemoteKeyArrowLeft:
		if err := d.SetOnly("NTC", "LEFT"); err != nil {
			log.Info.Println(err)
		}
	case characteristic.RemoteKeyArrowRight:
		if err := d.SetOnly("NTC", "RIGHT"); err != nil {
			log.Info.Println(err)
		}
	case characteristic.RemoteKeySelect:
		if err := d.SetOnly("NTC", "SELECT"); err != nil {
			log.Info.Println(err)
		}
	case characteristic.RemoteKeyBack:
		if err := d.SetOnly("NTC", "TOP"); err != nil {
			log.Info.Println(err)
		}
	}

}
