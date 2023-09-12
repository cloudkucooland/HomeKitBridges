package main

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/brutella/hap/log"
)

type Config struct {
	APIKey     string
	Integrator string
	Email      string
}

func loadConfig(filename string) (*Config, error) {
	conf := Config{
		APIKey: "nothing",
		Email:  "nobody@loopback.edu",
	}

	confFile, err := os.Open(filename)
	if err != nil {
		log.Info.Printf("%s\nunable to open config %s: using defaults\n%+v", err.Error(), filename, conf)
		return &conf, err
	}
	defer confFile.Close()

	raw, err := ioutil.ReadAll(confFile)
	if err != nil {
		log.Info.Printf("%s\nunable to read config %s: using defaults\n%+v", err.Error(), filename, conf)
		return &conf, err
	}

	err = json.Unmarshal(raw, &conf)
	if err != nil {
		log.Info.Printf("%s\nunable to parse config %s: using defaults\nraw: %s\n%+v", err.Error(), filename, string(raw), conf)
		return &conf, err
	}

	return &conf, nil
}
