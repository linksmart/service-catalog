// Copyright 2016 Fraunhofer Institute for Applied Information Technology FIT

package data

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	//senml "github.com/krylovsk/gosenml"
	"github.com/pborman/uuid"

	"code.linksmart.eu/hds/historical-datastore/registry"
	"code.linksmart.eu/hds/historical-datastore/common"
)

const (
	mqttRetryInterval = 10 // seconds
)

type MQTTConnector struct {
	sync.Mutex
	//registryClient registry.Client
	//storage        Storage
	managers       map[string]*ClientManager
	// cache of resource->ds
	//cache map[string]*registry.DataSource
	// failed mqtt registrations
	failedRegistrations map[string]*registry.MQTTConf
}

type ClientManager struct {
	url    string
	client paho.Client
	// connector *MQTTConnector
	// total subscriptions for each topic in this manager
	subscriptions map[string]*Subscription
}

type Subscription struct {
	//connector *MQTTConnector
	//url       string
	topic     string
	qos       byte
	receivers int
}

func NewMQTTConnector(registryClient registry.Client, storage Storage) (chan<- common.Notification, error) {
	c := &MQTTConnector{
		//registryClient:      registryClient,
		//storage:             storage,
		managers:            make(map[string]*ClientManager),
		//cache:               make(map[string]*registry.DataSource),
		failedRegistrations: make(map[string]*registry.MQTTConf),
	}

	// Run the notification listener
	ntChan := make(chan common.Notification)
	go NtfListenerMQTT(c, ntChan)

	//perPage := 100
	//for page := 1; ; page++ {
	//	datasources, total, err := c.registryClient.GetDataSources(page, perPage)
	//	if err != nil {
	//		return nil, logger.Errorf("MQTT: Error getting data sources: %v", err)
	//	}
	//
	//	for _, ds := range datasources {
	//		if ds.Connector.MQTT != nil {
	//			err := c.register(ds.Connector.MQTT)
	//			if err != nil {
	//				logger.Printf("MQTT: Error registering subscription: %v. Retrying in %ds", err, mqttRetryInterval)
	//				c.failedRegistrations[ds.ID] = ds.Connector.MQTT
	//			}
	//		}
	//	}
	//
	//	if page*perPage >= total {
	//		break
	//	}
	//}

	go c.retryRegistrations()

	return ntChan, nil
}

func (c *MQTTConnector) retryRegistrations() {
	for {
		time.Sleep(mqttRetryInterval * time.Second)
		c.Lock()
		for id, mqttConf := range c.failedRegistrations {
			err := c.register(mqttConf)
			if err != nil {
				logger.Printf("MQTT: Error registering subscription: %v. Retrying in %ds", err, mqttRetryInterval)
				continue
			}
			delete(c.failedRegistrations, id)
		}
		c.Unlock()
	}
}

func (c *MQTTConnector) register(mqttConf *registry.MQTTConf) error {

	if _, exists := c.managers[mqttConf.URL]; !exists { // NO CLIENT FOR THIS BROKER
		manager := &ClientManager{
			url:           mqttConf.URL,
			subscriptions: make(map[string]*Subscription),
		}

		manager.subscriptions[mqttConf.Topic] = &Subscription{
			connector: c,
			url:       mqttConf.URL,
			topic:     mqttConf.Topic,
			qos:       mqttConf.QoS,
			receivers: 1,
		}

		opts := paho.NewClientOptions() // uses defaults: https://godoc.org/github.com/eclipse/paho.mqtt.golang#NewClientOptions
		opts.AddBroker(mqttConf.URL)
		opts.SetClientID(fmt.Sprintf("HDS-%v", uuid.NewRandom()))
		opts.SetConnectTimeout(5 * time.Second)
		opts.SetOnConnectHandler(manager.onConnectHandler)
		opts.SetConnectionLostHandler(manager.onConnectionLostHandler)
		manager.client = paho.NewClient(opts)

		if token := manager.client.Connect(); token.Wait() && token.Error() != nil {
			return logger.Errorf("MQTT: Error connecting to broker: %v", token.Error())
		}
		c.managers[mqttConf.URL] = manager

	} else { // THERE IS A CLIENT FOR THIS BROKER
		manager := c.managers[mqttConf.URL]

		// TODO: check if another wildcard subscription matches the topic.
		if _, exists := manager.subscriptions[mqttConf.Topic]; !exists { // NO SUBSCRIPTION FOR THIS TOPIC
			subscription := &Subscription{
				connector: c,
				url:       mqttConf.URL,
				topic:     mqttConf.Topic,
				qos:       mqttConf.QoS,
				receivers: 1,
			}
			// Subscribe
			if token := manager.client.Subscribe(subscription.topic, subscription.qos, subscription.onMessage); token.Wait() && token.Error() != nil {
				return logger.Errorf("MQTT: Error subscribing: %v", token.Error())
			}
			manager.subscriptions[mqttConf.Topic] = subscription
			logger.Printf("MQTT: %s: Subscribed to %s", mqttConf.URL, mqttConf.Topic)

		} else { // There is a subscription for this topic
			logger.Debugf("MQTT: %s: Already subscribed to %s", mqttConf.URL, mqttConf.Topic)
			manager.subscriptions[mqttConf.Topic].receivers++
		}
	}

	return nil
}

