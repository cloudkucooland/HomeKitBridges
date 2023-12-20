package konnectedkhbridge

import (
	"encoding/json"
	"io"
	"os"

	"github.com/brutella/hap/log"
)

type Config struct {
	Pin        string // HomeKit setup pin (80899303)
	ListenAddr string // ip:port we listen on for updates from konnected devices (192.168.1.2:8999)
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
	Name string `json:"name"`
	Type string `json:"type"`
	// Actuator actuator `json:"actuator",omitempty`
	Pin uint8 `json:"pin"`
}

func LoadConfig(filename string) (*Config, error) {
	conf := Config{
		Pin:        "80899303",
		ListenAddr: "",
	}

	confFile, err := os.Open(filename)
	if err != nil {
		log.Info.Printf("%s\nunable to open config %s: using defaults\n%+v", err.Error(), filename, conf)
		return &conf, err
	}
	defer confFile.Close()

	raw, err := io.ReadAll(confFile)
	if err != nil {
		log.Info.Printf("%s\nunable to read config %s: using defaults\n%+v", err.Error(), filename, conf)
		return &conf, err
	}

	err = json.Unmarshal(raw, &conf)
	if err != nil {
		log.Info.Printf("%s\nunable to parse config %s: using defaults\nraw: %s\n%+v", err.Error(), filename, string(raw), conf)
		return &conf, err
	}

	// if not statically configured, auto-discover
	if conf.ListenAddr == "" {
		conf.ListenAddr = getListenAddress()
	}

	log.Info.Printf("using config: %+v", conf)

	return &conf, nil
}

// XXX todo
func getListenAddress() string {
	log.Info.Println("discovering local listen address")
	return "192.168.1.2:8999"
}
