package konnectedkhbridge

import (
	"strings"

	"github.com/brutella/hap/log"
	"github.com/huin/goupnp"
	"github.com/huin/goupnp/ssdp"
)

func discover(mac string) string {
	log.Info.Printf("Discovering Konnected")

	devices, err := goupnp.DiscoverDevices(ssdp.SSDPAll)
	if err != nil {
		return ""
	}

	// XXX TODO make sure the mac is correct
	for _, device := range devices {
		if !strings.Contains(device.USN, "konnected") {
			continue
		}

		log.Info.Printf("found: %s\n", device.Root.URLBaseStr)
		s := strings.Split(device.Root.URLBaseStr, "/")
		if len(s) < 3 {
			log.Info.Printf("found device, wrong url format")
			continue
		}
		return s[2]
	}
	return ""
}
