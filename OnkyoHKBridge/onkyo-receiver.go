package ohkb

import (
	"github.com/brutella/hap/accessory"
	"github.com/brutella/hap/characteristic"
	"github.com/brutella/hap/log"
	"github.com/brutella/hap/service"
	"strconv"
	"strings"

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

	// these break things if added
	// VolumeControlType *characteristic.VolumeControlType
	// VolumeSelector    *characteristic.VolumeSelector // bad things happen

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
	acc.Volume.OnValueRemoteUpdate(func(newstate int) {
		log.Info.Printf("OnkyoReceiver: HC requested speaker volume: %d", newstate)
	})
	acc.Speaker.AddC(acc.Volume.C)

	// acc.VolumeControlType = characteristic.NewVolumeControlType()
	// acc.VolumeControlType.Description = "VolumeControlType"
	// acc.VolumeControlType.SetValue(characteristic.VolumeControlTypeAbsolute)
	// this breaks things
	// acc.Speaker.AddC(acc.VolumeControlType.C)
	// acc.VolumeSelector = characteristic.NewVolumeSelector()
	// acc.VolumeSelector.Description = "VolumeSelector"
	// this break things
	// acc.Speaker.AddC(acc.VolumeSelector.C)

	acc.VolumeActive = characteristic.NewActive()
	acc.VolumeActive.Description = "Speaker Active"
	acc.VolumeActive.SetValue(characteristic.ActiveActive)
	acc.Volume.OnValueRemoteUpdate(func(newstate int) {
		log.Info.Printf("OnkyoReceiver: HC requested speaker active: %d", newstate)
	})
	acc.Speaker.AddC(acc.VolumeActive.C)

	acc.Speaker.Mute.SetValue(false)
	acc.Speaker.Mute.OnValueRemoteUpdate(func(newstate bool) {
		log.Info.Printf("OnkyoReceiver: HC requested speaker mute: %t", newstate)
	})
	acc.Speaker.AddC(acc.VolumeActive.C)
	acc.Speaker.Primary = false
	acc.AddS(acc.Speaker.S)
	// this should be required but breaks things
	// acc.Television.AddS(acc.Speaker.S) // breaks
	acc.Speaker.AddS(acc.Television.S) // does not break

	acc.Temp.S.Primary = false
	acc.AddS(acc.Temp.S)
	acc.Television.AddS(acc.Temp.S) // does this do anything? it doesn't seem to hurt...
	acc.Speaker.AddS(acc.Temp.S)    // does this do anything? it doesn't seem to hurt...

	acc.Sources = make(map[int]string)

	/* acc.A.OnIdentify(func() {
		log.Info.Printf("identify called for [%s]: %+v", acc.Name, acc.A)
		for _, service := range acc.A.GetServices() {
			log.Info.Printf("service: %+v", service)
			for _, char := range service.GetCharacteristics() {
				log.Info.Printf("characteristic : %+v", char)
			}
		}
	}) */

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

	/* acc.VolumeSelector.OnValueRemoteUpdate(func(newstate int) {
		log.Info.Printf("set volumeselector: %d", newstate)
	}) */

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

// doesn't do anything yet
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
		is.ConfiguredName.Description = "Name"
		is.ConfiguredName.SetValue(s.Name)
		is.ConfiguredName.Description = "ConfiguredName"
		inputSourceType := characteristic.InputSourceTypeHdmi
		// inputDeviceType := characteristic.InputDeviceTypeAudioSystem
		switch strings.ToUpper(s.ID) {
		case eiscp.SrcCBL: // CBL/SAT
			inputSourceType = characteristic.InputSourceTypeOther
			// inputDeviceType = characteristic.InputDeviceTypeAudioSystem
		case eiscp.SrcGame: // GAME
			inputSourceType = characteristic.InputSourceTypeHdmi
			// inputDeviceType = characteristic.InputDeviceTypeTv
		case eiscp.SrcAux1: // AUX
			inputSourceType = characteristic.InputSourceTypeOther
			// inputDeviceType = characteristic.InputDeviceTypeAudioSystem
		case eiscp.SrcPC: // PC
			inputSourceType = characteristic.InputSourceTypeOther
			// inputDeviceType = characteristic.InputDeviceTypeAudioSystem
		case eiscp.SrcDVD: // BD/DVD
			inputSourceType = characteristic.InputSourceTypeHdmi
			// inputDeviceType = characteristic.InputDeviceTypePlayback
		case eiscp.SrcStrm: // STRMBOX
			inputSourceType = characteristic.InputSourceTypeHdmi
			// inputDeviceType = characteristic.InputDeviceTypeTv
		case eiscp.SrcTV: // TV
			inputSourceType = characteristic.InputSourceTypeHdmi
			// inputDeviceType = characteristic.InputDeviceTypeTv
		case eiscp.SrcPhono: // Phono
			inputSourceType = characteristic.InputSourceTypeOther
			// inputDeviceType = characteristic.InputDeviceTypeAudioSystem
		case eiscp.SrcCD: // CD
			inputSourceType = characteristic.InputSourceTypeOther
			// inputDeviceType = characteristic.InputDeviceTypeAudioSystem
		case eiscp.SrcAM: // AM
			inputSourceType = characteristic.InputSourceTypeTuner
			// inputDeviceType = characteristic.InputDeviceTypeTuner
		case eiscp.SrcFM: // FM
			inputSourceType = characteristic.InputSourceTypeTuner
			// inputDeviceType = characteristic.InputDeviceTypeTuner
		case eiscp.SrcNetwork: // NET
			inputSourceType = characteristic.InputSourceTypeApplication
			// inputDeviceType = characteristic.InputDeviceTypePlayback
		case eiscp.SrcBluetooth: // BLUETOOTH
			inputSourceType = characteristic.InputSourceTypeApplication
			// inputDeviceType = characteristic.InputDeviceTypeAudioSystem
		}
		is.InputSourceType.SetValue(inputSourceType)
		is.InputSourceType.Description = "InputSourceType"
		is.IsConfigured.SetValue(characteristic.IsConfiguredConfigured)
		is.IsConfigured.Description = "IsConfigured"
		is.CurrentVisibilityState.SetValue(characteristic.CurrentVisibilityStateShown)
		is.CurrentVisibilityState.Description = "CurrentVisibilityState"

		// optional
		i, err := strconv.ParseInt(s.ID, 16, 32)
		if err != nil {
			log.Info.Println(err.Error())
		} else {
			// is.Identifier.SetValue(int(i))
			// is.Identifier.Description = "Identifier"
			t.Sources[int(i)] = s.Name
		}
		// is.InputDeviceType.SetValue(inputDeviceType)
		// is.InputDeviceType.Description = "InputDeviceType"
		// is.TargetVisibilityState.SetValue(characteristic.TargetVisibilityStateHidden)
		// is.TargetVisibilityState.Description = "TargetVisibilityState"

		// yes, both are required
		t.AddS(is.S)
		t.Television.AddS(is.S)

		// is.TargetVisibilityState.OnValueRemoteUpdate(func(newstate int) {
		// log.Info.Printf("%s TargetVisibilityState: %d", is.Name.GetValue(), newstate)
		// is.TargetVisibilityState.SetValue(newstate)  // not saved, but fine for now
		// is.CurrentVisibilityState.SetValue(newstate) // not saved, but fine for now
		// })
		is.IsConfigured.OnValueRemoteUpdate(func(newstate int) {
			log.Info.Printf("%s IsConfigured: %d", is.ConfiguredName.Value(), newstate)
		})
		// is.Identifier.OnValueRemoteUpdate(func(newstate int) {
		// log.Info.Printf("%s Identifier: %d", is.Name.GetValue(), newstate)
		// })
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
	svc.On.OnValueRemoteUpdate(func(newstate bool) {
		log.Info.Printf("OnkyoReceiver: HC requested On: %t", newstate)
	})

	svc.Volume = characteristic.NewVolume()
	svc.AddC(svc.Volume.C)
	svc.Volume.OnValueRemoteUpdate(func(newstate int) {
		log.Info.Printf("OnkyoReceiver: HC requested television volume: %d", newstate)
	})

	svc.StreamingStatus = characteristic.NewStreamingStatus()
	svc.AddC(svc.StreamingStatus.C)
	svc.StreamingStatus.OnValueRemoteUpdate(func(newstate []byte) {
		log.Info.Printf("OnkyoReceiver: HC requested StreamingStatus: %d", string(newstate))
	})

	svc.Active = characteristic.NewActive()
	svc.AddC(svc.Active.C)
	svc.Active.OnValueRemoteUpdate(func(newstate int) {
		log.Info.Printf("OnkyoReceiver: HC requested Active: %d", newstate)
	})

	svc.ActiveIdentifier = characteristic.NewActiveIdentifier()
	svc.AddC(svc.ActiveIdentifier.C)
	svc.ActiveIdentifier.OnValueRemoteUpdate(func(newstate int) {
		log.Info.Printf("OnkyoReceiver: HC requested ActiveIdentifier: %d", newstate)
	})

	svc.ConfiguredName = characteristic.NewConfiguredName()
	svc.AddC(svc.ConfiguredName.C)
	svc.ConfiguredName.OnValueRemoteUpdate(func(newstate string) {
		log.Info.Printf("OnkyoReceiver: HC requested ConfiguredName: %s", newstate)
	})

	svc.SleepDiscoveryMode = characteristic.NewSleepDiscoveryMode()
	svc.AddC(svc.SleepDiscoveryMode.C)

	svc.Brightness = characteristic.NewBrightness()
	svc.AddC(svc.Brightness.C)
	svc.Brightness.OnValueRemoteUpdate(func(newstate int) {
		log.Info.Printf("OnkyoReceiver: HC requested Brightness: %d", newstate)
	})

	svc.ClosedCaptions = characteristic.NewClosedCaptions()
	svc.AddC(svc.ClosedCaptions.C)
	svc.ClosedCaptions.OnValueRemoteUpdate(func(newstate int) {
		log.Info.Printf("OnkyoReceiver: HC requested ClosedCaptions: %d", newstate)
	})

	svc.DisplayOrder = characteristic.NewDisplayOrder()
	svc.AddC(svc.DisplayOrder.C)
	svc.DisplayOrder.OnValueRemoteUpdate(func(newstate []byte) {
		log.Info.Printf("OnkyoReceiver: HC requested DisplayOrder: %s", string(newstate))
	})

	svc.CurrentMediaState = characteristic.NewCurrentMediaState()
	svc.CurrentMediaState.SetValue(characteristic.CurrentMediaStatePlay)
	svc.AddC(svc.CurrentMediaState.C)

	svc.TargetMediaState = characteristic.NewTargetMediaState()
	svc.AddC(svc.TargetMediaState.C)
	svc.TargetMediaState.OnValueRemoteUpdate(func(newstate int) {
		log.Info.Printf("OnkyoReceiver: HC requested TargetMediaState: %d", newstate)
	})

	svc.PictureMode = characteristic.NewPictureMode()
	svc.AddC(svc.PictureMode.C)
	svc.PictureMode.OnValueRemoteUpdate(func(newstate int) {
		log.Info.Printf("OnkyoReceiver: HC requested PictureMode: %d", newstate)
	})

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
