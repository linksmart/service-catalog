// Copyright 2016 Fraunhofer Institute for Applied Information Technology FIT

package catalog

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sync"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	uuid "github.com/satori/go.uuid"
)

const (
	mqttRetryInterval = 10 // seconds
)

type MQTTConf struct {
	Brokers         []Broker `json:"brokers"`
	CommonRegTopic  []string `json:"commonRegTopics"`
	CommonWillTopic []string `json:"commonWillTopics"`
}

func (c MQTTConf) Validate() error {
	for _, broker := range c.Brokers {
		_, err := url.Parse(broker.URL)
		if err != nil {
			return err
		}
		if broker.QoS > 2 {
			return fmt.Errorf("MQTT: qos must be 0, 1, or 2")
		}
		if len(c.CommonRegTopic) == 0 && len(broker.RegTopics) == 0 {
			return fmt.Errorf("MQTT: regTopics not defined")
		}
	}
	return nil
}

type Broker struct {
	ID         string   `json:"id"`
	URL        string   `json:"url"`
	RegTopics  []string `json:"regTopics"`
	WillTopics []string `json:"willTopics"`
	QoS        byte     `json:"qos"`
	Username   string   `json:"username,omitempty"`
	Password   string   `json:"password,omitempty"`
	//CaFile     string   `json:"caFile,omitempty"`
	//CertFile   string   `json:"certFile,omitempty"`
	//KeyFile    string   `json:"keyFile,omitempty"`
	topics []string
	will   map[string]bool
}

type MQTTConnector struct {
	sync.Mutex
	controller          *Controller
	scID                string
	managers            map[string]*ClientManager
	failedRegistrations map[string]Broker
}

type ClientManager struct {
	url       string
	id        string
	client    paho.Client
	connector *MQTTConnector
	// total subscriptions for each topic in this manager
	subscriptions map[string]*Subscription
}

type Subscription struct {
	connector *MQTTConnector
	topic     string
	qos       byte
	receivers int
	will      bool
}

func StartMQTTConnector(controller *Controller, mqttConf MQTTConf, scDescription string) {
	c := &MQTTConnector{
		controller:          controller,
		scID:                scDescription,
		managers:            make(map[string]*ClientManager),
		failedRegistrations: make(map[string]Broker),
	}

	for _, broker := range mqttConf.Brokers {
		broker.will = make(map[string]bool)
		for _, topic := range append(mqttConf.CommonWillTopic, broker.WillTopics...) {
			broker.will[topic] = true
		}
		for _, topics := range [][]string{mqttConf.CommonRegTopic, mqttConf.CommonWillTopic, broker.RegTopics, broker.WillTopics} {
			broker.topics = append(broker.topics, topics...)
		}

		err := c.register(broker)
		if err != nil {
			logger.Printf("MQTT: Error registering subscription: %v. Retrying in %ds", err, mqttRetryInterval)
			c.failedRegistrations[broker.ID] = broker
		}
	}

	c.retryRegistrations()
}

func (c *MQTTConnector) retryRegistrations() {
	for {
		time.Sleep(mqttRetryInterval * time.Second)
		c.Lock()
		for id, broker := range c.failedRegistrations {
			err := c.register(broker)
			if err != nil {
				logger.Printf("MQTT: Error registering subscription: %v. Retrying in %ds", err, mqttRetryInterval)
				continue
			}
			delete(c.failedRegistrations, id)
		}
		c.Unlock()
	}
}

