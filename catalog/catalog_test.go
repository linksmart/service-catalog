// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package catalog

import (
	"testing"
)

func TestValidate(t *testing.T) {
	// VALID REGISTRATIONS

	s := &Service{
		ID:          "unique_id",
		Type:        "_service-name._tcp",
		Description: "service description",
		APIs: []API{{
			ID:          "api-id",
			Title:       "API title",
			Description: "API description",
			Protocol:    "HTTPS",
			URL:         "http://localhost:8080",
			Spec: Spec{
				MediaType: "application/vnd.oai.openapi+json;version=3.0",
				URL:       "http://localhost:8080/swaggerSpec.json",
				Schema:    map[string]interface{}{},
			},
			Meta: map[string]interface{}{},
		}},
		Doc:  "https://docs.linksmart.eu/display/SC",
		Meta: map[string]interface{}{},
		TTL:  10,
	}

	err := s.validate()
	if err != nil {
		t.Fatalf("Failed to validate a valid registration: %s", err)
	}

	s.ID = ""
	err = s.validate()
	if err != nil {
		t.Fatalf("Failed to validate a registration without ID: %s", err)
	}

	// INVALID REGISTRATIONS
	var bad Service
	bad = *s
	bad.ID = "id with space"
	err = bad.validate()
	if err == nil {
		t.Fatalf("Failed to invalidate a registration with ID including whitespace")
	}

	bad = *s
	bad.Type = ""
	err = bad.validate()
	if err == nil {
		t.Fatalf("Failed to invalidate a registration with no name")
	}

	bad = *s
	bad.APIs = []API{{
		ID:          "api-id",
		Title:       "API title",
		Description: "API description",
		Protocol:    "HTTPS",
		URL:         ":://localhost",
		Spec: Spec{
			MediaType: "application/vnd.oai.openapi+json;version=3.0",
			URL:       "http://localhost:8080/swaggerSpec.json",
			Schema:    map[string]interface{}{},
		},
		Meta: map[string]interface{}{},
	}}
	err = bad.validate()
	if err == nil {
		t.Fatalf("Failed to invalidate a registration with invalid API url")
	}

	bad = *s
	bad.APIs = []API{{
		ID:          "api-id",
		Title:       "API title",
		Description: "API description",
		Protocol:    "HTTPS",
		URL:         "http://localhost:8080",
		Spec: Spec{
			MediaType: "//",
			URL:       "http://localhost:8080/swaggerSpec.json",
			Schema:    map[string]interface{}{},
		},
		Meta: map[string]interface{}{},
	}}
	err = bad.validate()
	if err == nil {
		t.Fatalf("Failed to invalidate a registration with invalid API Spec mediaType")
	}

	// No such validation needed for 'doc' after the schema change
	/*bad = *s
	bad.Doc = ":://doc.linksmart.eu/DC"
	err = bad.validate()
	if err == nil {
		t.Fatalf("Failed to invalidate a registration with invalid doc url")
	}*/

	bad = *s
	bad.TTL = 0
	err = bad.validate()
	if err == nil {
		t.Fatalf("Failed to invalidate a registration with invalid TTL")
	}
}
