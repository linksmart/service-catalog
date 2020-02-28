package main

import (
	"log"
	"time"

	"github.com/linksmart/service-catalog/v3/catalog"
	"github.com/linksmart/service-catalog/v3/client"
)

func main() {
	service := catalog.Service{
		ID:          "unique_id",
		Type:        "_service-name._tcp",
		Description: "service description",
		APIs: []catalog.API{{
			ID:          "api-id",
			Title:       "API title",
			Description: "API description",
			Protocol:    "HTTPS",
			URL:         "http://localhost:8080",
			Spec: catalog.Spec{
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

	stopRegistrator, _, err := client.RegisterServiceAndKeepalive("http://localhost:8082", service, nil)
	if err != nil {
		log.Fatal(err)
	}

	time.Sleep(22 * time.Minute)
	// call this on interrupt/kill signal to remove service immediately, otherwise the service is removed after expiry
	stopRegistrator()
}