func (c *MQTTConnector) register(broker Broker) error {
	// TODO
	// the else section can be removed because no one will use two broker configuration blocks for the same broker
	if _, exists := c.managers[broker.URL]; !exists { // NO CLIENT FOR THIS BROKER
		manager := &ClientManager{
			url:           broker.URL,
			subscriptions: make(map[string]*Subscription),
			connector:     c,
			id:            broker.ID,
		}

		for _, topic := range broker.topics {
			manager.subscriptions[topic] = &Subscription{
				topic:     topic,
				qos:       broker.QoS,
				receivers: 1,
				will:      broker.will[topic],
				connector: c,
			}
		}

		opts := paho.NewClientOptions() // uses defaults: https://godoc.org/github.com/eclipse/paho.mqtt.golang#NewClientOptions
		opts.AddBroker(broker.URL)
		opts.SetClientID(fmt.Sprintf("SC-%v", uuid.NewV4().String()))
		opts.SetConnectTimeout(5 * time.Second)
		opts.SetOnConnectHandler(manager.onConnectHandler)
		opts.SetConnectionLostHandler(manager.onConnectionLostHandler)
		if broker.Username != "" {
			opts.SetUsername(broker.Username)
			opts.SetPassword(broker.Password)
		}
		// TODO: add support for certificate auth
		//
		manager.client = paho.NewClient(opts)

		if token := manager.client.Connect(); token.Wait() && token.Error() != nil {
			return fmt.Errorf("MQTT: Error connecting to broker: %v", token.Error())
		}
		c.managers[broker.URL] = manager

	} else { // THERE IS A CLIENT FOR THIS BROKER
		manager := c.managers[broker.URL]

		for _, topic := range broker.topics {
			// TODO: check if another wildcard subscription matches the topic.
			if _, exists := manager.subscriptions[topic]; !exists { // NO SUBSCRIPTION FOR THIS TOPIC

				subscription := &Subscription{
					topic:     topic,
					qos:       broker.QoS,
					receivers: 1,
					will:      broker.will[topic],
					connector: c,
				}

				// Subscribe
				if token := manager.client.Subscribe(subscription.topic, subscription.qos, subscription.onMessage); token.Wait() && token.Error() != nil {
					return fmt.Errorf("MQTT: Error subscribing: %v", token.Error())
				}
				manager.subscriptions[topic] = subscription
				logger.Printf("MQTT: %s: Subscribed to %s %s", broker.URL, topic, subscription.printIfWill())

			} else { // THERE IS A SUBSCRIPTION FOR THIS TOPIC
				logger.Printf("MQTT: %s: Already subscribed to %s", broker.URL, topic)
				manager.subscriptions[topic].receivers++
			}
		}
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
		logger.Printf("MQTT: %s: Subscribed to %s %s", m.url, subscription.topic, subscription.printIfWill())
	}

	//Add this broker to list of MQTT brokers
	m.addBrokerAsService()
}

func (m *ClientManager) addBrokerAsService() {
	service := Service{
		ID:          m.id,
		Name:        "_mqtt._tcp",
		Description: "MQTT Broker",
		Meta: map[string]interface{}{
			"registrator": m.connector.scID,
		},
		APIs: map[string]string{
			"MQTT": m.url,
		},
	}
	_, err := m.connector.controller.add(service)
	if err != nil {
		logger.Printf("MQTT: Error registering broker %s: %s", m.id, err)
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

	if service.ID == "" {
		logger.Printf("MQTT: Invalid service: No ID provided")
		return
	}

	s.connector.handleService(service, s.will)
}

func (s *Subscription) printIfWill() string {
	if s.will {
		return "(will topic)"
	}
	return ""
}

func (connector *MQTTConnector) handleService(service Service, will bool) {
	if will {
		connector.controller.delete(service.ID)
		logger.Printf("MQTT: Removed service %s", service.ID)
		return
	}

	_, err := connector.controller.update(service.ID, service)
	if err != nil {
		switch err.(type) {
		case *NotFoundError:
			// Create a new service with the given id
			_, err := connector.controller.add(service)
			if err != nil {
				switch err.(type) {
				case *ConflictError:
					logger.Printf("MQTT: Error adding the service:%s", err.Error())
				case *BadRequestError:
					logger.Printf("MQTT: Invalid service registration:%s", err.Error())
				default:
					logger.Printf("MQTT: Error updating the service:%s", err.Error())
				}
			}
			logger.Printf("MQTT: Created service %s", service.ID)
			return
		case *ConflictError:
			logger.Printf("MQTT: Error updating the service:%s", err.Error())
		case *BadRequestError:
			logger.Printf("MQTT: Invalid service registration:%s", err.Error())
		default:
			logger.Printf("MQTT: Error updating the service:%s", err.Error())
		}
	}
	logger.Printf("MQTT: Updated service %s", service.ID)
}
