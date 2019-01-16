package catalog

import (
	"fmt"
	"net/url"
)

type MQTTConf struct {
	Client            MQTTClient   `json:"client"`
	AdditionalClients []MQTTClient `json:"additionalClients"`
	CommonRegTopics   []string     `json:"commonRegTopics"`
	CommonWillTopics  []string     `json:"commonWillTopics"`
	TopicPrefix       string       `json:"topicPrefix"`
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
	// internal
	topics     []string
	will       map[string]bool
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