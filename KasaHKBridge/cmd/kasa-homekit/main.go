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

	"github.com/vishvananda/netlink"
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
				Value:       "/var/db/HomeKitBridges/Kasa",
				Usage:       "configuration directory",
				Destination: &dir,
			},
		},
		Action: func(c *cli.Context) error {
			fulldir, err := filepath.Abs(dir)
			if err != nil {
				log.Info.Panic("unable to get config directory", dir)
			}

			// listen for interface status changes
			var linkstatuschan = make(chan netlink.LinkUpdate, 5)
			var disconnectchan = make(chan struct{})
			if err := netlink.LinkSubscribe(linkstatuschan, disconnectchan); err != nil {
				log.Info.Panic(err.Error())
			}

			// wait for signal to shut down
			sigch := make(chan os.Signal, 3)
			signal.Notify(sigch, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGHUP, os.Interrupt)

			refresh := make(chan bool, 3)

			// start the UDP listener before anything else
			listenctx, listencancel := context.WithCancel(context.Background())
			var listenwaitgroup sync.WaitGroup
			listenwaitgroup.Go(func() {
				kasahkbridge.Listener(listenctx, refresh)
			})

			// discover & provision the devices
			if err = kasahkbridge.SetBroadcasts(); err != nil {
				log.Info.Panic(err)
			}
			if err = kasahkbridge.Startup(listenctx, refresh); err != nil {
				log.Info.Panic(err)
			}

			// does not change over time
			bridge := kasahkbridge.Bridge()
			var hapwaitgroup sync.WaitGroup

		DONE:
			for {
				hapctx, hapcancel := context.WithCancel(context.Background())
				devices := kasahkbridge.Devices()
				// kasahkbridge.BridgeAddState()
				log.Info.Printf("serving %d kasa devices", len(devices))
				hapserver, err := hap.NewServer(hap.NewFsStore(fulldir), bridge, devices...)
				if err != nil {
					log.Info.Panic(err)
				}

				// serve HomeKit
				hapwaitgroup.Go(func() {
					hapserver.ListenAndServe(hapctx)
				})

				select {
				case <-refresh:
					// TODO if less than 3 seconds since last restart, just wait?
					log.Info.Printf("new device discovered, restarting")
					hapcancel()
					hapwaitgroup.Wait()
					// loop back around, getting updated device list
				case <-listenctx.Done():
					log.Info.Printf("shutdown: context canceled")
					hapcancel()
					hapwaitgroup.Wait()
					break DONE
				case sig := <-sigch:
					log.Info.Printf("shutdown requested by signal: %s", sig)
					hapcancel()
					hapwaitgroup.Wait()
					break DONE
				case <-linkstatuschan:
					log.Info.Printf("interface change, updating broadcast addresses")
					_ = kasahkbridge.SetBroadcasts()
					hapcancel()
					hapwaitgroup.Wait()
					// loop back around
				}
			}
			close(disconnectchan)
			listencancel()
			listenwaitgroup.Wait()
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Info.Panic(err)
	}
}
