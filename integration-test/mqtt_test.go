package integration_test

import (
	"encoding/json"
	"log"
	"os"
	"testing"
	"time"

	"code.linksmart.eu/sc/service-catalog/client"

	"strings"

	"code.linksmart.eu/sc/service-catalog/catalog"
	paho "github.com/eclipse/paho.mqtt.golang"
	uuid "github.com/satori/go.uuid"
)

var (
	// Can be overriden with env variables
	ServiceCatalogURL = "http://localhost:8082"
	Brokers           = []string{"tcp://localhost:1883"}
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
	if len(s1.Docs) != len(s2.Docs) {
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
		ID:          "TestHost/" + id,
		Meta:        map[string]interface{}{"test-id": id},
		Description: "Test Service " + id,
		Name:        "_test._tcp",
		APIs:        map[string]string{"Dummy": "mqtt://test.mosquitto.org:1883"},
		Docs: []catalog.Doc{{
			Description: "REST API Specs",
			Type:        "openapi",
			URL:         "http://link-to-openapi-specs.json",
			APIs:        []string{"Dummy"},
		}},
		TTL: 100,
	}
}

//TODO_: Improve this: with MQTT brokers runnning as docker images and the test script in another container. Use Bamboo to trigger this.
func TestMain(m *testing.M) {

	// Take urls from envs (if provided)
	if url := os.Getenv("SC"); url != "" {
		log.Println("Setting service catalog:", url)
		ServiceCatalogURL = url
	}
	if joinedUrls := os.Getenv("BROKERS"); joinedUrls != "" {
		urls := strings.Split(joinedUrls, ",")
		log.Println("Setting brokers:", urls)
		Brokers = make([]string, len(urls))
		for i, url := range urls {
			Brokers[i] = url
		}
	}

	manager = &ClientManager{
		url:       Brokers[0],
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
			log.Println(token.Error())
		} else {
			log.Println("connected to broker", manager.url)
			break
		}
		if counter == 30 {
			log.Fatalln("Timed out waiting for broker")
		}
		log.Println("Waiting for broker", manager.url)
		time.Sleep(1 * time.Second)
	}
	<-manager.connected

	for counter := 1; ; counter++ {
		httpRemoteClient, _ := client.NewHTTPClient(ServiceCatalogURL, nil)
		_, _, err := httpRemoteClient.GetMany(1, 100, nil)
		if err != nil {
			log.Println(err)
		} else {
			log.Println("reached service catalog at", ServiceCatalogURL)
			break
		}
		if counter == 30 {
			log.Fatalln("Timed out waiting for service catalog")
		}
		log.Println("Waiting for service catalog", ServiceCatalogURL)
		time.Sleep(1 * time.Second)
	}

	if m.Run() == 1 {
		os.Exit(1)
	}

	manager.client.Disconnect(100)
	os.Exit(0)
}

func TestCreateDelete(t *testing.T) {

	//Publish a service
	service := MockedService(uuid.NewV4().String())
	b, _ := json.Marshal(service)
	manager.client.Publish("LS/v2/IT/someid/service", 1, false, b)

	time.Sleep(sleepTime)
	//verify if the service is created
	httpRemoteClient, _ := client.NewHTTPClient(ServiceCatalogURL, nil)
	gotService, err := httpRemoteClient.Get(service.ID)
	if err != nil {
		t.Fatalf("Error retrieveing the service %s: %s", service.ID, err)
		return
	}
	if !sameServices(gotService, service, true) {
		t.Fatalf("The retrieved service is not the same as the added one:\n Added:\n %v \n Retrieved: \n %v", service, gotService)
	}

	//destroy the service
	time.Sleep(sleepTime)

	if token := manager.client.Publish("LS/v2/IT/someid/will", 1, false, b); token.Wait() && token.Error() != nil {
		t.Fatalf("Error publishing: %s", token.Error())
	}

	//verify if the service is deleted
	time.Sleep(sleepTime)
	gotService, err = httpRemoteClient.Get(service.ID)
	if err != nil {
		switch err.(type) {
		case *catalog.NotFoundError:
			break
		default:
			t.Fatalf("Error while fetching services: %s", err)
		}
	} else {
		t.Fatalf("Service was fetched even after deletion")
	}

}

func TestCreateUpdate(t *testing.T) {

	//Publish a service
	service := MockedService(uuid.NewV4().String())
	b, _ := json.Marshal(service)
	manager.client.Publish("LS/v2/IT/someid/service", 1, false, b)

	time.Sleep(sleepTime)
	//verify if the service is created
	httpRemoteClient, _ := client.NewHTTPClient(ServiceCatalogURL, nil)
	gotService, err := httpRemoteClient.Get(service.ID)
	if err != nil {
		t.Fatalf("Error retrieveing the service %s", service.ID)
	}
	if !sameServices(gotService, service, true) {
		t.Fatalf("The retrieved service is not the same as the added one:\n Added:\n %v \n Retrieved: \n %v", service, gotService)
	}

	//update the service
	service.TTL = 200
	b, _ = json.Marshal(service)
	if token := manager.client.Publish("LS/v2/IT/someid/service", 1, false, b); token.Wait() && token.Error() != nil {
		t.Fatalf("Error publishing: %s", token.Error())
	}

	time.Sleep(sleepTime)
	//verify if the service is updated
	gotService, err = httpRemoteClient.Get(service.ID)
	if err != nil {
		t.Fatalf("Error retrieveing the service %s: %s", service.ID, err)
	}
	if !sameServices(gotService, service, true) {
		t.Fatalf("The retrieved service is not the same as the added one:\n Added:\n %v \n Retrieved: \n %v", service, gotService)
	}

}

