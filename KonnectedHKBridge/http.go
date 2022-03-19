package konnectedkhbridge

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/brutella/hap/characteristic"
	"github.com/brutella/hap/log"

	"github.com/gorilla/mux"
)

// handler is registered with the HTTP platform
// it listens for Konnected devices and respond appropriately
// if the board doesn't get a 200 in response, it retries, and failing several retries, it reboots
// we will just say OK no matter what for now
func handler(w http.ResponseWriter, r *http.Request) {
	log.Info.Printf("konnected: %+v", r)

	vars := mux.Vars(r)
	device := vars["device"]

	k := chooseKonnected(device) // if remote addr differes from expected, update?

	log.Info.Printf("konnected state for device (%s / %s)", r.RemoteAddr, device)

	// verify token, if set in local config
	if k.password != "" {
		sentToken := r.Header.Get("Authorization")
		if sentToken == "" {
			log.Info.Printf("Authorization token not sent")
			// http.Error(w, `{ "status": "bad" }`, http.StatusForbidden)
			fmt.Fprint(w, `{ "status": "OK" }`)
			return
		}
		if sentToken[7:] != k.password {
			log.Info.Printf("Authorization token invalid")
			// http.Error(w, `{ "status": "bad" }`, http.StatusForbidden)
			fmt.Fprint(w, `{ "status": "OK" }`)
			return
		}
	}

	jBlob, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Info.Printf("konnected: unable to read update")
		// http.Error(w, `{ "status": "bad" }`, http.StatusInternalServerError)
		fmt.Fprint(w, `{ "status": "OK" }`)
		return
	}
	// if konnected provisioned with a trailing / on the url..
	if string(jBlob) == "" {
		log.Info.Printf("konnected: sent empty message")
		// acknowledge the notice so it doesn't retransmit
		fmt.Fprint(w, `{ "status": "OK" }`)
		// trigger a manual pull
		err := k.getStatusAndUpdate()
		if err != nil {
			log.Info.Println(err.Error())
		}
		return
	}

	var p sensor
	// log.Info.Printf("sent from %+v: %s", a.Name, string(jBlob))
	err = json.Unmarshal(jBlob, &p)
	if err != nil {
		log.Info.Printf("konnected: unable to understand update")
		// http.Error(w, `{ "status": "bad" }`, http.StatusNotAcceptable)
		fmt.Fprint(w, `{ "status": "OK" }`)
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
			case characteristic.SecuritySystemCurrentStateStayArm:
				// k.doorchirps()
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
			svc.(*KonnectedBuzzer).Active.SetValue(int(p.State))
		default:
			log.Info.Println("bad type in handler: %+v", svc)
			k.doorchirps()
		}
	}
	fmt.Fprint(w, `{ "status": "OK" }`)
}

func HTTPServer(ctx context.Context, addr string) {
	router := mux.NewRouter()

	router.HandleFunc("", handler)
	router.HandleFunc("{device}", handler)
	router.HandleFunc("/{device}", handler)
	router.HandleFunc("/konnected", handler)
	router.HandleFunc("/konnected/{device}", handler)
	router.HandleFunc("/konnected/device/{device}", handler)
	router.NotFoundHandler = http.HandlerFunc(handler)

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
