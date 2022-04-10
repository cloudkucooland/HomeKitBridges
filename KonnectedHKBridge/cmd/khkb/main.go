package main

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/cloudkucooland/konnectedhkbridge"

	"github.com/brutella/hap"
	"github.com/brutella/hap/log"

	"github.com/urfave/cli/v2"
)

// TODO dump cli and use the native flag type
func main() {
	var dir, file string
	var debug bool

	app := cli.App{
		Name:  "onkyo homekit bridge",
		Usage: "server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "dir",
				Value:       "/var/db/HomeKitBridges/Konnected",
				Usage:       "configuration directory",
				Destination: &dir,
			},
			&cli.StringFlag{
				Name:        "config",
				Value:       "khkb.json",
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
			conf, err := konnectedkhbridge.LoadConfig(cfd)
			if err != nil {
				log.Info.Panic(err.Error())
			}

			ctx, cancel := context.WithCancel(context.Background())
			var wg sync.WaitGroup

			// respond to HTTP requests from the konnected devices (even before they are configured, so they can be bootstrapped)
			wg.Add(1)
			go func(ctx context.Context) {
				defer wg.Done()
				konnectedkhbridge.HTTPServer(ctx, conf.ListenAddr)
			}(ctx)

			// discover & provision the devices
			devices, err := konnectedkhbridge.Startup(ctx, conf)
			if err != nil {
				log.Info.Panic(err)
			}

			// push the config into HomeKit server
			s, err := hap.NewServer(hap.NewFsStore(fulldir), devices[0], devices[1:]...)
			if err != nil {
				log.Info.Panic(err)
			}
			if conf.Pin != "" {
				s.Pin = conf.Pin
			}

			// serve HomeKit
			wg.Add(1)
			go func(ctx context.Context) {
				defer wg.Done()
				s.ListenAndServe(ctx)
			}(ctx)

			// wait for signal to shut down
			sigch := make(chan os.Signal, 3)
			signal.Notify(sigch, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGHUP, os.Interrupt)

			// wait until shutdown signal sent
			sig := <-sigch

			log.Info.Printf("shutdown requested by signal: %s", sig)
			cancel() // closes homekit and konnected services

			wg.Wait()
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Info.Panic(err)
	}
}
