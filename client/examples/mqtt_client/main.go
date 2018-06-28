package main

import (
	"encoding/json"
	"log"
	"time"

	"code.linksmart.eu/sc/service-catalog/catalog"
	paho "github.com/eclipse/paho.mqtt.golang"
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
		Name:        "_service-name._tcp",
		Description: "service description",
		APIs:        map[string]string{"Data": "mqtt://test.mosquitto.org:1883"},
		Docs: []catalog.Doc{{
			Description: "doc description",
			URL:         "http://doc.linksmart.eu/DC",
			Type:        "text/html",
			APIs:        []string{"Data"},
		}},
		Meta: map[string]interface{}{"pub_key": "qwertyuiopasdfghjklzxcvbnm"},
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
