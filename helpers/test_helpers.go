package helpers

import (
	"encoding/json"
	"os"
)

type RouterApiConfig struct {
	Address string `json:"address"`
	Port    uint16 `json:"port"`
}

func LoadConfig() RouterApiConfig {

	loadedConfig := loadConfigJsonFromPath()

	if loadedConfig.Address == "" {
		panic("missing configuration 'address'")
	}

	if loadedConfig.Port == 0 {
		panic("missing configuration 'port'")
	}

	return loadedConfig
}

func loadConfigJsonFromPath() RouterApiConfig {
	var config RouterApiConfig

	path := configPath()

	configFile, err := os.Open(path)
	if err != nil {
		panic(err)
	}

	decoder := json.NewDecoder(configFile)
	err = decoder.Decode(&config)
	if err != nil {
		panic(err)
	}

	return config
}

func configPath() string {
	path := os.Getenv("ROUTER_API_CONFIG")
	if path == "" {
		panic("Must set $ROUTER_API_CONFIG to point to an integration config .json file.")
	}

	return path
}
