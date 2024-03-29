package main

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/brutella/hap"
	"github.com/brutella/hap/log"

	"github.com/urfave/cli/v2"

	"github.com/redgoose/daikin-skyport"
    "github.com/cloudkucooland/HomeKitBridges/DaikinOneHKBridge"

)

func main() {
	var dir, file string
	var debug bool

	app := cli.App{
		Name:  "daikin homekit bridge",
		Usage: "server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "dir",
				Value:       "/var/db/HomeKitBridges/Daikin",
				Usage:       "configuration directory",
				Destination: &dir,
			},
			&cli.StringFlag{
				Name:        "config",
				Value:       "daikin.json",
				Usage:       "configuration file",
				Destination: &file,
			},
			&cli.BoolFlag{
				Name:        "debug",
				Value:       false,
				Usage:       "enable debug",
				Destination: &debug,
			},
		},
		Action: func(c *cli.Context) error {
			if debug {
				log.Debug.Enable()
			}

			fulldir, err := filepath.Abs(dir)
			if err != nil {
				log.Info.Panic("unable to get config directory", dir)
			}
			cfd := filepath.Join(fulldir, file)
			conf, err := dhkb.LoadConfig(cfd)
			if err != nil {
				log.Info.Panic(err.Error())
			}

			// start the daikin logic
			d := daikin.New(conf.Email, conf.Password)
			devices, err := d.GetDevices()
			if err != nil {
				log.Info.Panic(err.Error())
			}

			// if we want to be smart, we can add each and every, now just use the last since I only have one
			var device *daikin.Device
			for _, d := range *devices {
				log.Info.Printf("%+v", d)
				device = &d
			}

			// build the HAP device
			thermostat := dhkb.NewDaikinOne(d, device)

			// add thermostat to HomeKit server
			s, err := hap.NewServer(hap.NewFsStore(fulldir), thermostat.A)
			if err != nil {
				log.Info.Panic(err)

			}

			ctx, cancel := context.WithCancel(context.Background())

			// update the thermostat with data from Daikin cloud -- move this into the devices.go file, should be per device
			go func(ctx context.Context, thermostat *dhkb.DaikinAccessory) {
				ticker := time.NewTicker(180 * time.Second)
				defer ticker.Stop()

				for {
					select {
					case <-ticker.C:
						dhkb.Update(thermostat, d)
					case <-ctx.Done():
						return
					}
				}
			}(ctx, thermostat)

			// serve HomeKit
			go func(ctx context.Context) {
				s.ListenAndServe(ctx)
			}(ctx)

			// wait for signal to shut down
			sigch := make(chan os.Signal, 3)
			signal.Notify(sigch, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGHUP, os.Interrupt)

			// wait until shutdown signal sent
			sig := <-sigch

			log.Info.Printf("shutdown requested by signal: %s", sig)
			cancel()

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Info.Panic(err)
	}
}
