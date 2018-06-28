// Copyright 2016 Fraunhofer Institute for Applied Information Technology FIT

package catalog

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	uuid "github.com/satori/go.uuid"
)

const (
	mqttConnectTimeout = 5 * time.Second
	mqttClientIDPrefix = "SC-"
)

func initMQTTClientOptions(client MQTTClient) (*paho.ClientOptions, error) {
	opts := paho.NewClientOptions() // uses defaults: https://godoc.org/github.com/eclipse/paho.mqtt.golang#NewClientOptions
	opts.AddBroker(client.BrokerURI)
	opts.SetClientID(fmt.Sprintf("%s%s", mqttClientIDPrefix, uuid.NewV4().String()))
	opts.SetConnectTimeout(mqttConnectTimeout)

	if client.Username != "" {
		opts.SetUsername(client.Username)
	}
	if client.Password != "" {
		opts.SetPassword(client.Password)
	}

	// TLS CONFIG
	tlsConfig := &tls.Config{}
	if client.CaFile != "" {
		if !strings.HasPrefix(client.BrokerURI, "ssl") {
			logger.Printf("MQTT: Warning: Configuring TLS with a non-SSL protocol: %s", client.BrokerURI)
		}
		// Import trusted certificates from CAfile.pem.
		// Alternatively, manually add CA certificates to
		// default openssl CA bundle.
		tlsConfig.RootCAs = x509.NewCertPool()
		pemCerts, err := ioutil.ReadFile(client.CaFile)
		if err == nil {
			tlsConfig.RootCAs.AppendCertsFromPEM(pemCerts)
		}
	}
	if client.CertFile != "" && client.KeyFile != "" {
		// Import client certificate/key pair
		cert, err := tls.LoadX509KeyPair(client.CertFile, client.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("error loading client keypair: %s", err)
		}
		// Just to print out the client certificate..
		cert.Leaf, err = x509.ParseCertificate(cert.Certificate[0])
		if err != nil {
			return nil, fmt.Errorf("error parsing client certificate: %s", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}
	opts.SetTLSConfig(tlsConfig)

	return opts, nil
}
