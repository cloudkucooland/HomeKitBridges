package konnectedkhbridge

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
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
	var conf Config

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
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	if err := conf.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	// if not statically configured, auto-discover
	if conf.ListenAddr == "" {
		conf.ListenAddr = getListenAddress()
	}

	log.Info.Printf("using config: %+v", conf)
	return &conf, nil
}

func getListenAddress() string {
	log.Info.Println("discovering local listen address")

	// Try UDP method
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err == nil {
		defer conn.Close()
		localAddr := conn.LocalAddr().(*net.UDPAddr)
		ip := localAddr.IP.String()
		addr := fmt.Sprintf("%s:8999", ip)
		log.Info.Printf("detected via UDP: %s", addr)
		return addr
	}

	// Fallback to interface scan
	ifaces, err := net.Interfaces()
	if err != nil {
		log.Info.Printf("failed to list interfaces: %v", err)
		return "0.0.0.0:8999"
	}

	for _, iface := range ifaces {
		// skip down or loopback
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok || ipNet.IP.IsLoopback() {
				continue
			}

			ip := ipNet.IP.To4()
			if ip == nil {
				continue
			}

			addr := fmt.Sprintf("%s:8999", ip.String())
			log.Info.Printf("detected via interface: %s (%s)", addr, iface.Name)
			return addr
		}
	}

	log.Info.Println("no suitable IP found, falling back to 0.0.0.0")
	return "0.0.0.0:8999"
}
