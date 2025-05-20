package config

import (
	"encoding/json"
	"os"
)

const filePath = "config.json"

// Config holds the application configuration
type Config struct {
	IPAddress             string
	Port                  int
	CommonAddress         int
	TelemetryCount        int
	TeleindCount          int
	InterrogationInterval int // in seconds

	TelemetryDescriptions map[int]string `json:"telemetry_descriptions"`
	TeleindDescriptions   map[int]string `json:"teleind_descriptions"`
}

// NewConfig creates a new configuration with default values
func NewConfig() *Config {
	return &Config{
		IPAddress:             "127.0.0.1",
		Port:                  2404,
		CommonAddress:         1,
		TelemetryCount:        100,
		TeleindCount:          100,
		InterrogationInterval: 15,

		TelemetryDescriptions: make(map[int]string),
		TeleindDescriptions:   make(map[int]string),
	}
}

// Save persists the configuration
func (c *Config) Save() error {
	fd, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}

	defer fd.Close()

	encoder := json.NewEncoder(fd)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(c)
	if err != nil {
		return err
	}

	return nil
}

func LoadFromDisk() (cfg *Config) {
	cfg = NewConfig()
	fd, err := os.Open(filePath)
	if err != nil {
		return
	}

	defer fd.Close()
	decoder := json.NewDecoder(fd)
	err = decoder.Decode(&cfg)
	if err != nil {
		return
	}
	return
}
