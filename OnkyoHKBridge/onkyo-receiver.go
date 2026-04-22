package ohkb

import (
	"strconv"
	"strings"

	"github.com/brutella/hap/accessory"
	"github.com/brutella/hap/characteristic"
	"github.com/brutella/hap/log"
	"github.com/brutella/hap/service"

	"github.com/cloudkucooland/go-onkyo"
)

type OnkyoReceiver struct {
	*accessory.A

	Amp        *eiscp.Device
	Television *OnkyoReceiverSvc
	Speaker    *service.Speaker
	Temp       *service.TemperatureSensor

	// added to Speaker
	VolumeActive *characteristic.Active
	Volume       *characteristic.Volume

	Sources map[int]string
}

func NewOnkyoReceiver(info accessory.Info, dev *eiscp.Device) *OnkyoReceiver {
	acc := OnkyoReceiver{}

	acc.Amp = dev

	acc.A = accessory.New(info, accessory.TypeTelevision)
	acc.Television = NewOnkyoReceiverSvc()
	acc.Speaker = service.NewSpeaker()
	acc.Temp = service.NewTemperatureSensor()

	acc.Television.SleepDiscoveryMode.SetValue(characteristic.SleepDiscoveryModeAlwaysDiscoverable)
	acc.Television.PowerModeSelection.SetValue(characteristic.PowerModeSelectionShow)
	acc.Television.Primary = true
	acc.AddS(acc.Television.S)

	acc.Volume = characteristic.NewVolume()
	acc.Volume.Description = "Master Volume"
	acc.Speaker.AddC(acc.Volume.C)

	acc.VolumeActive = characteristic.NewActive()
	acc.VolumeActive.Description = "Speaker Active"
	acc.VolumeActive.SetValue(characteristic.ActiveActive)
	acc.Speaker.AddC(acc.VolumeActive.C)

	acc.Speaker.Mute.SetValue(false)
	acc.Speaker.Primary = false
	acc.AddS(acc.Speaker.S)

	// Link Speaker to Television for proper iOS volume control
	acc.Television.AddS(acc.Speaker.S)

	acc.Temp.S.Primary = false
	acc.AddS(acc.Temp.S)

	acc.Sources = make(map[int]string)

	acc.Television.On.OnValueRemoteUpdate(func(newstate bool) {
		log.Info.Printf("setting power to %t", newstate)
		_, err := acc.Amp.SetPower(newstate)
		if err != nil {
			log.Info.Println(err.Error())
		}
	})

	acc.Volume.OnValueRemoteUpdate(func(newstate int) {
		log.Info.Printf("setting volume to: %d", newstate)
		_, err := acc.Amp.SetVolume(uint8(newstate))
		if err != nil {
			log.Info.Println(err.Error())
		}
	})

	acc.Speaker.Mute.OnValueRemoteUpdate(func(newstate bool) {
		log.Info.Printf("setting mute to: %t", newstate)
		_, err := acc.Amp.SetMute(newstate)
		if err != nil {
			log.Info.Println(err.Error())
		}
	})

	acc.Television.ActiveIdentifier.OnValueRemoteUpdate(func(newstate int) {
		log.Info.Printf("Setting input to %02X", newstate)
		_, err := acc.Amp.SetSourceByCode(newstate)
		if err != nil {
			log.Info.Println(err.Error())
		}
	})

	acc.Television.RemoteKey.OnValueRemoteUpdate(func(newstate int) {
		handleRemote(&acc, newstate)
	})

	return &acc
}

func (t *OnkyoReceiver) AddZones(nfi *eiscp.NRI) {
	for _, s := range nfi.Device.ZoneList.Zone {
		if s.Name != "Main" && s.Value == "1" {
			log.Info.Printf("discovered zone: %+v", s)
		}
	}
}

