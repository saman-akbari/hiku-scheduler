package config

import (
	"bytes"
	"encoding/json"
	"log"
	"os"

	"hiku/proxy"
)

// JSONConfig holds the data configured via a JSON file. This shall be used
// to parse the JSON file and create a proper Config struct that dictates the
// scheduler's behavior.
type JSONConfig struct {
	Host     string   `json:"host"`
	Port     int      `json:"port"`
	Balancer string   `json:"balancer"`
	Workers  []string `json:"workers"`
}

func (c JSONConfig) ToConfig() Config {
	return Config{
		Host:         c.Host,
		Port:         c.Port,
		Balancer:     createBalancerFromConfig(c),
		ReverseProxy: proxy.NewHTTPReverseProxy(),
	}
}

func LoadConfigFromFile(configFilepath string) JSONConfig {
	var config JSONConfig

	file, rfErr := os.ReadFile(configFilepath)
	if rfErr != nil {
		log.Fatalf("Cannot read config file (%s)", rfErr)
	}

	decoder := json.NewDecoder(bytes.NewReader(file))
	jsonErr := decoder.Decode(&config) // Parse json config file
	if jsonErr != nil {
		log.Fatalf("Config file Ill-formed (%s)", jsonErr)
	}

	return config
}
