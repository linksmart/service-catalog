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
	ID          string `json:"id"`
	Description string `json:"description"`
	Title       string `json:"title"`
	Type        string `json:"type"`
	APIs        []API  `json:"apis"`
	//Docs        []Doc                  `json:"docs"`
	Meta      map[string]interface{} `json:"meta"`
	Doc       string                 `json:"doc"`
	TTL       uint                   `json:"ttl,omitempty"`
	Created   time.Time              `json:"created"`
	Updated   time.Time              `json:"updated"`
	CreatedBy string                 `json:"createdBy"`
	UpdatedBy string                 `json:"updatedBy"`
	// Expires is the time when service will be removed from the system (Only when TTL is set)
	Expires time.Time `json:"expires,omitempty"`
}

/*// Doc is an external resource documenting the service and/or APIs. E.g. OpenAPI specs, Wiki page
type Doc struct {
	Description string   `json:"description"`
	Type        string   `json:"type"`
	URL         string   `json:"url"`
	APIs        []string `json:"apis"`
}*/

// API - an API (e.g. REST API, MQTT API, etc.) exposed by the service
type API struct {
	ID          string                 `json:"id"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Protocol    string                 `json:"protocol"`
	Endpoint    string                 `json:"endpoint"`
	Spec        Spec                   `json:"spec"`
	Meta        map[string]interface{} `json:"meta"`
}

// API.spec - the complete specification of the interface exposed by the service
// spec in the form of an url to external specification document is preferred, if not present, the 'schema' could be used
// Recommended - Request-response: OpenAPI/Swagger Spec, PubSub: AsyncAPI Spec
type Spec struct {
	Type   string                 `json:"mediaType"`
	Url    string                 `json:"url"`
	Schema map[string]interface{} `json:"schema"`
}

// Validates the Service configuration
func (s Service) validate() error {

	if strings.ContainsAny(s.ID, " ") {
		return fmt.Errorf("service id must not contain spaces")
	}
	_, err := url.Parse("http://example.com/" + s.ID)
	if err != nil {
		return fmt.Errorf("service id is invalid: %v", err)
	}

	if s.Type == "" {
		return fmt.Errorf("service type not defined")
	}
	if strings.ContainsAny(s.Type, " ") {
		return fmt.Errorf("service type must not contain spaces")
	}

	// If a service needs to use the TTL functionality, TTL should be between 1 and 2147483647
	// The appropriately value for TTL should be provided by the service provider based on how critical the availability of his/her service is
	if s.TTL == 0 || s.TTL > MaxServiceTTL {
		return fmt.Errorf("service TTL should be between 1 and 2147483647 (i.e. 1 second to approx. 68 years)")
	}

	// TODO: request payload validations as described below (create an issue to discuss and finalize):
	// mandatory: type (done), title, apis[x].title, apis[x].protocol?, apis[x].endpoint?, apis[x].spec?

	for _, API := range s.APIs {
		if _, err := url.Parse(API.Endpoint); err != nil {
			return fmt.Errorf("invalid service API endpoint: %s", API.Endpoint)
		}
	}

	for _, API := range s.APIs {
		// if doc.Type == "" {
		// 	return fmt.Errorf("doc type not defined")
		// }
		/*if _, _, err := mime.ParseMediaType(API.Spec.Type); err != nil {
			return fmt.Errorf("invalid API spec MIME type: %s: %s", API.Spec.Type, err)
		}*/
		if _, err := url.Parse(API.Spec.Url); err != nil {
			return fmt.Errorf("invalid API spec url: %s", API.Spec.Url)
		}
		// if len(s.APIs) != 0 {
		// 	for _, api := range doc.APIs {
		// 		if _, found := s.APIs[api]; !found {
		// 			return fmt.Errorf("service API name in doc does not match any apis: %s", api)
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
	iterator() <-chan *Service
	Close() error
}

// Listener interface can be used for notification of the catalog updates
// NOTE: Implementations are expected to be thread safe
type Listener interface {
	added(s Service)
	updated(s Service)
	deleted(s Service)
}
