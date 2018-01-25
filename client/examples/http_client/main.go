package main

import (
	"log"
	"time"

	"code.linksmart.eu/sc/service-catalog/catalog"
	"code.linksmart.eu/sc/service-catalog/client"
)

func main() {
	service := catalog.Service{
		ID:          "unique_id",
		Name:        "_service-name._tcp",
		Description: "service description",
		APIs:        map[string]string{"API 1": "http://localhost:8080"},
		Docs: []catalog.Doc{{
			Description: "doc description",
			URL:         "http://doc.linksmart.eu/DC",
			Type:        "text/html",
			APIs:        []string{"API 1"},
		}},
		Meta: map[string]interface{}{"pub_key": "qwertyuiopasdfghjklzxcvbnm"},
		TTL:  10,
	}

	stopRegistrator, err := client.RegisterServiceAndKeepalive("http://localhost:8082", service, nil)
	if err != nil {
		log.Fatal(err)
	}

	time.Sleep(22 * time.Minute)
	// call this on interrupt/kill signal to remove service immediately, otherwise the service is removed after expiry
	stopRegistrator()
}
