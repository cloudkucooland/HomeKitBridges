package konnectedkhbridge

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"github.com/brutella/hap/characteristic"
	"github.com/brutella/hap/log"

	// use go-chi since it is what hap uses, no need for multiple
	"github.com/go-chi/chi"
)

const jsonOK = `{ "status": "OK" }`

// handler listens for Konnected devices and respond appropriately
// if the board doesn't get a 200 in response, it retries, and failing several retries, it reboots
// we will just say OK no matter what for now
func handler(w http.ResponseWriter, r *http.Request) {
	device := chi.URLParam(r, "device")
	if device == "" {
		log.Info.Printf("device unset: %+v", r)
		fmt.Fprint(w, jsonOK)
		return
	}

	k := chooseKonnected(device)
	if k == nil {
		log.Info.Printf("Unknown device: %s %+v", device, r)
		fmt.Fprint(w, jsonOK)
		return
	}

	// verify token, if set in local config
	if k.password != "" {
		sentToken := r.Header.Get("Authorization")
		if sentToken == "" {
			log.Info.Printf("Authorization token not sent")
			// http.Error(w, `{ "status": "bad" }`, http.StatusForbidden)
			fmt.Fprint(w, jsonOK)
			return
		}
		if sentToken[7:] != k.password {
			log.Info.Printf("Authorization token invalid")
			// http.Error(w, `{ "status": "bad" }`, http.StatusForbidden)
			fmt.Fprint(w, jsonOK)
			return
		}
	}

	// if remote addr differes from expected, update?
	remHost, _, _ := net.SplitHostPort(r.RemoteAddr)
	setHost, setPort, _ := net.SplitHostPort(k.ip)
	if remHost != setHost {
		log.Info.Printf("need to update IP address to  %s from %s (port: %s)", remHost, setHost, setPort)
		k.ip = fmt.Sprintf("%s:%s", remHost, setPort)

		// if the device wasn't discovered on boot, the port will be garbage, re-discover
		if k.A.Info.FirmwareRevision.Value() == "bootstrap" {
			ip := discover(device)
			if ip != "" {
				log.Info.Printf("rediscovery: (%s) got: (%s)", device, ip)
				k.ip = ip
				if err := k.getStatusAndUpdate(); err != nil {
					log.Info.Println(err.Error())
				}
			} else {
				log.Info.Println("rediscovery failed: still in bootstrap mode, try rebooting the hardware")
			}
		}
	}

	jBlob, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Info.Printf("konnected: unable to read update: %s", err.Error())
		// http.Error(w, `{ "status": "bad" }`, http.StatusInternalServerError)
		fmt.Fprint(w, jsonOK)
		return
	}
	// if konnected provisioned with a trailing / on the url..
	if string(jBlob) == "" {
		log.Info.Printf("konnected: sent empty message")
		// acknowledge the notice so it doesn't retransmit
		fmt.Fprint(w, jsonOK)
		// trigger a manual pull
		if err := k.getStatusAndUpdate(); err != nil {
			log.Info.Println(err.Error())
		}
		return
	}

	var p sensor
	// log.Info.Printf("sent from %s %s: %s", device, r.RemoteAddr, string(jBlob))
	err = json.Unmarshal(jBlob, &p)
	if err != nil {
		log.Info.Printf("konnected: unable to understand update")
		// http.Error(w, `{ "status": "bad" }`, http.StatusNotAcceptable)
		fmt.Fprint(w, jsonOK)
		return
	}

	// tell homekit about the change and run any actions
	// move this logic into the device e.g. k.Update(p.Pin, p.State)
	if svc, ok := k.pins[p.Pin]; ok {
		switch svc.(type) {
		case *KonnectedMotionSensor:
			svc.(*KonnectedMotionSensor).MotionDetected.SetValue(p.State == 1)
			switch k.SecuritySystem.SecuritySystemCurrentState.Value() {
			case characteristic.SecuritySystemCurrentStateDisarmed:
				// nothing
				// log.Info.Printf("%s: %s", svc.(*KonnectedMotionSensor).Name.Value(), p.State)
			case characteristic.SecuritySystemCurrentStateStayArm:
				// nothing
				// log.Info.Printf("%s: %s", svc.(*KonnectedMotionSensor).Name.Value(), p.State)
			default:
				// for now we won't do anything since the cats trip it
				log.Info.Printf("motion detected while alarm armed; pin: %d", p.Pin)
				k.doorchirps()
			}
		case *KonnectedContactSensor:
			svc.(*KonnectedContactSensor).ContactSensorState.SetValue(int(p.State))
			switch k.SecuritySystem.SecuritySystemCurrentState.Value() {
			case characteristic.SecuritySystemCurrentStateAwayArm:
				k.countdownAlarm()
			case characteristic.SecuritySystemCurrentStateNightArm:
				k.instantAlarm()
			case characteristic.SecuritySystemCurrentStateStayArm:
				// nothing for now
				k.doorchirps()
			default:
				k.doorchirps()
			}
			state := "opened"
			if p.State == 0 {
				state = "closed"
			}
			log.Info.Printf("%s: %s", svc.(*KonnectedContactSensor).Name.Value(), state)
		case *KonnectedBuzzer: // not used
			log.Info.Printf("%s: %s", svc.(*KonnectedBuzzer).Switch.Name.Value(), p.State)
			// svc.(*KonnectedBuzzer).Beeper.SetValue(int(p.State))
		default:
			log.Info.Println("bad type in handler: %+v", svc)
			k.doorchirps()
		}
	}
	fmt.Fprint(w, jsonOK)
}

func HTTPServer(ctx context.Context, addr string) {
	router := chi.NewRouter()
	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		// do something better here
		w.Write([]byte("Konnected HomeKit Bridge"))
	})

	router.Route("/konnected/device/{device}", func(r chi.Router) {
		r.Get("/", handler)
		r.Put("/", handler)
		r.Post("/", handler)
	})

	srv := &http.Server{
		Handler:      router,
		Addr:         addr,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Info.Printf("starting http service at %s", addr)
	go srv.ListenAndServe()
	<-ctx.Done()
	log.Info.Printf("stopping http service")
	srv.Shutdown(context.Background())
}

// for when we support multiple devices
func chooseKonnected(mac string) *Konnected {
	if k, ok := ks[mac]; ok {
		return k
	}

	log.Info.Printf("unknown konnected sent request...")
	// not a known mac, return a bootstrap device
	return nil // crash hard now
}
