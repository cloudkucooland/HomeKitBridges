package ohkb

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/brutella/hap/log"
)

type Config struct {
	IP   string `json:"ip"`   // IP address or "" for auto-discover
	Poll uint16 `json:"poll"` // seconds between status polls
	Pin  string `json:"pin"`  // setup PIN
}

func LoadConfig(filename string) (*Config, error) {
	conf := Config{
		Poll: 60,
	}

	confFile, err := os.Open(filename)
	if err != nil {
		log.Info.Printf("%s\nunable to open config %s: using defaults\n%+v", err.Error(), filename, conf)
		return &conf, err
	}
	defer confFile.Close()

	raw, err := io.ReadAll(confFile)
	if err != nil {
		log.Info.Printf("%s\nunable to read config %s: using defaults\n%+v", err.Error(), filename, conf)
		return &conf, err
	}

	err = json.Unmarshal(raw, &conf)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return &conf, nil
}
