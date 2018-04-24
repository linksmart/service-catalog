// Copyright 2016 Fraunhofer Institute for Applied Information Technology FIT

package catalog

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	uuid "github.com/satori/go.uuid"
)

const (
	mqttRetryInterval = 10 // seconds
)

type MQTTConf struct {
	Client            MQTTClient   `json:"client"`
	AdditionalClients []MQTTClient `json:"additionalClients"`
	CommonRegTopics   []string     `json:"commonRegTopics"`
	CommonWillTopics  []string     `json:"commonWillTopics"`
	TopicPrefix       string       `json:"topicPrefix"`
}

func (c MQTTConf) Validate() error {

	for _, client := range append(c.AdditionalClients, c.Client) {
		if client.BrokerURI == "" {
			continue
		}
		_, err := url.Parse(client.BrokerURI)
		if err != nil {
			return err
		}
		if client.QoS > 2 {
			return fmt.Errorf("QoS must be 0, 1, or 2")
		}
		if len(c.CommonRegTopics) == 0 && len(client.RegTopics) == 0 {
			return fmt.Errorf("regTopics not defined")
		}
	}
	return nil
}

type MQTTClient struct {
	BrokerID   string   `json:"brokerID"`
	BrokerURI  string   `json:"brokerURI"`
	RegTopics  []string `json:"regTopics"`
	WillTopics []string `json:"willTopics"`
	QoS        byte     `json:"qos"`
	Username   string   `json:"username,omitempty"`
	Password   string   `json:"password,omitempty"`
	CaFile     string   `json:"caFile,omitempty"`   // trusted CA certificates file path
	CertFile   string   `json:"certFile,omitempty"` // client certificate file path
	KeyFile    string   `json:"keyFile,omitempty"`  // client private key file path
	topics     []string
	will       map[string]bool
}

type MQTTConnector struct {
	sync.Mutex
	controller          *Controller
	scID                string
	managers            map[string]*ClientManager
	failedRegistrations map[string]MQTTClient
	topicPrefix         string
}

type ClientManager struct {
	uri       string
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
		failedRegistrations: make(map[string]MQTTClient),
	}
	controller.AddListener(c)
	for _, client := range append(mqttConf.AdditionalClients, mqttConf.Client) {
		if client.BrokerURI == "" {
			continue
		}
		if client.BrokerID == "" {
			client.BrokerID = uuid.NewV4().String()
		}
		client.will = make(map[string]bool)
		for _, topic := range append(mqttConf.CommonWillTopics, client.WillTopics...) {
			client.will[topic] = true
		}
		for _, topics := range [][]string{mqttConf.CommonRegTopics, mqttConf.CommonWillTopics, client.RegTopics, client.WillTopics} {
			client.topics = append(client.topics, topics...)
		}

		err := c.register(client)
		if err != nil {
			logger.Printf("MQTT: Error registering subscription: %v. Retrying in %ds", err, mqttRetryInterval)
			c.failedRegistrations[client.BrokerID] = client
		}
	}

	c.topicPrefix = mqttConf.TopicPrefix

	c.retryRegistrations()
}

func (c *MQTTConnector) retryRegistrations() {
	for {
		time.Sleep(mqttRetryInterval * time.Second)
		c.Lock()
		for id, client := range c.failedRegistrations {
			err := c.register(client)
			if err != nil {
				logger.Printf("MQTT: Error registering subscription: %v. Retrying in %ds", err, mqttRetryInterval)
				continue
			}
			delete(c.failedRegistrations, id)
		}
		c.Unlock()
	}
}

