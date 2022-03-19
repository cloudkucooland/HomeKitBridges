package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/cloudkucooland/HomeKitBridges/OnkyoHKBridge"

	"github.com/brutella/hap"
	"github.com/brutella/hap/log"

	"github.com/urfave/cli/v2"
)

func main() {
	var dir, file string
	var debug bool

	app := cli.App{
		Name:  "onkyo homekit bridge",
		Usage: "server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "dir",
				Value:       "/var/run/HomeKitBridges/Onkyo",
				Usage:       "configuration directory",
				Destination: &dir,
			},
			&cli.StringFlag{
				Name:        "config",
				Value:       "ohkb.json",
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
			confFile, err := os.Open(cfd)
			if err != nil {
				log.Info.Panic("unable to open config: ", cfd)
			}
			raw, err := ioutil.ReadAll(confFile)
			if err != nil {
				log.Info.Panic(err)
			}
			confFile.Close()

			conf := config{}
			err = json.Unmarshal(raw, &conf)
			if err != nil {
				log.Info.Panic(err, string(raw))
			}
			// defaults
			if conf.Poll == 0 {
				conf.Poll = 60
			}

			ctx, cancel := context.WithCancel(context.Background())

			// discover & configure onkyo device
			receiver, err := ohkb.DiscoverOnkyo(ctx, conf.IP)

			// start onkyo background puller -- move to DiscoverOnkyo
			go func() {
				t := time.Tick(time.Duration(conf.Poll) * time.Second)
				select {
				case <-ctx.Done():
					return
				case <-t:
					receiver.Update()
				}
			}()

			s, err := hap.NewServer(hap.NewFsStore(fulldir), receiver.A)
			if err != nil {
				log.Info.Panic(err)
			}
			if conf.Pin != "" {
				s.Pin = conf.Pin
			}

			// await context cancel
			go s.ListenAndServe(ctx)

			// wait for signal to shut down
			sigch := make(chan os.Signal, 3)
			signal.Notify(sigch, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGHUP, os.Interrupt)

			// loop until signal sent
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

type config struct {
	IP   string // IP address or "" for auto-discover
	Poll uint16 // seconds between status polls
	Pin  string // setup PIN
}
