// Copyright 2016 Fraunhofer Institute for Applied Information Technology FIT

package catalog

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/pborman/uuid"
)

const (
	mqttRetryInterval = 10 // seconds
)

type MqttError struct{ s string }

func (e *MqttError) Error() string { return e.s }

// MQTT describes a MQTT Connector
type MQTTConf struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Topic    string `json:"topic"`
	QoS      byte   `json:"qos"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	CaFile   string `json:"caFile,omitempty"`
	CertFile string `json:"certFile,omitempty"`
	KeyFile  string `json:"keyFile,omitempty"`
}

type MQTTConnector struct {
	sync.Mutex
	//registryClient registry.Client
	controller *Controller
	managers       map[string]*ClientManager
	// cache of resource->ds
	//cache map[string]*registry.DataSource
	// failed mqtt registrations
	failedRegistrations map[string] MQTTConf
}

type ClientManager struct {
	url    string
	client paho.Client
	// connector *MQTTConnector
	// total subscriptions for each topic in this manager
	subscriptions map[string]*Subscription
}

type Subscription struct {
	connector *MQTTConnector
	//url       string
	topic     string
	qos       byte
	receivers int
}

func NewMQTTAPI(controller *Controller,mqttConfs [] MQTTConf) error {
	c := &MQTTConnector{
		//registryClient:      registryClient,
		controller:             controller,
		managers:            make(map[string]*ClientManager),
		failedRegistrations: make(map[string] MQTTConf),
	}

	for _,mqttConf := range mqttConfs{
		c.failedRegistrations[mqttConf.ID] = mqttConf
	}

	go c.retryRegistrations()

	return nil
}

func (c *MQTTConnector) retryRegistrations() {
	for {
		time.Sleep(mqttRetryInterval * time.Second)
		c.Lock()
		for id, mqttConf := range c.failedRegistrations {
			err := c.register(mqttConf)
			if err != nil {
				fmt.Sprintf("MQTT: Error registering subscription: %v. Retrying in %ds", err, mqttRetryInterval)
				continue
			}
			delete(c.failedRegistrations, id)
		}
		c.Unlock()
	}
}

func (c *MQTTConnector) register(mqttConf MQTTConf) error {

	if _, exists := c.managers[mqttConf.URL]; !exists { // NO CLIENT FOR THIS BROKER
		manager := &ClientManager{
			url:           mqttConf.URL,
			subscriptions: make(map[string]*Subscription),
		}

		manager.subscriptions[mqttConf.Topic] = &Subscription{
			topic:     mqttConf.Topic,
			qos:       mqttConf.QoS,
			receivers: 1,
		}

		opts := paho.NewClientOptions() // uses defaults: https://godoc.org/github.com/eclipse/paho.mqtt.golang#NewClientOptions
		opts.AddBroker(mqttConf.URL)
		opts.SetClientID(fmt.Sprintf("SC-%v", uuid.NewRandom()))
		opts.SetConnectTimeout(5 * time.Second)
		opts.SetOnConnectHandler(manager.onConnectHandler)
		opts.SetConnectionLostHandler(manager.onConnectionLostHandler)
		manager.client = paho.NewClient(opts)

		if token := manager.client.Connect(); token.Wait() && token.Error() != nil {
			return &MqttError(fmt.Sprintf("MQTT: Error connecting to broker: %v", token.Error()))
		}
		c.managers[mqttConf.URL] = manager

	} else { // THERE IS A CLIENT FOR THIS BROKER
		manager := c.managers[mqttConf.URL]

		// TODO: check if another wildcard subscription matches the topic.
		if _, exists := manager.subscriptions[mqttConf.Topic]; !exists { // NO SUBSCRIPTION FOR THIS TOPIC
			subscription := &Subscription{
				topic:     mqttConf.Topic,
				qos:       mqttConf.QoS,
				receivers: 1,
			}
			// Subscribe
			if token := manager.client.Subscribe(subscription.topic, subscription.qos, subscription.onMessage); token.Wait() && token.Error() != nil {
				return &MqttError(fmt.Sprintf("MQTT: Error subscribing: %v", token.Error()))
			}
			manager.subscriptions[mqttConf.Topic] = subscription
			return &MqttError(fmt.Sprintf("MQTT: %s: Subscribed to %s", mqttConf.URL, mqttConf.Topic))

		} else { // There is a subscription for this topic
			return &MqttError(fmt.Sprintf("MQTT: %s: Already subscribed to %s", mqttConf.URL, mqttConf.Topic))
			manager.subscriptions[mqttConf.Topic].receivers++
		}
	}

	return nil
}

func (c *MQTTConnector) unregister(mqttConf MQTTConf) error {
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

	var service Service

	err := json.Unmarshal(msg.Payload(), &service)
	if err != nil {
		logger.Printf("MQTT: Error parsing json: %s : %v", msg.Payload(), err)
		return
	}

	s.connector.controller.add(service)
}