func (c *MQTTConnector) register(client MQTTClient) error {
	// TODO
	// the else section can be removed because no one will use two broker configuration blocks for the same client
	if _, exists := c.managers[client.BrokerURI]; !exists { // NO CLIENT FOR THIS BROKER
		manager := &ClientManager{
			uri:           client.BrokerURI,
			subscriptions: make(map[string]*Subscription),
			connector:     c,
			id:            client.BrokerID,
		}

		for _, topic := range client.topics {
			manager.subscriptions[topic] = &Subscription{
				topic:     topic,
				qos:       client.QoS,
				receivers: 1,
				will:      client.will[topic],
				connector: c,
			}
		}

		opts, err := initMQTTClientOptions(client)
		if err != nil {
			return fmt.Errorf("unable to configure MQTT client: %s", err)
		}
		// Add handlers
		opts.SetOnConnectHandler(manager.onConnectHandler)
		opts.SetConnectionLostHandler(manager.onConnectionLostHandler)

		manager.client = paho.NewClient(opts)
		logger.Printf("MQTT: %s: connecting...", manager.uri)

		if token := manager.client.Connect(); token.Wait() && token.Error() != nil {
			return fmt.Errorf("error connecting to broker: %v", token.Error())
		}
		c.managers[client.BrokerURI] = manager

	} else { // THERE IS A CLIENT FOR THIS BROKER
		manager := c.managers[client.BrokerURI]

		for _, topic := range client.topics {
			// TODO: check if another wildcard subscription matches the topic.
			if _, exists := manager.subscriptions[topic]; !exists { // NO SUBSCRIPTION FOR THIS TOPIC

				subscription := &Subscription{
					topic:     topic,
					qos:       client.QoS,
					receivers: 1,
					will:      client.will[topic],
					connector: c,
				}

				// Subscribe
				if token := manager.client.Subscribe(subscription.topic, subscription.qos, subscription.onMessage); token.Wait() && token.Error() != nil {
					return fmt.Errorf("error subscribing: %v", token.Error())
				}
				manager.subscriptions[topic] = subscription
				logger.Printf("MQTT: %s: Subscribed to %s %s", client.BrokerURI, topic, subscription.printIfWill())

			} else { // THERE IS A SUBSCRIPTION FOR THIS TOPIC
				logger.Printf("MQTT: %s: Already subscribed to %s", client.BrokerURI, topic)
				manager.subscriptions[topic].receivers++
			}
		}
	}

	return nil
}

//Controller Listener interface implementation
func (m *MQTTConnector) added(s Service) {
	m.publishAliveService(s)
}

//Controller Listener interface implementation
func (m *MQTTConnector) updated(s Service) {
	m.publishAliveService(s)
}

//Controller Listener interface implementation
func (m *MQTTConnector) deleted(s Service) {
	m.removeService(s)
}

func (connector *MQTTConnector) publishAliveService(s Service) {
	payload, err := json.Marshal(s)
	if err != nil {
		logger.Printf("MQTT: Error parsing json: %s ", err)
		return
	}
	topic := connector.topicPrefix + s.Name + "/" + s.ID + "/alive"
	logger.Printf("MQTT: publishing Service id:%s, topic:%s", s.ID, topic)
	for _, manager := range connector.managers {
		if token := manager.client.Publish(topic, 1, true, payload); token.Wait() && token.Error() != nil {
			logger.Printf("MQTT: %s: Error publishing: %v", manager.uri, token.Error())
		}
	}
}

func (connector *MQTTConnector) removeService(s Service) {
	//remove the retained message
	topic := connector.topicPrefix + s.Name + "/" + s.ID + "/alive"
	logger.Printf("MQTT: removing the retain message topic:%s", topic)
	//payload,err := json.Marshal(nil)
	for _, manager := range connector.managers {
		if token := manager.client.Publish(topic, 1, true, ""); token.Wait() && token.Error() != nil {
			logger.Printf("MQTT: %s: Error publishing: %v", manager.uri, token.Error())
		}
	}

	//update to listening services
	topic = connector.topicPrefix + s.Name + "/" + s.ID + "/dead"
	logger.Printf("MQTT: Publishing deletion Service topic:%s", topic)
	payload, err := json.Marshal(s)
	if err != nil {
		logger.Printf("MQTT: Error parsing json: %s ", err)
		return
	}
	for _, manager := range connector.managers {
		if token := manager.client.Publish(topic, 1, false, payload); token.Wait() && token.Error() != nil {
			logger.Printf("MQTT: %s: Error publishing: %v", manager.uri, token.Error())
		}
	}
}

