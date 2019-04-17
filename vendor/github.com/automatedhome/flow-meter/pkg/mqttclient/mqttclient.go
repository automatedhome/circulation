package mqttclient

import (
	"fmt"
	"log"
	"net/url"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func createOptions(clientID string, uri *url.URL) *mqtt.ClientOptions {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s", uri.Host))
	//opts.SetUsername(uri.User.Username())
	//password, _ := uri.User.Password()
	//opts.SetPassword(password)
	opts.SetClientID(clientID)
	opts.SetKeepAlive(2 * time.Second)
	opts.SetPingTimeout(1 * time.Second)
	opts.SetAutoReconnect(true)
	return opts
}

// Connect will create new mqtt client
func Connect(clientID string, uri *url.URL) mqtt.Client {
	opts := createOptions(clientID, uri)
	client := mqtt.NewClient(opts)
	token := client.Connect()
	for !token.WaitTimeout(3 * time.Second) {
	}
	if err := token.Error(); err != nil {
		log.Fatal(err)
	}
	return client
}

// New will create new mqtt client and start handling messages from specified topic
func New(id string, uri *url.URL, topics []string, callback mqtt.MessageHandler) {
	client := Connect(id, uri)

	topicsMap := make(map[string]byte)

	for _, s := range topics {
		topicsMap[s] = 0
	}
	client.SubscribeMultiple(topicsMap, callback)
}
