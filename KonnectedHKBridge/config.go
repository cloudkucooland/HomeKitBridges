package konnectedkhbridge

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/brutella/hap/log"
)

type Config struct {
	Pin        string // HomeKit setup pin (80899303)
	ListenAddr string // ip:port we listen on for updates from konnected devices (:8999)
	Devices    []Device
}

type Device struct {
	ip       string // not exposed, we always auto-discover
	Mac      string // device identifier for a Konnected
	Password string // token to verify requests from a Konnected
	Zones    []Zone // list of sensors/actuators
}

// exposed in accessory.KonnectedZones
type Zone struct {
	Pin  uint8  `json:"pin"`
	Name string `json:"name"`
	Type string `json:"type"`
	// Actuator actuator `json:"actuator",omitempty`
}

func LoadConfig(filename string) (*Config, error) {
	conf := Config{
		Pin:        "80899303",
		ListenAddr: "192.168.1.2:8999",
	}

	confFile, err := os.Open(filename)
	if err != nil {
		log.Info.Printf("unable to open config %s: using defaults (%+v)", filename, conf)
		return &conf, nil
	}

	raw, err := ioutil.ReadAll(confFile)
	if err != nil {
		log.Info.Printf(err.Error())
		return nil, err
	}
	confFile.Close()
    // log.Info.Printf(string(raw))

	err = json.Unmarshal(raw, &conf)
	if err != nil {
		log.Info.Printf(err.Error(), string(raw))
		return nil, err
	}
	log.Info.Printf("using config: %+v", conf)

	return &conf, nil
}
