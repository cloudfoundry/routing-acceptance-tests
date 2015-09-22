package helpers

import (
	"encoding/json"
	"os"
)

type RouterApiConfig struct {
	Address           string `json:"address"`
	Port              uint16 `json:"port"`
	BBSAddress        string `json:"bbs_api_url,omitempty"`
	BBSClientCertFile string `json:"bbs_client_cert,omitempty"`
	BBSClientKeyFile  string `json:"bbs_client_key,omitempty"`
	BBSCACertFile     string `json:"bbs_ca_cert,omitempty"`
	BBSRequireSSL     bool   `json:"bbs_require_ssl"`
}

const (
	DEFAULT_BBS_API_URL = "http://bbs.service.cf.internal:8889"
)

func LoadConfig() RouterApiConfig {

	loadedConfig := loadConfigJsonFromPath()

	if loadedConfig.Address == "" {
		panic("missing configuration 'address'")
	}

	if loadedConfig.Port == 0 {
		panic("missing configuration 'port'")
	}

	if loadedConfig.BBSAddress == "" {
		loadedConfig.BBSAddress = DEFAULT_BBS_API_URL
	}

	if loadedConfig.BBSRequireSSL &&
		(loadedConfig.BBSClientCertFile == "" || loadedConfig.BBSClientKeyFile == "" || loadedConfig.BBSCACertFile == "") {
		panic("ssl enabled: missing configuration for mutual auth")
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
