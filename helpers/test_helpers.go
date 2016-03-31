package helpers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"

	"github.com/cloudfoundry-incubator/bbs"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/nu7hatch/gouuid"
)

type RoutingConfig struct {
	helpers.Config
	RoutingApiUrl     string       `json:"-"` //"-" is used for ignoring field
	Addresses         []string     `json:"addresses"`
	BBSAddress        string       `json:"bbs_api_url,omitempty"`
	BBSClientCertFile string       `json:"bbs_client_cert,omitempty"`
	BBSClientKeyFile  string       `json:"bbs_client_key,omitempty"`
	BBSCACertFile     string       `json:"bbs_ca_cert,omitempty"`
	BBSRequireSSL     bool         `json:"bbs_require_ssl"`
	OAuth             *OAuthConfig `json:"oauth"`
}

type OAuthConfig struct {
	TokenEndpoint            string `json:"token_endpoint"`
	ClientName               string `json:"client_name"`
	ClientSecret             string `json:"client_secret"`
	Port                     int    `json:"port"`
	SkipOAuthTLSVerification bool   `json:"skip_oauth_tls_verification"`
}

const (
	DEFAULT_BBS_API_URL = "http://bbs.service.cf.internal:8889"
)

func LoadConfig() RoutingConfig {
	loadedConfig := loadConfigJsonFromPath()
	loadedConfig.Config = helpers.LoadConfig()

	if loadedConfig.OAuth == nil {
		panic("missing configuration oauth")
	}

	if len(loadedConfig.Addresses) == 0 {
		panic("missing configuration 'addresses'")
	}

	if loadedConfig.AppsDomain == "" {
		panic("missing configuration apps_domain")
	}

	if loadedConfig.ApiEndpoint == "" {
		panic("missing configuration api")
	}

	if loadedConfig.BBSAddress == "" {
		loadedConfig.BBSAddress = DEFAULT_BBS_API_URL
	}

	if loadedConfig.BBSRequireSSL &&
		(loadedConfig.BBSClientCertFile == "" || loadedConfig.BBSClientKeyFile == "" || loadedConfig.BBSCACertFile == "") {
		panic("ssl enabled: missing configuration for mutual auth")
	}

	loadedConfig.RoutingApiUrl = fmt.Sprintf("%s%s", loadedConfig.Protocol(), loadedConfig.ApiEndpoint)

	return loadedConfig
}

func loadConfigJsonFromPath() RoutingConfig {
	var config RoutingConfig

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
	path := os.Getenv("CONFIG")
	if path == "" {
		panic("Must set $CONFIG to point to an integration config .json file.")
	}

	return path
}

func (c RoutingConfig) Protocol() string {
	if c.UseHttp {
		return "http://"
	} else {
		return "https://"
	}
}

func RandomName() string {
	guid, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}

	return guid.String()
}

func GetBbsClient(routerApiConfig RoutingConfig) (bbs.Client, error) {
	bbsUrl, err := url.Parse(routerApiConfig.BBSAddress)
	if err != nil {
		return nil, err
	}

	var bbsClient bbs.Client

	if bbsUrl.Scheme == "http" {
		bbsClient = bbs.NewClient(bbsUrl.String())
	} else if bbsUrl.Scheme == "https" {
		bbsClient, err = bbs.NewSecureClient(bbsUrl.String(), routerApiConfig.BBSCACertFile,
			routerApiConfig.BBSClientCertFile, routerApiConfig.BBSClientKeyFile)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, errors.New("invalid-scheme-in-bbs-address")
	}
	return bbsClient, nil
}
