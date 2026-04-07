package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/cloudkucooland/konnectedhkbridge"

	"github.com/brutella/hap"
	"github.com/brutella/hap/log"

	"github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Name:  "konnected-homekit",
		Usage: "HomeKit Bridge for Konnected.io devices",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "dir",
				Value: "/var/db/HomeKitBridges/Konnected",
				Usage: "configuration directory",
			},
			&cli.StringFlag{
				Name:  "config",
				Value: "khkb.json",
				Usage: "configuration file",
			},
			&cli.BoolFlag{
				Name:  "debug",
				Value: false,
				Usage: "enable debug logging",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Bool("debug") {
				log.Debug.Enable()
			}

			dir := cmd.String("dir")
			file := cmd.String("config")

			fulldir, err := filepath.Abs(dir)
			if err != nil {
				return err
			}

			cfd := filepath.Join(fulldir, file)
			conf, err := konnectedkhbridge.LoadConfig(cfd)
			if err != nil {
				return fmt.Errorf("could not initialize config: %w", err)
			}

			var wg sync.WaitGroup

			// Start HTTP service for Konnected hardware callbacks
			wg.Go(func() {
				konnectedkhbridge.HTTPServer(ctx, conf.ListenAddr)
			})

			statePath := filepath.Join(fulldir, "state.json")
			stateManager := konnectedkhbridge.NewPersistentState(statePath)
			if err := stateManager.Load(); err != nil {
				log.Info.Printf("Warning: could not load state: %v", err)
			}

			// Initialize and provision devices
			devices, err := konnectedkhbridge.Startup(ctx, conf, stateManager)
			if err != nil {
				return err
			}

			// Setup HomeKit Server
			s, err := hap.NewServer(hap.NewFsStore(fulldir), devices[0], devices[1:]...)
			if err != nil {
				return err
			}

			if conf.Pin != "" {
				s.Pin = conf.Pin
			}

			// Start HomeKit server
			wg.Go(func() {
				if err := s.ListenAndServe(ctx); err != nil {
					log.Info.Println("HAP server stopped:", err)
				}
			})

			log.Info.Println("Bridge is running. Press Ctrl+C to terminate.")

			<-ctx.Done()

			log.Info.Println("Shutting down services...")
			wg.Wait()
			return nil
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Info.Fatal(err)
	}
}
