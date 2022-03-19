package konnectedkhbridge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/brutella/hap/accessory"
	"github.com/brutella/hap/characteristic"
	"github.com/brutella/hap/log"
)

var client *http.Client
var disarmed chan (bool)
var ks map[string]*Konnected

type system struct {
	Mac       string     `json:"mac"`
	IP        string     `json:"ip",omitempty`
	Gateway   string     `json:"gw",omitempty`
	Netmask   string     `json:"nm",omitempty`
	Hardware  string     `json:"hwVersion",omitempty`
	RSSI      int8       `json:"rssi",omitempty`
	Software  string     `json:"swVersion",omitempty`
	Port      uint16     `json:"port",omitempty`
	Uptime    uint64     `json:"uptime",omitempty`
	Heap      uint64     `json:"heap",omitempty`
	Settings  settings   `json:"settings"`
	Sensors   []sensor   `json:"sensors"`
	DBSensors []sensor   `json:"ds18b20_sensors"`
	Actuators []actuator `json:"actuators"`
	DHTs      []dht      `json:"dht_sensors"`
}

type settings struct {
	EndpointType string `json:"endpoint_type",omitempty`
	Endpoint     string `json:"endpoint",omitempty`
	Token        string `json:"token",omitempty`
}

type provisiondata struct {
	EndpointType string         `json:"endpoint_type",omitempty`
	Endpoint     string         `json:"endpoint",omitempty`
	Token        string         `json:"token",omitempty`
	Sensors      []provisionpin `json:"sensors"`
	Actuators    []provisionpin `json:"actuators"`
}

type provisionpin struct {
	Pin     uint8 `json:"pin,omitempty"`
	Trigger uint8 `json:"trigger,omitempty"`
}

type sensor struct {
	Pin   uint8 `json:"pin"`
	State uint8 `json:"state"`
	Retry uint8 `json:"retry",omitempty`
}

type actuator struct {
	Pin     uint8 `json:"pin"`
	Trigger uint8 `json:"trigger"`
}

type dht struct {
	Pin  uint8 `json:"pin"`
	Poll uint  `json:"poll_interval"`
}

type command struct {
	Pin       uint8  `json:"pin"`
	State     uint8  `json:"state"`
	Momentary uint16 `json:"state",omitempty`
	Times     uint8  `json:"times",omitempty`
	Pause     uint8  `json:"pause",omitempty`
}

// Startup sets the globals, discovers & loads settings from the from the Konnected devices
// how to rediscover when IP addresses change, without needing to disrupt the HAP service?
func Startup(ctx context.Context, config *Config) ([]*accessory.A, error) {
	disarmed = make(chan bool)

	client = &http.Client{
		Transport: &http.Transport{MaxIdleConns: 5, IdleConnTimeout: 30 * time.Second},
		Timeout:   time.Second * time.Duration(10),
	}

	// our list of devices, indexed by Mac
	ks = make(map[string]*Konnected)

	// the list returned to the caller, used to populate HAP
	var klist []*accessory.A

	for _, d := range config.Devices {
		if d.Mac == "" {
			log.Info.Printf("Mac address required: %+v", d)
			continue
		}

		// do UPnP discovery, looking for a konnected device with this Mac
		d.ip = discover(d.Mac)

		details := &system{
			Mac:      d.Mac,
			IP:       d.ip,
			Hardware: "bootstrap",
			Software: "bootstrap",
			Port:     8999,
			Settings: settings{
				EndpointType: "rest",
				Endpoint:     "http://bootstrap/",
				Token:        "",
			},
		}
		if d.ip != "" {
			var err error
			details, err = getDetails(d.ip)
			if err != nil {
				log.Info.Printf("unable to poll Konnected device: %s; using bootstrap mode. %s", d.ip, err.Error())
			} else {
				log.Info.Printf("fetched: %+v", details)
			}
		} else {
			log.Info.Printf("unable to discover (%s); using bootstrap mode", d.Mac)
		}

		k := NewKonnected(details, &d)
		// before or after NewKonnected?
		// k.provision(details, &config, &d)
		ks[d.Mac] = k
		klist = append(klist, k.A)
	}

	return klist, nil
}

// generic request type if we already have setup the Konnected
func doRequest(method, url string, buf io.Reader) (*[]byte, error) {
	req, err := http.NewRequest(method, url, buf)
	if err != nil {
		return nil, err
	}

	if method == "PUT" {
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Info.Println(err.Error())
		return nil, err
	}
	return &body, nil
}

// getDetails is called before we have a populated *Konnected
func getDetails(ip string) (*system, error) {
	url := fmt.Sprintf("http://%s/status", ip)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return &system{}, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Info.Println(err.Error())
		return &system{}, err
	}

	var s system
	if err := json.Unmarshal(body, &s); err != nil {
		return &system{}, err
	}
	return &s, nil
}

