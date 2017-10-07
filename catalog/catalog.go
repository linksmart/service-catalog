// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package catalog

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

// Structs

// Service is a service entry in the catalog
type Service struct {
	ID           string                 `json:"id"`
	Description  string                 `json:"description"`
	ExternalDocs []ExternalDoc          `json:"externalDocs"`
	Meta         map[string]interface{} `json:"meta"`
	TTL          uint                   `json:"ttl,omitempty"`
	Created      time.Time              `json:"created"`
	Updated      time.Time              `json:"updated"`
	// Expires is the time when service will be removed from the system (Only when TTL is set)
	Expires *time.Time `json:"expires,omitempty"`
}

// ExternalDoc is an external resource for extended documentation. E.g. OpenAPI specs, Wiki page
type ExternalDoc struct {
	Description string `json:"description"`
	URL         string `json:"url"`
}

// Validates the Service configuration
func (s Service) validate() error {

	if strings.ContainsAny(s.ID, " ") {
		return fmt.Errorf("id must not contain spaces")
	}
	_, err := url.Parse("http://example.com/" + s.ID)
	if err != nil {
		return fmt.Errorf("invalid service id: %v", err)
	}

	for _, ed := range s.ExternalDocs {
		if _, err := url.Parse(ed.URL); err != nil {
			return fmt.Errorf("invalid external doc url: %s", ed.URL)
		}
	}

	return nil
}

// Interfaces

// Storage interface
type Storage interface {
	add(s *Service) error
	get(id string) (*Service, error)
	update(id string, s *Service) error
	delete(id string) error
	list(page, perPage int) ([]Service, int, error)
	total() (int, error)
	Close() error
}

// Listener interface can be used for notification of the catalog updates
// NOTE: Implementations are expected to be thread safe
type Listener interface {
	added(s Service)
	updated(s Service)
	deleted(id string)
}
