package konnectedkhbridge

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/brutella/hap/log"

	// use go-chi since it is what hap uses, no need for multiple
	"github.com/go-chi/chi/v5"
)

var StrictHTTP bool

const jsonOK = `{ "status": "OK" }`

// handler listens for Konnected devices and respond appropriately
// if the board doesn't get a 200 in response, it retries, and failing several retries, it reboots
// we will just say OK no matter what for now
func handler(w http.ResponseWriter, r *http.Request) {
	device := chi.URLParam(r, "device")
	if device == "" {
		respondError(w, http.StatusBadRequest, "missing device", StrictHTTP)
		return
	}

	k := chooseKonnected(device)
	if k == nil {
		respondError(w, http.StatusNotFound, "unknown device", StrictHTTP)
		return
	}

	// verify token, if set in local config
	if k.password != "" {
		sentToken := r.Header.Get("Authorization")
		if sentToken == "" {
			respondError(w, http.StatusUnauthorized, "Authorization token not sent", StrictHTTP)
			return
		}
		if len(sentToken) < 7 || sentToken[:7] != "Bearer " || sentToken[7:] != k.password {
			respondError(w, http.StatusForbidden, "Authorization token invalid", StrictHTTP)
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
					respondError(w, http.StatusBadRequest, err.Error(), StrictHTTP)
					return
				}
			} else {
				respondError(w, http.StatusBadRequest, "rediscovery failed: still in bootstrap mode. Reboot konnected hardware", StrictHTTP)
				return
			}
		}
	}

	jBlob, err := io.ReadAll(r.Body)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "unable to read update", StrictHTTP)
		return
	}
	// if konnected provisioned with a trailing / on the url..
	if len(jBlob) == 0 {
		log.Info.Printf("konnected: sent empty message")
		// acknowledge the notice so it doesn't retransmit
		respondOK(w)
		// trigger a manual pull
		go k.getStatusAndUpdate()
		return
	}

	var p sensor
	err = json.Unmarshal(jBlob, &p)
	if err != nil {
		log.Info.Printf("Received: %s", string(jBlob))
		respondError(w, http.StatusNotAcceptable, "unable to parse update", StrictHTTP)
		return
	}

	// tell homekit about the change and run any actions
	k.HandleUpdate(p)
	respondOK(w)
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
	if err := srv.Shutdown(context.Background()); err != nil {
		log.Info.Println("HTTP server shutdown error:", err)
	}
}

// for when we support multiple devices
func chooseKonnected(mac string) *Konnected {
	if k, ok := ks[mac]; ok {
		return k
	}

	log.Info.Printf("unknown konnected sent request...")
	// not a known mac, return a bootstrap device
	return nil
}

func respondOK(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, jsonOK)
}

func respondError(w http.ResponseWriter, code int, msg string, strict bool) {
	if strict {
		http.Error(w, msg, code)
		return
	}

	log.Info.Printf("handler error (suppressed %d): %s", code, msg)
	respondOK(w)
}
