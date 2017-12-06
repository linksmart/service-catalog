// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package catalog

import (
	"fmt"
	"mime"
	"net/url"
	"strings"
	"time"
)

// Structs

// Service is a service entry in the catalog
type Service struct {
	ID          string                 `json:"id"`
	Description string                 `json:"description"`
	APIs        map[string]string      `json:"apis"`
	Docs        []Doc                  `json:"docs"`
	Meta        map[string]interface{} `json:"meta"`
	TTL         uint                   `json:"ttl,omitempty"`
	Created     time.Time              `json:"created"`
	Updated     time.Time              `json:"updated"`
	// Expires is the time when service will be removed from the system (Only when TTL is set)
	Expires *time.Time `json:"expires,omitempty"`
}

// API is representation of service's API
type API struct {
	Protocol string `json:"protocol"`
	URL      string `json:"url"`
}

// Doc is an external resource documenting the service and/or APIs. E.g. OpenAPI specs, Wiki page
type Doc struct {
	Description string   `json:"description"`
	Type        string   `json:"type"`
	URL         string   `json:"url"`
	APIs        []string `json:"apis"`
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

	for _, URL := range s.APIs {
		if _, err := url.Parse(URL); err != nil {
			return fmt.Errorf("invalid API url: %s", URL)
		}
	}

	for _, doc := range s.Docs {
		// if doc.Type == "" {
		// 	return fmt.Errorf("doc type not defined")
		// }
		if _, _, err := mime.ParseMediaType(doc.Type); err != nil {
			return fmt.Errorf("invalid doc MIME type: %s: %s", doc.URL, err)
		}
		if _, err := url.Parse(doc.URL); err != nil {
			return fmt.Errorf("invalid doc url: %s", doc.URL)
		}
		// if len(s.APIs) != 0 {
		// 	for _, api := range doc.APIs {
		// 		if _, found := s.APIs[api]; !found {
		// 			return fmt.Errorf("api name in doc does not match any apis: %s", api)
		// 		}
		// 	}
		// }
	}

	return nil
}

// Error describes an API error (serializable in JSON)
type Error struct {
	// Code is the (http) code of the error
	Code int `json:"code"`
	// Message is the (human-readable) error message
	Message string `json:"message"`
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