func TestCreateDeleteWithIdInTopic(t *testing.T) {
	id := "1234"

	//Publish a service
	service := MockedService("")
	service.ID = "" // clear the id field
	b, _ := json.Marshal(service)
	manager.client.Publish("LS/v2/IT/someid/service/"+id, 1, false, b)

	time.Sleep(sleepTime)
	//verify if the service is created
	httpRemoteClient, _ := client.NewHTTPClient(ServiceCatalogURL, nil)
	_, err := httpRemoteClient.Get(id)
	if err != nil {
		t.Fatalf("Error retrieveing the service %s: %s", id, err)
	}

	//destroy the service
	time.Sleep(sleepTime)

	if token := manager.client.Publish("LS/v2/IT/someid/will/"+id, 1, false, b); token.Wait() && token.Error() != nil {
		t.Fatalf("Error publishing: %s", token.Error())
	}

	//verify if the service is deleted
	time.Sleep(sleepTime)
	_, err = httpRemoteClient.Get(id)
	if err != nil {
		switch err.(type) {
		case *catalog.NotFoundError:
			break
		default:
			t.Fatalf("Error while fetching services: %s", err)
		}
	} else {
		t.Fatalf("Service was fetched even after deletion")
	}

}

func TestAnnouncement(t *testing.T) {
	id := "1234"
	//Publish a service
	service := MockedService("")
	service.ID = "" // clear the id field
	updateService := MockedService("")
	updateService.ID = id
	updateService.TTL = 200

	create := make(chan bool)
	update := make(chan bool)
	gotDead := make(chan bool)
	deleteRetain := make(chan bool)

	aliveTopic := "LS/v2/SC/" + service.Name + "/" + id + "/+"
	log.Println("subscribing:", aliveTopic)
	if token1 := manager.client.Subscribe(aliveTopic, 1, func(client paho.Client, msg paho.Message) {
		defer log.Println("Got a message exit", msg.Topic())
		log.Println("Got a message", msg.Topic())
		var gotService catalog.Service
		if strings.HasSuffix(msg.Topic(), "alive") {
			if len(msg.Payload()) == 0 {
				log.Println("Got a empty message")
				deleteRetain <- true
				return
			}
			err := json.Unmarshal(msg.Payload(), &gotService)
			if err != nil {
				t.Fatalf("Error while fetching services: %s", err)
				return
			}
			if sameServices(service, &gotService, false) {
				log.Println("Got a Created Service")
				create <- true
			} else if sameServices(updateService, &gotService, false) {
				log.Println("Got an Updated service")
				update <- true
			} else {
				//t.Fatalf("The message was something not expected")
			}
		} else if strings.HasSuffix(msg.Topic(), "dead") {
			log.Println("Got a gotDead message")
			err := json.Unmarshal(msg.Payload(), &gotService)
			if err != nil {
				t.Fatalf("Error while fetching services: %s", err)
				return
			}
			if sameServices(updateService, &gotService, false) {
				log.Println("Announcement: Service deleted %s", gotService.ID)
				gotDead <- true
			}
		} else {
			//t.Fatalf("The message topic was something not expected")
		}

	}); token1.Wait() && token1.Error() != nil {
		t.Fatal(token1.Error())
	}

	b, _ := json.Marshal(service)
	if token3 := manager.client.Publish("LS/v2/IT/someid/service/"+id, 1, false, b); token3.Wait() && token3.Error() != nil {
		t.Fatalf("Error publishing: %s", token3.Error())
	}

	//wait for creation
	select {
	case <-create:
		log.Println("Creation Announcement was successful")
	case <-time.After(10 * time.Second):
		t.Fatalf("timeout waiting for creation announcement")
	}

	b, _ = json.Marshal(updateService)
	if token4 := manager.client.Publish("LS/v2/IT/someid/service/"+id, 1, false, b); token4.Wait() && token4.Error() != nil {
		t.Fatalf("Error publishing: %s", token4.Error())
	}

	//wait for update announcement
	select {
	case <-update:
		log.Println("Update announcement was successful")
	case <-time.After(10 * time.Second):
		t.Fatalf("timeout waiting for update announcement")
	}

	if token5 := manager.client.Publish("LS/v2/IT/someid/will/"+id, 1, false, b); token5.Wait() && token5.Error() != nil {
		t.Fatalf("Error publishing: %s", token5.Error())
	}

	for i := 0; i < 2; i++ {
		//wait for gotDead announcement
		select {
		case <-gotDead:
			log.Println("Delete announcement was successful")
		case <-deleteRetain:
			log.Println("DeleteRetain announcement was successful")
		case <-time.After(10 * time.Second):
			t.Fatalf("timeout waiting for gotDead/deleteRetain announcements")
		}
	}

}
