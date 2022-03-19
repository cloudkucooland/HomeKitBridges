package ohkb

import (
	"context"
	"fmt"
	"strconv"

	"github.com/cloudkucooland/go-onkyo"

	// "github.com/brutella/hap/accessory"
	"github.com/brutella/hap/characteristic"
	"github.com/brutella/hap/log"
)

func iscpListener(ctx context.Context, o *OnkyoReceiver) {
	// select ctx.Done() o.Amp.Responses

	for resp := range o.Amp.Responses {
		v := resp.Parsed
		switch resp.Command {
		case "PWR":
			if o.Television.On.Value() != v.(bool) {
				p := characteristic.ActiveInactive
				if v.(bool) {
					p = characteristic.ActiveActive
				}
				o.Television.On.SetValue(v.(bool))
				o.Television.Active.SetValue(p)
				o.VolumeActive.SetValue(p) // speaker
			}
		case "MVL":
			if int(v.(uint8)) != o.Television.Volume.Value() {
				o.Television.Volume.SetValue(int(v.(uint8)))
			}
		case "AMT":
			if v.(bool) != o.Speaker.Mute.Value() {
				o.Speaker.Mute.SetValue(v.(bool))
			}
		case "TPD":
			if float64(v.(uint8)) != o.Temp.CurrentTemperature.Value() {
				// log.Info.Printf("temp: %dC\n", v.(uint8))
				o.Temp.CurrentTemperature.SetValue(float64(v.(uint8)))
			}
		case "SLI":
			// resp.Response is ID, resp.Parsed is name
			i, _ := strconv.ParseInt(string(resp.Response), 16, 32)
			if int(i) != o.Television.ActiveIdentifier.Value() {
				log.Info.Println("setting source from listener")
				o.Television.ActiveIdentifier.SetValue(int(i))
				o.Television.ConfiguredName.SetValue(fmt.Sprintf("%s:%s", o.Info.Name, o.Sources[int(i)]))
			}
		case "NRI":
			log.Info.Println("Onkyo Details pulled")
		case "NTM":
			// ignore
		case "NFI":
			// ignore
		case "NJA":
			// ignore
		case "NLS":
			// log.Info.Printf("%+v", eiscp.Menu)
		case "NLT":
			log.Info.Printf("%+v", eiscp.Menu)
		case "UPD":
			log.Info.Printf("Update info: %s\n", resp.Parsed)
		case "NST":
			nps := v.(*eiscp.NetworkPlayStatus)
			log.Info.Printf("setting CurrentMediaState to %s", nps.State)
			switch nps.State {
			case "Play":
				if o.Television.CurrentMediaState.Value() != characteristic.CurrentMediaStatePlay {
					log.Info.Println("NST: Play")
					o.Television.CurrentMediaState.SetValue(characteristic.CurrentMediaStatePlay)
					// o.Television.Active.SetValue(characteristic.ActiveActive)
					// o.VolumeActive.SetValue(characteristic.ActiveActive)
				}
			case "Stop":
				if o.Television.CurrentMediaState.Value() != characteristic.CurrentMediaStateStop {
					o.Television.CurrentMediaState.SetValue(characteristic.CurrentMediaStateStop)
					log.Info.Println("NST: Stop")
					// o.Television.Active.SetValue(characteristic.ActiveInactive)
					// o.VolumeActive.SetValue(characteristic.ActiveInactive)
				}
			case "Pause":
				if o.Television.CurrentMediaState.Value() != characteristic.CurrentMediaStatePause {
					o.Television.CurrentMediaState.SetValue(characteristic.CurrentMediaStatePause)
					log.Info.Println("NST: Pause")
					// o.Television.Active.SetValue(characteristic.ActiveInactive)
					// o.VolumeActive.SetValue(characteristic.ActiveInactive)
				}
			default:
				log.Info.Println("Unknown media state")
				o.Television.CurrentMediaState.SetValue(characteristic.CurrentMediaStateUnknown)
				log.Info.Println("NST: unknown")
				// o.Television.Active.SetValue(characteristic.ActiveInactive)
				// o.VolumeActive.SetValue(characteristic.ActiveInactive)
			}
		case "MOT":
			log.Info.Printf("Music Optimizer: %t\n", v.(bool))
		case "DIM":
			log.Info.Printf("Dimmer: %s\n", v.(string))
		case "RAS":
			log.Info.Printf("Cinema Filter: %t\n", v.(bool))
		case "PCT":
			log.Info.Printf("Phase Control: %t\n", v.(bool))
		case "NDS":
			log.Info.Printf("Network: %+v\n", v.(*eiscp.NetworkStatus))
		default:
			log.Info.Printf("unhandled response on listener: %s %+v\n", resp.Command, v)
		}
	}
}