func (m *ClientManager) onConnectHandler(client paho.Client) {
	logger.Printf("MQTT: %s: Connected.", m.uri)
	m.client = client
	for _, subscription := range m.subscriptions {
		if token := m.client.Subscribe(subscription.topic, subscription.qos, subscription.onMessage); token.Wait() && token.Error() != nil {
			logger.Printf("MQTT: %s: Error subscribing: %v", m.uri, token.Error())
		}
		logger.Printf("MQTT: %s: Subscribed to %s %s", m.uri, subscription.topic, subscription.printIfWill())
	}

	//Add this broker to list of MQTT brokers
	m.addBrokerAsService()
}

func (m *ClientManager) addBrokerAsService() {
	service, err := m.connector.controller.get(m.id)
	if err != nil {
		switch err.(type) {
		case *NotFoundError:
			// First registration
			service = &Service{
				ID:          m.id,
				Name:        "_mqtt._tcp",
				Description: "MQTT Broker",
				Meta: map[string]interface{}{
					"registrator": m.connector.scID,
					"connected":   true,
				},
				APIs: map[string]string{
					APITypeMQTT: m.uri,
				},
			}
			_, err := m.connector.controller.add(*service)
			if err != nil {
				logger.Printf("MQTT: Error registering broker %s: %s", m.id, err)
			}
			logger.Printf("MQTT: %s: Registered as %s", m.uri, m.id)
			return
		default:
			logger.Printf("MQTT: Error retrieving broker %s: %s", m.id, err)
		}
	}
	// MQTTClient re-connect
	service.Meta["connected"] = true
	_, err = m.connector.controller.update(m.id, *service)
	if err != nil {
		logger.Printf("MQTT: Error updating broker %s: %s", m.id, err)
	}
	logger.Printf("MQTT: %s: Updated broker %s", m.uri, m.id)
}

func (m *ClientManager) onConnectionLostHandler(client paho.Client, err error) {
	logger.Printf("MQTT: %s: Connection lost: %v", m.uri, err)
	service, err := m.connector.controller.get(m.id)
	if err != nil {
		logger.Printf("MQTT: Error retrieving broker %s: %s", m.id, err)
	}

	service.Meta["connected"] = false
	_, err = m.connector.controller.update(m.id, *service)
	if err != nil {
		logger.Printf("MQTT: Error updating broker %s: %s", m.id, err)
	}
	logger.Printf("MQTT: %s: Updated broker %s", m.uri, m.id)
}

func (s *Subscription) onMessage(client paho.Client, msg paho.Message) {
	logger.Debugf("MQTT: %s %s", msg.Topic(), msg.Payload())

	var id string
	// Get id from topic. Expects: <prefix>will/<id>
	if s.will {
		if parts := strings.SplitAfter(msg.Topic(), "will/"); len(parts) == 2 && parts[1] != "" {
			s.connector.handleService(Service{ID: parts[1]}, s.will)
			return
		}
	}
	// Get id from topic. Expects: <prefix>service/<id>
	if parts := strings.SplitAfter(msg.Topic(), "service/"); len(parts) == 2 {
		id = parts[1]
	}

	var service Service
	err := json.Unmarshal(msg.Payload(), &service)
	if err != nil {
		logger.Printf("MQTT: Error parsing json: %s : %v", msg.Payload(), err)
		return
	}

	if service.ID == "" && id == "" {
		logger.Printf("MQTT: Invalid registration: ID not provided")
		return
	} else if service.ID == "" {
		logger.Debugf("MQTT: Getting id from topic: %s", id)
		service.ID = id
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
		logger.Printf("MQTT: Removed service with id %s", service.ID)
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
					logger.Printf("MQTT: Error adding service: %s", err.Error())
				case *BadRequestError:
					logger.Printf("MQTT: Invalid service registration: %s", err.Error())
				default:
					logger.Printf("MQTT: Error creating service: %s", err.Error())
				}
			} else {
				logger.Printf("MQTT: Created service with id %s", service.ID)
			}
		case *ConflictError:
			logger.Printf("MQTT: Error updating service: %s", err.Error())
		case *BadRequestError:
			logger.Printf("MQTT: Invalid service registration: %s", err.Error())
		default:
			logger.Printf("MQTT: Error updating service: %s", err.Error())
		}
	} else {
		logger.Printf("MQTT: Updated service with id %s", service.ID)
	}
}
