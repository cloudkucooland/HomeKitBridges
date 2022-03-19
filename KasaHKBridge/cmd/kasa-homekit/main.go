package main

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/cloudkucooland/HomeKitBridges/KasaHKBridge"

	"github.com/brutella/hap"
	"github.com/brutella/hap/log"

	"github.com/urfave/cli/v2"
)

// TODO dump cli and use the native flag type
func main() {
	var dir string

	app := cli.App{
		Name:  "Kasa homekit bridge",
		Usage: "server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "dir",
				Value:       "/var/run/HomeKitBridges/Kasa",
				Usage:       "configuration directory",
				Destination: &dir,
			},
		},
		Action: func(c *cli.Context) error {
			fulldir, err := filepath.Abs(dir)
			if err != nil {
				log.Info.Panic("unable to get config directory", dir)
			}

			// wait for signal to shut down
			sigch := make(chan os.Signal, 3)
			signal.Notify(sigch, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGHUP, os.Interrupt)

			// start the UDP listener before anything else
			listenctx, listencancel := context.WithCancel(context.Background())
			var lwg sync.WaitGroup
			lwg.Add(1)
			go func(listenctx context.Context) {
				defer lwg.Done()
				kasahkbridge.Listener(listenctx)
			}(listenctx)

			// discover & provision the devices
			refresh := make(chan bool)
			if err = kasahkbridge.Startup(listenctx, refresh); err != nil {
				log.Info.Panic(err)
			}

			// does not change over time
			bridge := kasahkbridge.Bridge()
			var hapwg sync.WaitGroup

		DONE:
			for {
				hapctx, hapcancel := context.WithCancel(context.Background())
				devices := kasahkbridge.Devices()
				log.Info.Printf("serving %d kasa devices", len(devices))
				s, err := hap.NewServer(hap.NewFsStore(fulldir), bridge, devices...)
				if err != nil {
					log.Info.Panic(err)
				}

				// serve HomeKit
				lwg.Add(1)
				hapwg.Add(1)
				go func(hapctx context.Context) {
					defer lwg.Done()
					defer hapwg.Done()
					s.ListenAndServe(hapctx)
				}(hapctx)

				select {
				case <-refresh:
					log.Info.Printf("new device discovered, restarting")
					hapcancel()
					hapwg.Wait()
					// loop back around, getting updated device list
				case <-listenctx.Done():
					log.Info.Printf("shutdown: context canceled")
					hapcancel()
					hapwg.Wait()
					break DONE
				case sig := <-sigch:
					log.Info.Printf("shutdown requested by signal: %s", sig)
					hapcancel()
					hapwg.Wait()
					break DONE
				}
			}
			listencancel()

			lwg.Wait()
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Info.Panic(err)
	}
}