func (t *OnkyoReceiver) AddInputs(nfi *eiscp.NRI) {
	for _, s := range nfi.Device.SelectorList.Selector {
		// skip the label
		if s.ID == "80" {
			continue
		}
		log.Info.Printf("adding input source: %+v", s)
		is := service.NewInputSource()

		is.ConfiguredName.SetValue(s.Name)
		inputSourceType := characteristic.InputSourceTypeHdmi
		switch strings.ToUpper(s.ID) {
		case eiscp.SrcCBL: // CBL/SAT
			inputSourceType = characteristic.InputSourceTypeOther
		case eiscp.SrcGame: // GAME
			inputSourceType = characteristic.InputSourceTypeHdmi
		case eiscp.SrcAux1: // AUX
			inputSourceType = characteristic.InputSourceTypeOther
		case eiscp.SrcPC: // PC
			inputSourceType = characteristic.InputSourceTypeOther
		case eiscp.SrcDVD: // BD/DVD
			inputSourceType = characteristic.InputSourceTypeHdmi
		case eiscp.SrcStrm: // STRMBOX
			inputSourceType = characteristic.InputSourceTypeHdmi
		case eiscp.SrcTV: // TV
			inputSourceType = characteristic.InputSourceTypeHdmi
		case eiscp.SrcPhono: // Phono
			inputSourceType = characteristic.InputSourceTypeOther
		case eiscp.SrcCD: // CD
			inputSourceType = characteristic.InputSourceTypeOther
		case eiscp.SrcAM: // AM
			inputSourceType = characteristic.InputSourceTypeTuner
		case eiscp.SrcFM: // FM
			inputSourceType = characteristic.InputSourceTypeTuner
		case eiscp.SrcNetwork: // NET
			inputSourceType = characteristic.InputSourceTypeApplication
		case eiscp.SrcBluetooth: // BLUETOOTH
			inputSourceType = characteristic.InputSourceTypeApplication
		}
		is.InputSourceType.SetValue(inputSourceType)
		is.IsConfigured.SetValue(characteristic.IsConfiguredConfigured)
		is.CurrentVisibilityState.SetValue(characteristic.CurrentVisibilityStateShown)

		i, err := strconv.ParseInt(s.ID, 16, 32)
		if err != nil {
			log.Info.Println(err.Error())
		} else {
			t.Sources[int(i)] = s.Name
		}

		t.AddS(is.S)
		t.Television.AddS(is.S)

		is.IsConfigured.OnValueRemoteUpdate(func(newstate int) {
			log.Info.Printf("%s IsConfigured: %d", is.ConfiguredName.Value(), newstate)
		})
	}
}

type OnkyoReceiverSvc struct {
	*service.S

	On                 *characteristic.On
	Volume             *characteristic.Volume
	StreamingStatus    *characteristic.StreamingStatus
	Active             *characteristic.Active
	ActiveIdentifier   *characteristic.ActiveIdentifier
	ConfiguredName     *characteristic.ConfiguredName
	SleepDiscoveryMode *characteristic.SleepDiscoveryMode
	Brightness         *characteristic.Brightness
	ClosedCaptions     *characteristic.ClosedCaptions
	DisplayOrder       *characteristic.DisplayOrder
	CurrentMediaState  *characteristic.CurrentMediaState
	TargetMediaState   *characteristic.TargetMediaState
	PictureMode        *characteristic.PictureMode
	PowerModeSelection *characteristic.PowerModeSelection
	RemoteKey          *characteristic.RemoteKey
}

func NewOnkyoReceiverSvc() *OnkyoReceiverSvc {
	svc := OnkyoReceiverSvc{}
	svc.S = service.New(service.TypeTelevision)

	svc.On = characteristic.NewOn()
	svc.AddC(svc.On.C)

	svc.Volume = characteristic.NewVolume()
	svc.AddC(svc.Volume.C)

	svc.StreamingStatus = characteristic.NewStreamingStatus()
	svc.AddC(svc.StreamingStatus.C)

	svc.Active = characteristic.NewActive()
	svc.AddC(svc.Active.C)

	svc.ActiveIdentifier = characteristic.NewActiveIdentifier()
	svc.AddC(svc.ActiveIdentifier.C)

	svc.ConfiguredName = characteristic.NewConfiguredName()
	svc.AddC(svc.ConfiguredName.C)

	svc.SleepDiscoveryMode = characteristic.NewSleepDiscoveryMode()
	svc.AddC(svc.SleepDiscoveryMode.C)

	svc.Brightness = characteristic.NewBrightness()
	svc.AddC(svc.Brightness.C)

	svc.ClosedCaptions = characteristic.NewClosedCaptions()
	svc.AddC(svc.ClosedCaptions.C)

	svc.DisplayOrder = characteristic.NewDisplayOrder()
	svc.AddC(svc.DisplayOrder.C)

	svc.CurrentMediaState = characteristic.NewCurrentMediaState()
	svc.CurrentMediaState.SetValue(characteristic.CurrentMediaStatePlay)
	svc.AddC(svc.CurrentMediaState.C)

	svc.TargetMediaState = characteristic.NewTargetMediaState()
	svc.AddC(svc.TargetMediaState.C)

	svc.PictureMode = characteristic.NewPictureMode()
	svc.AddC(svc.PictureMode.C)

	svc.PowerModeSelection = characteristic.NewPowerModeSelection()
	svc.AddC(svc.PowerModeSelection.C)
	svc.PowerModeSelection.OnValueRemoteUpdate(func(newstate int) {
		log.Info.Printf("OnkyoReceiver: HC requested PowerModeSelection: %d", newstate)
		svc.PowerModeSelection.SetValue(newstate)
	})

	svc.RemoteKey = characteristic.NewRemoteKey()
	svc.AddC(svc.RemoteKey.C)
	svc.RemoteKey.SetValue(characteristic.RemoteKeyInfo)

	return &svc
}
