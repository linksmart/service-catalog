// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package client

import (
	"fmt"
	"time"

	"code.linksmart.eu/com/go-sec/auth/obtainer"
	"code.linksmart.eu/sc/service-catalog/catalog"
)

// RegisterService registers service into a catalog
func RegisterService(endpoint string, service catalog.Service, ticket *obtainer.Client) (*catalog.Service, error) {
	// Configure client
	client, err := NewHTTPClient(endpoint, ticket)
	if err != nil {
		return nil, fmt.Errorf("error creating HTTP client: %s", err)
	}

	updatedService, err := client.Put(&service)
	if err != nil {
		return nil, fmt.Errorf("error PUTing registration: %v", err)
	}

	return updatedService, nil
}

// UnregisterService removes service from a catalog
func UnregisterService(endpoint string, service catalog.Service, ticket *obtainer.Client) error {
	// Configure client
	client, err := NewHTTPClient(endpoint, ticket)
	if err != nil {
		return fmt.Errorf("error creating HTTP client: %s", err)
	}

	err = client.Delete(service.ID)
	if err != nil {
		return fmt.Errorf("error PUTing registration: %v", err)
	}

	return nil
}

// RegisterServiceAndKeepalive registers a service into a catalog and continuously updates it in order to avoid expiry
// endpoint: catalog endpoint. If empty - will be discovered using DNS-SD
// service: service registration
// ticket: set to nil for no auth
func RegisterServiceAndKeepalive(endpoint string, service catalog.Service, ticket *obtainer.Client) (func() error, error) {

	client, err := NewHTTPClient(endpoint, ticket)
	if err != nil {
		return nil, err
	}

	ticker := time.NewTicker(time.Duration(service.TTL/2) * time.Second)
	go func() {
		for ; true; <-ticker.C {
			_, err := client.Put(&service)
			if err != nil {
				logger.Printf("Error updating service registration for %s: %s", service.ID, err)
				continue
			}
			logger.Printf("Updated service registration for %s", service.ID)
		}
	}()

	stop := func() error {
		ticker.Stop()
		client.Delete(service.ID)
		if err != nil {
			logger.Printf("Error removing service registration for %s: %s", service.ID, err)
		}
		return nil
	}

	return stop, nil
}
