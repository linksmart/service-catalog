// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"

	"code.linksmart.eu/com/go-sec/authz"
	"code.linksmart.eu/sc/service-catalog/catalog"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	ID           string           `json:"id"`
	Description  string           `json:"description"`
	DNSSDEnabled bool             `json:"dnssdEnabled"`
	Storage      StorageConf      `json:"storage"`
	HTTP         HTTPConf         `json:"http"`
	MQTT         catalog.MQTTConf `json:"mqtt"`
	Auth         ValidatorConf    `json:"auth"`
}

func (c *Config) validate() error {

	err := c.Storage.validate()
	if err != nil {
		return err
	}

	err = c.HTTP.validate()
	if err != nil {
		return err
	}

	err = c.MQTT.Validate()
	if err != nil {
		return err
	}

	if c.Auth.Enabled {
		// Validate ticket validator config
		err = c.Auth.validate()
		if err != nil {
			return err
		}
	}

	return nil
}

func loadConfig(confPath string) (*Config, error) {
	file, err := ioutil.ReadFile(confPath)
	if err != nil {
		return nil, err
	}

	var config Config
	err = json.Unmarshal(file, &config)
	if err != nil {
		return nil, err
	}

	// Override loaded values with environment variables
	err = envconfig.Process("sc", &config)
	if err != nil {
		return nil, err
	}

	if err = config.validate(); err != nil {
		return nil, err
	}
	return &config, nil
}

type StorageConf struct {
	Type string `json:"type"`
	DSN  string `json:"dsn"`
}

func (c StorageConf) validate() error {
	if !catalog.SupportedBackends[c.Type] {
		return fmt.Errorf("storage: unsupported backend")
	}
	_, err := url.Parse(c.DSN)
	if err != nil {
		return fmt.Errorf("storage: DSN should be a valid URL: %v", err)
	}
	return nil
}

type HTTPConf struct {
	BindAddr string `json:"bindAddr"`
	BindPort int    `json:"bindPort"`
}

func (c HTTPConf) validate() error {
	if c.BindAddr == "" {
		return fmt.Errorf("http: bindAddr not defined")
	}
	if c.BindPort == 0 {
		return fmt.Errorf("http: bindPort not defined")
	}
	return nil
}

// Ticket Validator Config
type ValidatorConf struct {
	// Auth switch
	Enabled bool `json:"enabled"`
	// Authentication provider name
	Provider string `json:"provider"`
	// Authentication provider URL
	ProviderURL string `json:"providerURL"`
	// Service ID
	ServiceID string `json:"serviceID"`
	// Basic Authentication switch
	BasicEnabled bool `json:"basicEnabled"`
	// Authorization config
	Authz *authz.Conf `json:"authorization"`
}

func (c ValidatorConf) validate() error {

	// Validate Provider
	if c.Provider == "" {
		return errors.New("Ticket Validator: Auth provider name (provider) is not specified.")
	}

	// Validate ProviderURL
	if c.ProviderURL == "" {
		return errors.New("Ticket Validator: Auth provider URL (providerURL) is not specified.")
	}
	_, err := url.Parse(c.ProviderURL)
	if err != nil {
		return errors.New("Ticket Validator: Auth provider URL (providerURL) is invalid: " + err.Error())
	}

	// Validate ServiceID
	if c.ServiceID == "" {
		return errors.New("Ticket Validator: Auth Service ID (serviceID) is not specified.")
	}

	// Validate Authorization
	if c.Authz != nil {
		if err := c.Authz.Validate(); err != nil {
			return err
		}
	}

	return nil
}
