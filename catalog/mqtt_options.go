// Copyright 2016 Fraunhofer Institute for Applied Information Technology FIT

package catalog

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	uuid "github.com/satori/go.uuid"
)

const (
	mqttConnectTimeout = 5 * time.Second
	mqttClientIDPrefix = "SC-"
)

func initMQTTClientOptions(broker Broker) (*paho.ClientOptions, error) {
	opts := paho.NewClientOptions() // uses defaults: https://godoc.org/github.com/eclipse/paho.mqtt.golang#NewClientOptions
	opts.AddBroker(broker.URL)
	opts.SetClientID(fmt.Sprintf("%s%s", mqttClientIDPrefix, uuid.NewV4().String()))
	opts.SetConnectTimeout(mqttConnectTimeout)

	if broker.Username != "" {
		opts.SetUsername(broker.Username)
		opts.SetPassword(broker.Password)
	}

	// TLS CONFIG:
	//	based on https://github.com/eclipse/paho.mqtt.golang/blob/master/cmd/ssl/main.go
	// Import trusted certificates from CAfile.pem.
	// Alternatively, manually add CA certificates to
	// default openssl CA bundle.
	certpool := x509.NewCertPool()
	pemCerts, err := ioutil.ReadFile(broker.CaFile)
	if err == nil {
		certpool.AppendCertsFromPEM(pemCerts)
	}
	// Import client certificate/key pair
	cert, err := tls.LoadX509KeyPair(broker.CertFile, broker.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("error loading client keypair: %s", err)
	}
	// Just to print out the client certificate..
	cert.Leaf, err = x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return nil, fmt.Errorf("error parsing client certificate: %s", err)
	}
	// Create tls.Config with desired tls properties
	tlsConfig := &tls.Config{
		// RootCAs = certs used to verify server cert.
		RootCAs: certpool,
		// ClientAuth = whether to request cert from server.
		// Since the server is set up for SSL, this happens
		// anyways.
		ClientAuth: tls.NoClientCert,
		// ClientCAs = certs used to validate client cert.
		ClientCAs: nil,
		// InsecureSkipVerify = verify that cert contents
		// match server. IP matches what is in cert etc.
		InsecureSkipVerify: true,
		// Certificates = list of certs client sends to server.
		Certificates: []tls.Certificate{cert},
	}

	opts.SetTLSConfig(tlsConfig)

	return opts, nil
}
