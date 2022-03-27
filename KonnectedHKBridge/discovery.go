package konnectedkhbridge

import (
	"strings"

	"github.com/brutella/hap/log"

	"github.com/huin/goupnp"
	"github.com/huin/goupnp/ssdp"
)

// this is dumb for multiple devices - pass in an []mac and populate them all as/if found
// with a single upnp request, return map[mac]ip
func discover(mac string) string {
	log.Info.Printf("Discovering Konnected")

	devices, err := goupnp.DiscoverDevices(ssdp.SSDPAll)
	if err != nil {
		log.Info.Println("discovery failed: ", err.Error())
		return ""
	}

	for _, device := range devices {
		if !strings.Contains(device.USN, "konnected") {
			continue
		}

		if device.Root == nil || device.USN == "" || len(device.USN) < 43 {
			log.Info.Printf("odd format : %+v", device)
			continue
		}

		// only the last 6 of the mac are in the USN
		if device.USN[36:42] != mac[6:] {
			log.Info.Printf("mac does not match, continuing: %s %s", device.USN[36:42], mac[6:])
			continue
		}

		log.Info.Printf("found: %s for %s", device.Root.URLBaseStr, mac)
		s := strings.Split(device.Root.URLBaseStr, "/")
		if len(s) < 3 {
			log.Info.Printf("found device, wrong url format")
			continue
		}
		return s[2]
	}
	return ""
}
