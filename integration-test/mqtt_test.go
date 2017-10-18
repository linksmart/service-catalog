package integration_test

import (
	"fmt"

	paho "github.com/eclipse/paho.mqtt.golang"

	"code.linksmart.eu/sc/service-catalog/catalog"
	"encoding/json"

	"testing"
	"time"
)

type ClientManager struct {
	url    string
	client paho.Client
	t      *testing.T
	c      chan bool
}

func sameServices(s1, s2 *catalog.Service, checkID bool) bool {
	// Compare IDs if specified
	if checkID {
		if s1.ID != s2.ID {
			return false
		}
	}

	// Compare metadata
	for k1, v1 := range s1.Meta {
		v2, ok := s2.Meta[k1]
		if !ok || v1 != v2 {
			return false
		}
	}
	for k2, v2 := range s2.Meta {
		v1, ok := s1.Meta[k2]
		if !ok || v1 != v2 {
			return false
		}
	}

	// Compare number of protocols
	if len(s1.ExternalDocs) != len(s2.ExternalDocs) {
		return false
	}

	// Compare all other attributes
	if s1.Description != s2.Description || s1.TTL != s2.TTL {
		return false
	}

	return true
}

func (m *ClientManager) onConnectHandler(client paho.Client) {
	fmt.Printf("MQTT: %s: Connected.", m.url)
	m.client = client

	close(m.c)

}

func (m *ClientManager) onConnectionLostHandler(client paho.Client, err error) {
	fmt.Printf("MQTT: %s: Connection lost: %v", m.url, err)
}

func MockedService(id string) *catalog.Service {
	return &catalog.Service{
		ID:          "TestHost/TestService" + id,
		Meta:        map[string]interface{}{"test-id": id},
		Description: "Test Service " + id,
		ExternalDocs: []catalog.ExternalDoc{{
			Description: "REST",
			URL:         "http://link-to-openapi-specs.json",
		}},
		TTL: 100,
	}
}

func TestMqtt(t *testing.T) {

	URL1 := "tcp://localhost:1883"

	manager := &ClientManager{
		url: URL1,
		t:   t,
		c:   make(chan bool),
	}
	opts := paho.NewClientOptions() // uses defaults: https://godoc.org/github.com/eclipse/paho.mqtt.golang#NewClientOptions
	opts.AddBroker(manager.url)
	opts.SetClientID(fmt.Sprintf("sc-tester-%v", 1))
	opts.SetConnectTimeout(5 * time.Second)
	opts.SetOnConnectHandler(manager.onConnectHandler)
	opts.SetConnectionLostHandler(manager.onConnectionLostHandler)
	manager.client = paho.NewClient(opts)
	counter := 1
	for ; counter < 100; counter++ {

		if token := manager.client.Connect(); token.Wait() && token.Error() != nil {
			time.Sleep(1 * time.Second)
		} else {
			t.Log("connected to Broker")
			break
		}
	}
	if counter == 100 {
		t.Fatalf("Timed out waiting for broker")
	}
	<-manager.c

	//Publish a service
	service := MockedService("1")
	b, _ := json.Marshal(service)
	manager.client.Publish("LS/MOCK/1/SER/1.0/REG", 1, false, b)

	time.Sleep(2 * time.Second)
	//verify if the service is created
	httpRemoteClient, _ := catalog.NewRemoteCatalogClient("http://localhost:8082", nil)
	gotService, err := httpRemoteClient.Get(service.ID)
	if err != nil {
		t.Fatalf("Error retrieveing the service %s", service.ID)
		return
	}
	if !sameServices(gotService, service, true) {
		t.Fatalf("The retrieved service is not the same as the added one:\n Added:\n %v \n Retrieved: \n %v", service, gotService)
	}

	//destroy the service
	time.Sleep(2 * time.Second)

	manager.client.Publish("LS/MOCK/1/SER/1.0/WILL", 1, false, b)

	//verify if the service is deleted
	time.Sleep(2 * time.Second)
	gotService, err = httpRemoteClient.Get(service.ID)
	if err != nil {
		switch err.(type) {
		case *catalog.NotFoundError:
			break
		default:
			t.Fatalf("Error while fetching services:%v", err)
		}
	} else {
		t.Fatalf("Service was fetched even after deletion")
	}

}
