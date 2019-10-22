// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package catalog

import (
	"testing"
)

func TestValidate(t *testing.T) {
	// VALID REGISTRATIONS

	s := &Service{
		ID:          "unique_id",
		Description: "service description",
		Type:        "_test._tcp",
		APIs:        map[string]string{"API 1": "http://localhost:8080"},
		Docs: []Doc{{
			Description: "doc description",
			URL:         "http://doc.linksmart.eu/DC",
			Type:        "text/html",
			APIs:        []string{"API 1"},
		}},
		Meta: map[string]interface{}{"pub_key": "qwertyuiopasdfghjklzxcvbnm"},
		TTL:  30,
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
	bad.APIs = map[string]string{"API 1": ":://localhost"}
	err = bad.validate()
	if err == nil {
		t.Fatalf("Failed to invalidate a registration with invalid API url")
	}

	bad = *s
	bad.Docs = []Doc{{URL: ":://doc.linksmart.eu/DC"}}
	err = bad.validate()
	if err == nil {
		t.Fatalf("Failed to invalidate a registration with invalid doc url")
	}

	bad = *s
	bad.Docs = []Doc{{Type: "//"}}
	err = bad.validate()
	if err == nil {
		t.Fatalf("Failed to invalidate a registration with invalid doc MIME type")
	}

	bad = *s
	bad.TTL = 0
	err = bad.validate()
	if err == nil {
		t.Fatalf("Failed to invalidate a registration with invalid TTL")
	}
}
