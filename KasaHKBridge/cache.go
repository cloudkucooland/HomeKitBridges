package kasahkbridge

import (
	"encoding/json"
	"net"
	"os"
	"path/filepath"

	"github.com/brutella/hap/log"
	"github.com/cloudkucooland/go-kasa"
)

const cachefilename = "startupcache.json"

func SaveCache(path string) error {
	startupcache := make(map[string]kasa.Sysinfo)

	fp := filepath.Join(path, cachefilename)
	cache, err := os.OpenFile(fp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if err != nil {
		log.Info.Printf("unable to open file for startup cache: %s", err.Error())
		return err
	}
	defer cache.Close()

	encoder := json.NewEncoder(cache)
	for id, k := range kasas {
		startupcache[id] = k.sysinfo()
	}

	if err := encoder.Encode(startupcache); err != nil {
		log.Info.Printf("unable to encode startup cache: %s", err.Error())
		return err
	}
	return nil
}

func loadCache(path string) error {
	fp := filepath.Join(path, cachefilename)
	cache, err := os.ReadFile(fp)
	if err != nil {
		log.Info.Printf("unable to open file for startup cache: %s", err.Error())
		return err
	}

	startupcache := make(map[string]kasa.Sysinfo)

	if err := json.Unmarshal(cache, &startupcache); err != nil {
		log.Info.Printf("startup cache unmarshal failed: %s", err.Error())
		return err
	}

	// startup with junk IP addresses, will be fixed on first query
	var ip byte = 1
	for id, k := range startupcache {
		ipd := net.IPv4(169, 254, 199, ip)

		kd := kasa.KasaDevice{}
		kd.GetSysinfo.Sysinfo = k

		switch kd.GetSysinfo.Sysinfo.Model {
		case "HS103(US)":
			kasas[id] = NewHS103(kd, ipd)
		case "HS200(US)", "HS210(US)":
			kasas[id] = NewHS200(kd, ipd)
		case "HS220(US)":
			kasas[id] = NewHS220(kd, ipd)
		case "KP115(US)":
			kasas[id] = NewKP115(kd, ipd)
		case "KP303(US)":
			kasas[id] = NewKP303(kd, ipd)
		case "HS300(US)":
			kasas[id] = NewHS300(kd, ipd)
		}
		ip++
	}
	return nil
}
