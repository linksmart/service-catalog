// Copyright 2016 Fraunhofer Institute for Applied Information Technology FIT

package catalog

import (
	"encoding/json"
	"fmt"
	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/pborman/uuid"
	"sync"
	"time"
)

const (
	mqttRetryInterval = 10 // seconds
)

type MqttError struct{ s string }

func (e *MqttError) Error() string { return e.s }

// MQTT describes a MQTT Connector
type MQTTConf struct {
	Brokers   []Broker `json:"brokers"`
	RegTopic  string   `json:"regTopic"`
	WillTopic string   `json:"willTopic"`
}
type Broker struct {
	ID       string   `json:"id"`
	URL      string   `json:"url"`
	Topics   []string `json:"topic"`
	QoS      byte     `json:"qos"`
	Username string   `json:"username,omitempty"`
	Password string   `json:"password,omitempty"`
	CaFile   string   `json:"caFile,omitempty"`
	CertFile string   `json:"certFile,omitempty"`
	KeyFile  string   `json:"keyFile,omitempty"`
}

type MQTTConnector struct {
	sync.Mutex
	controller          *Controller
	managers            map[string]*ClientManager
	RegTopic            string
	WillTopic           string
	failedRegistrations map[string]Broker
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
	IsWill    bool
}

func NewMQTTAPI(controller *Controller, mqttConf MQTTConf) error {
	c := &MQTTConnector{
		//registryClient:      registryClient,
		controller:          controller,
		managers:            make(map[string]*ClientManager),
		failedRegistrations: make(map[string]Broker),
		RegTopic:            mqttConf.RegTopic,
		WillTopic:           mqttConf.WillTopic,
	}

	for _, broker := range mqttConf.Brokers {

		broker.Topics = append(broker.Topics, mqttConf.RegTopic, mqttConf.WillTopic)

		c.failedRegistrations[broker.ID] = broker
	}

	go c.retryRegistrations()

	return nil
}

func (c *MQTTConnector) retryRegistrations() {
	for {
		time.Sleep(mqttRetryInterval * time.Second)
		c.Lock()
		for id, broker := range c.failedRegistrations {
			err := c.register(broker)
			if err != nil {
				fmt.Sprintf("MQTT: Error registering subscription: %v. Retrying in %ds", err, mqttRetryInterval)
				continue
			}
			delete(c.failedRegistrations, id)
		}
		c.Unlock()
	}
}

func (c *MQTTConnector) register(broker Broker) error {

	if _, exists := c.managers[broker.URL]; !exists { // NO CLIENT FOR THIS BROKER
		manager := &ClientManager{
			url:           broker.URL,
			subscriptions: make(map[string]*Subscription),
		}

		for _, topic := range broker.Topics {
			manager.subscriptions[topic] = &Subscription{
				topic:     topic,
				qos:       broker.QoS,
				receivers: 1,
				IsWill:    (topic == c.WillTopic),
			}
		}
		opts := paho.NewClientOptions() // uses defaults: https://godoc.org/github.com/eclipse/paho.mqtt.golang#NewClientOptions
		opts.AddBroker(broker.URL)
		opts.SetClientID(fmt.Sprintf("SC-%v", uuid.NewRandom()))
		opts.SetConnectTimeout(5 * time.Second)
		opts.SetOnConnectHandler(manager.onConnectHandler)
		opts.SetConnectionLostHandler(manager.onConnectionLostHandler)
		manager.client = paho.NewClient(opts)

		if token := manager.client.Connect(); token.Wait() && token.Error() != nil {
			return &MqttError(fmt.Sprintf("MQTT: Error connecting to broker: %v", token.Error()))
		}
		c.managers[broker.URL] = manager

	} else { // THERE IS A CLIENT FOR THIS BROKER
		manager := c.managers[broker.URL]

		for _, topic := range broker.Topics {
			// TODO: check if another wildcard subscription matches the topic.
			if _, exists := manager.subscriptions[topic]; !exists { // NO SUBSCRIPTION FOR THIS TOPIC

				subscription := &Subscription{
					topic:     topic,
					qos:       broker.QoS,
					receivers: 1,
					IsWill:    (topic == c.WillTopic),
				}

				// Subscribe
				if token := manager.client.Subscribe(subscription.topic, subscription.qos, subscription.onMessage); token.Wait() && token.Error() != nil {
					return &MqttError(fmt.Sprintf("MQTT: Error subscribing: %v", token.Error()))
				}
				manager.subscriptions[topic] = subscription
				return &MqttError(fmt.Sprintf("MQTT: %s: Subscribed to %s", broker.URL, topic))

			} else { // There is a subscription for this topic
				return &MqttError(fmt.Sprintf("MQTT: %s: Already subscribed to %s", broker.URL, topic))
				manager.subscriptions[topic].receivers++
			}
		}
	}

	return nil
}

func (c *MQTTConnector) unregister(broker Broker) error {
	manager := c.managers[broker.URL]
	for _, topic := range broker.Topics {
		manager.subscriptions[topic].receivers--

		if manager.subscriptions[topic].receivers == 0 {
			// Unsubscribe
			if token := manager.client.Unsubscribe(topic); token.Wait() && token.Error() != nil {
				return logger.Errorf("MQTT: Error unsubscribing: %v", token.Error())
			}
			delete(manager.subscriptions, topic)
			logger.Printf("MQTT: %s: Unsubscribed from %s", broker.URL, topic)
		}
	}
	if len(manager.subscriptions) == 0 {
		// Disconnect
		manager.client.Disconnect(250)
		delete(c.managers, broker.URL)
		logger.Printf("MQTT: %s: Disconnected!", broker.URL)
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

	s.createOrUpdate(service)

}

func (s *Subscription) createOrUpdate(service Service) {
	_, err := s.connector.controller.update(service.ID, service)

	if service.ID == "" { //There is no way SC can communicate back a new service's ID back to the service.
		logger.Printf("MQTT: Cannot register a service without ID")
		return
	}

	if s.IsWill {
		s.connector.controller.delete(service.ID)
		return
	}

	if err != nil {
		switch err.(type) {
		case *NotFoundError:
			// Create a new service with the given id
			s.createService(service)
			return
		case *ConflictError:
			logger.Printf("MQTT: Error updating the service:%s", err.Error())
			return
		case *BadRequestError:
			logger.Printf("MQTT: Invalid service registration:%s", err.Error())
			return
		default:
			logger.Printf("MQTT: Error updating the service:%s", err.Error())
			return
		}
	}
}

func (s *Subscription) createService(service Service) {
	_, err := s.connector.controller.add(service)
	if err != nil {
		switch err.(type) {
		case *ConflictError:
			logger.Printf("MQTT: Error adding the service:%s", err.Error())
			return
		case *BadRequestError:
			logger.Printf("MQTT: Invalid service registration:%s", err.Error())
			return
		default:
			logger.Printf("MQTT: Error updating the service:%s", err.Error())
			return
		}
	}
}