func (k *Konnected) getStatus() (*[]sensor, error) {
	url := fmt.Sprintf("http://%s/device", k.ip)
	body, err := doRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var s []sensor
	if err := json.Unmarshal(*body, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func (k *Konnected) getStatusAndUpdate() error {
	status, err := k.getStatus()
	if err != nil {
		return err
	}

	for _, v := range *status {
		if p, ok := k.pins[v.Pin]; ok {
			switch p.(type) {
			case *KonnectedMotionSensor:
				p.(*KonnectedMotionSensor).MotionDetected.SetValue(v.State == 1)
			case *KonnectedContactSensor:
				if p.(*KonnectedContactSensor).ContactSensorState.Value() != int(v.State) {
					p.(*KonnectedContactSensor).ContactSensorState.SetValue(int(v.State))
				}
			default:
				log.Info.Printf("konnected device not processed: pin %d", v.Pin)
			}
		}
	}
	return nil
}

func Background() {
	go func() {
		for range time.Tick(time.Second * time.Duration(20)) {
			for _, k := range ks {
				err := k.getStatusAndUpdate()
				if err != nil {
					log.Info.Println(err.Error())
				}
			}
		}
	}()
}

func (k *Konnected) beep() {
	if k.SecuritySystem.SecuritySystemCurrentState.Value() !=
		characteristic.SecuritySystemCurrentStateAlarmTriggered {
		k.doBuzz(`"state":1, "momentary":120, "times":2, "pause":55`, characteristic.ActiveInactive)
	} else {
		log.Info.Println("not beeping since in triggered state")
	}
}

func (k *Konnected) doorchirps() {
	if k.SecuritySystem.SecuritySystemCurrentState.Value() !=
		characteristic.SecuritySystemCurrentStateAlarmTriggered {
		k.doBuzz(`"state":1, "momentary":10, "times":5, "pause":30`, characteristic.ActiveInactive)
	} else {
		log.Info.Println("not doing chirps since in triggered state")
	}
}

func (k *Konnected) instantAlarm() {
	k.SecuritySystem.SecuritySystemCurrentState.SetValue(characteristic.SecuritySystemCurrentStateAlarmTriggered)
	log.Info.Println("sending alarm")
	k.doBuzz(`"state":1`, characteristic.ActiveActive)

	// notify noonlight

	go func() {
		select {
		case <-disarmed:
			// cancelAlarm called
			// send all-clear to noonlight
			k.beep()
		case <-time.After(5 * time.Minute):
			k.beep() // no point of ringing for longer
		}
	}()
}

func (k *Konnected) countdownAlarm() {
	log.Info.Println("starting countdown")
	k.SecuritySystem.SecuritySystemCurrentState.SetValue(characteristic.SecuritySystemCurrentStateAlarmTriggered)

	k.doBuzz(`"state":1, "momentary":50, "pause":450`, characteristic.ActiveInactive)

	go func() {
		select {
		case <-disarmed:
			// cancelAlarm called
		case <-time.After(1 * time.Minute):
			k.instantAlarm()
		}
	}()
}

func (a *Konnected) getBuzzerPin() uint8 {
	// TBD do the work...
	return 8
}

func (k *Konnected) getBuzzer() *KonnectedBuzzer {
	pin := k.getBuzzerPin()
	if svc, ok := k.pins[pin]; ok {
		return svc.(*KonnectedBuzzer)
	}
	return nil
}

func (k *Konnected) cancelAlarm() {
	if k.SecuritySystem.SecuritySystemCurrentState.Value() ==
		characteristic.SecuritySystemCurrentStateDisarmed {
		log.Info.Println("not triggered, nothing to cancel")
		return
	}

	k.doBuzz(`"state": 0`, characteristic.ActiveInactive)
	disarmed <- true
	k.SecuritySystem.SecuritySystemCurrentState.SetValue(characteristic.SecuritySystemCurrentStateDisarmed)
}

func (k *Konnected) doBuzz(cmd string, hcstate int) error {
	if buzzer := k.getBuzzer(); buzzer != nil {
		buzzer.Active.SetValue(hcstate)
	}

	pin := k.getBuzzerPin()
	url := fmt.Sprintf("http://%s/device", k.ip)
	fullcmd := fmt.Sprintf("{\"pin\":%d, %s}", pin, cmd)
	_, err := doRequest("PUT", url, bytes.NewBuffer([]byte(fullcmd)))
	if err != nil {
		return err
	}
	return nil
}

// too dangerous to use just yet
func (k *Konnected) provision(s *system, c *Config, d *Device) error {
	// curl -X PUT -H "Content-Type: application/json" -d '{"endpoint_type":"rest", "endpoint":"http://192.168.12.253:8444/konnected", "token":"notyet", "sensors":[{"pin":1},{"pin":2},{"pin":5},{"pin":6},{"pin":7},{"pin":9}], "actuators":[{"pin":8, "trigger": 1}]}' http://192.168.12.186:15301/settings

	if d.Mac == "bootstrap" || d.ip == "" {
		log.Info.Printf("not reprovisioning device in bootstrap mode")
		return nil
	}

	newEndpoint := fmt.Sprintf("http://%s/konnected", c.ListenAddr)
	if s.Settings.Endpoint == newEndpoint {
		return nil
	}

	log.Info.Printf("endpoints differ, reprovisioning: %s / %s", s.Settings.Endpoint, newEndpoint)

	url := fmt.Sprintf("http://%s/settings", d.ip)
	rpd := provisiondata{
		EndpointType: "rest",
		Endpoint:     newEndpoint,
		Token:        d.Password,
	}
	for _, p := range d.Zones {
		switch p.Type {
		case "door", "motion":
			rpd.Sensors = append(rpd.Sensors, provisionpin{Pin: p.Pin})
		case "buzzer":
			rpd.Actuators = append(rpd.Actuators, provisionpin{Pin: p.Pin})
		case "unused":
		default:
			log.Info.Printf("unknown type %+v", p)
		}
	}
	rpd.Actuators = append(rpd.Actuators, provisionpin{Trigger: 1})

	b, err := json.Marshal(rpd)
	if err != nil {
		log.Info.Printf(err.Error())
		return err
	}
	log.Info.Printf("reprovisioning: %s", b)
	result, err := doRequest("PUT", url, bytes.NewReader(b))
	if err != nil {
		log.Info.Printf(err.Error())
		return err
	}
	log.Info.Printf("%s", result)
	return nil
}