func (c *MQTTConnector) unregister(mqttConf *registry.MQTTConf) error {
	manager := c.managers[mqttConf.URL]
	manager.subscriptions[mqttConf.Topic].receivers--

	if manager.subscriptions[mqttConf.Topic].receivers == 0 {
		// Unsubscribe
		if token := manager.client.Unsubscribe(mqttConf.Topic); token.Wait() && token.Error() != nil {
			return logger.Errorf("MQTT: Error unsubscribing: %v", token.Error())
		}
		delete(manager.subscriptions, mqttConf.Topic)
		logger.Printf("MQTT: %s: Unsubscribed from %s", mqttConf.URL, mqttConf.Topic)
	}
	if len(manager.subscriptions) == 0 {
		// Disconnect
		manager.client.Disconnect(250)
		delete(c.managers, mqttConf.URL)
		logger.Printf("MQTT: %s: Disconnected!", mqttConf.URL)
	}

	return nil
}

func (m *ClientManager) onConnectHandler(client paho.Client) {
	logger.Printf("MQTT: %s: Connected.", m.url)
	m.client = client
	for _, subscription := range m.subscriptions {
		if token := m.client.Subscribe(subscription.topic, subscription.qos, subscription.onMessage); token.Wait() && token.Error() != nil {
			logger.Printf("MQTT: %s: Error subscribing: %v", m.url, token.Error())
		}
		logger.Printf("MQTT: %s: Subscribed to %s", m.url, subscription.topic)
	}
}

func (m *ClientManager) onConnectionLostHandler(client paho.Client, err error) {
	logger.Printf("MQTT: %s: Connection lost: %v", m.url, err)
}

func (s *Subscription) onMessage(client paho.Client, msg paho.Message) {
	logger.Debugf("MQTT: %s %s", msg.Topic(), msg.Payload())

	data := make(map[string][]DataPoint)
	sources := make(map[string]registry.DataSource)
	var senmlMessage senml.Message

	err := json.Unmarshal(msg.Payload(), &senmlMessage)
	if err != nil {
		logger.Printf("MQTT: Error parsing json: %s : %v", msg.Payload(), err)
		return
	}

	err = senmlMessage.Validate()
	if err != nil {
		logger.Printf("MQTT: Invalid SenML: %s : %v", msg.Payload(), err)
		return
	}

	// Fill the data map with provided data points
	entries := senmlMessage.Expand().Entries
	for _, e := range entries {
		if e.Name == "" {
			logger.Printf("MQTT: Error: Resource name not specified: %s", msg.Payload())
			return
		}
		// Find the data source for this entry
		ds, exists := s.connector.cache[e.Name]
		if !exists {
			ds, err = s.connector.registryClient.FindDataSource("resource", "equals", e.Name)
			if err != nil {
				logger.Printf("MQTT: Error finding data source: %v", e.Name)
				return
			}
			if ds == nil {
				logger.Printf("MQTT: Error: Unable to find resource in registry: %v", e.Name)
				return
			}
			s.connector.cache[e.Name] = ds
		}

		// Check if the message is wanted
		if ds.Connector.MQTT == nil {
			logger.Printf("MQTT: Ignoring unwanted message for data source: %v", e.Name)
			return
		}
		if ds.Connector.MQTT.URL != s.url {
			logger.Printf("MQTT: Ignoring message from unwanted broker %v for data source: %v", s.url, e.Name)
			return
		}
		if ds.Connector.MQTT.Topic != s.topic {
			// logger.Printf("MQTT: Ignoring message with unwanted topic %v for data source: %v", s.topic, e.Name)
			return
		}

		// Check if type of value matches the data source type in registry
		typeError := false
		switch ds.Type {
		case common.FLOAT:
			if e.BooleanValue != nil || e.StringValue != nil && *e.StringValue != "" {
				typeError = true
			}
		case common.STRING:
			if e.Value != nil || e.BooleanValue != nil {
				typeError = true
			}
		case common.BOOL:
			if e.Value != nil || e.StringValue != nil && *e.StringValue != "" {
				typeError = true
			}
		}
		if typeError {
			logger.Printf("MQTT: Error: Entry for data point %v has a type that is incompatible with source registration. Source %v has type %v.", e.Name, ds.ID, ds.Type)
			return
		}

		_, ok := data[ds.ID]
		if !ok {
			data[ds.ID] = []DataPoint{}
			sources[ds.ID] = *ds
		}
		p := NewDataPoint()
		data[ds.ID] = append(data[ds.ID], p.FromEntry(e))
	}

	// Add data to the storage
	err = s.connector.storage.Submit(data, sources)
	if err != nil {
		logger.Printf("MQTT: Error writing data to the database: %v", err)
		return
	}
}