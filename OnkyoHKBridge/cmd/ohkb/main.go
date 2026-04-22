package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/cloudkucooland/HomeKitBridges/OnkyoHKBridge"

	"github.com/brutella/hap"
	"github.com/brutella/hap/log"

	"github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Name:  "onkyo-homekit",
		Usage: "HomeKit Bridge for Onkyo receivers",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "dir",
				Value: "/var/db/HomeKitBridges/Onkyo",
				Usage: "configuration directory",
			},
			&cli.StringFlag{
				Name:  "config",
				Value: "ohkb.json",
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
			conf, err := ohkb.LoadConfig(cfd)
			if err != nil {
				return fmt.Errorf("could not initialize config: %w", err)
			}

			// discover & configure onkyo device
			receiver, err := ohkb.DiscoverOnkyo(ctx, conf.IP, conf.Poll)
			if err != nil {
				return err
			}

			s, err := hap.NewServer(hap.NewFsStore(fulldir), receiver.A)
			if err != nil {
				return err
			}

			if conf.Pin != "" {
				s.Pin = conf.Pin
			}

			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := s.ListenAndServe(ctx); err != nil {
					log.Info.Println("HAP server stopped:", err)
				}
			}()

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
