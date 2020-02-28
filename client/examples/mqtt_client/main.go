package main

import (
	"encoding/json"
	"log"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/linksmart/service-catalog/v3/catalog"
)

func main() {
	// Create Paho MQTT client and connect
	opts := paho.NewClientOptions()
	opts.AddBroker("tcp://linksmart-1:1883")
	client := paho.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}

	// Create and serialize registration object
	service := catalog.Service{
		ID:          "unique_id2",
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
	b, _ := json.Marshal(service)

	// Publish periodically
	ticker := time.NewTicker(time.Duration(service.TTL) * time.Second)
	go func() {
		for ; true; <-ticker.C {
			log.Print("Updating...")
			client.Publish("LS/v2/example/uuid/service", 1, false, b)
		}
	}()

	time.Sleep(22 * time.Second)
	ticker.Stop()
	// a kind of will message leads to immediate service removal, otherwise the service is removed after expiry; see docs
}
