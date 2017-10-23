package integration_test

import (
	"encoding/json"
	"log"
	"os"
	"testing"
	"time"

	"code.linksmart.eu/sc/service-catalog/catalog"
	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/satori/go.uuid"
)

const (
	ServiceCatalogURL = "http://service-catalog:8082"
	Broker1           = "tcp://broker1:1883"
)

type ClientManager struct {
	url       string
	client    paho.Client
	connected chan bool
}

var manager *ClientManager

const sleepTime = 2 * time.Second

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
	log.Printf("MQTT: %s: Connected.\n", m.url)
	m.client = client

	m.connected <- true
}

func (m *ClientManager) onConnectionLostHandler(client paho.Client, err error) {
	log.Printf("MQTT: %s: Connection lost: %v \n", m.url, err)
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

//TODO_: Improve this: with MQTT brokers runnning as docker images and the test script in another container. Use Bamboo to trigger this.
func TestMain(m *testing.M) {

	manager = &ClientManager{
		url:       Broker1,
		connected: make(chan bool),
	}
	opts := paho.NewClientOptions() // uses defaults: https://godoc.org/github.com/eclipse/paho.mqtt.golang#NewClientOptions
	opts.AddBroker(manager.url)
	opts.SetClientID(uuid.NewV4().String())
	opts.SetConnectTimeout(5 * time.Second)
	opts.SetOnConnectHandler(manager.onConnectHandler)
	opts.SetConnectionLostHandler(manager.onConnectionLostHandler)
	manager.client = paho.NewClient(opts)

	for counter := 1; ; counter++ {
		if token := manager.client.Connect(); token.Wait() && token.Error() != nil {
			time.Sleep(1 * time.Second)
		} else {
			log.Println("connected to Broker")
			break
		}
		log.Println("Waiting for broker", manager.url)
		if counter == 100 {
			log.Fatalln("Timed out waiting for broker")
		}
	}
	<-manager.connected

	for counter := 1; ; counter++ {
		httpRemoteClient, _ := catalog.NewRemoteCatalogClient(ServiceCatalogURL, nil)
		_, _, err := httpRemoteClient.List(0, 100)
		if err == nil {
			log.Println("reached service catalog at", ServiceCatalogURL)
			break
		}
		log.Println("Waiting for service catalog", ServiceCatalogURL)
		time.Sleep(1 * time.Second)
		if counter == 100 {
			log.Fatalln("Timed out waiting for broker")
		}
	}

	if m.Run() == 1 {
		os.Exit(1)
	}

	manager.client.Disconnect(100)
	os.Exit(0)
}

func TestCreateAndDelete(t *testing.T) {

	//Publish a service
	service := MockedService("1")
	b, _ := json.Marshal(service)
	manager.client.Publish("LS/MOCK/1/SER/1.0/REG", 1, false, b)

	time.Sleep(sleepTime)
	//verify if the service is created
	httpRemoteClient, _ := catalog.NewRemoteCatalogClient(ServiceCatalogURL, nil)
	gotService, err := httpRemoteClient.Get(service.ID)
	if err != nil {
		t.Fatalf("Error retrieveing the service %s", service.ID)
		return
	}
	if !sameServices(gotService, service, true) {
		t.Fatalf("The retrieved service is not the same as the added one:\n Added:\n %v \n Retrieved: \n %v", service, gotService)
	}

	//destroy the service
	time.Sleep(sleepTime)

	manager.client.Publish("LS/MOCK/1/SER/1.0/WILL", 1, false, b)

	//verify if the service is deleted
	time.Sleep(sleepTime)
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

func TestCreateUpdateAndDelete(t *testing.T) {

	//Publish a service
	service := MockedService("1")
	b, _ := json.Marshal(service)
	manager.client.Publish("LS/MOCK/1/SER/1.0/REG", 1, false, b)

	time.Sleep(sleepTime)
	//verify if the service is created
	httpRemoteClient, _ := catalog.NewRemoteCatalogClient(ServiceCatalogURL, nil)
	gotService, err := httpRemoteClient.Get(service.ID)
	if err != nil {
		t.Fatalf("Error retrieveing the service %s", service.ID)
		return
	}
	if !sameServices(gotService, service, true) {
		t.Fatalf("The retrieved service is not the same as the added one:\n Added:\n %v \n Retrieved: \n %v", service, gotService)
	}

	//update the service
	service.TTL = 200
	b, _ = json.Marshal(service)
	manager.client.Publish("LS/MOCK/1/SER/1.0/REG", 1, false, b)

	time.Sleep(sleepTime)
	//verify if the service is created
	gotService, err = httpRemoteClient.Get(service.ID)
	if err != nil {
		t.Fatalf("Error retrieveing the service %s", service.ID)
		return
	}
	if !sameServices(gotService, service, true) {
		t.Fatalf("The retrieved service is not the same as the added one:\n Added:\n %v \n Retrieved: \n %v", service, gotService)
	}
	//destroy the service
	time.Sleep(sleepTime)

	manager.client.Publish("LS/MOCK/1/SER/1.0/WILL", 1, false, b)

	//verify if the service is deleted
	time.Sleep(sleepTime)
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
