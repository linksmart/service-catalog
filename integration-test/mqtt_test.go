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
	//time.Sleep(5 * time.Second)
	//Publish a service
	service := MockedService("1")
	b, _ := json.Marshal(service)
	manager.client.Publish("LS/MOCK/1/SER/1.0/REG", 1, false, b)

	//create new remote client
	//	httpClient,_ := data.NewRemoteClient("0.0.0.0:8082",nil)
	//	services := httpClient.List()

}
