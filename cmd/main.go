package main

import (
	"flag"
	"log"
	"net/url"
	"strconv"
	"time"

	mqttclient "github.com/automatedhome/flow-meter/pkg/mqttclient"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Settings struct {
	Interval time.Duration
	Duration time.Duration
}

var publishTopic string
var nextPossibleRun time.Time
var settings Settings

func onMessage(client mqtt.Client, message mqtt.Message) {
	value, err := strconv.ParseBool(string(message.Payload()))
	if err != nil {
		log.Printf("Received incorrect message payload: '%v'\n", message.Payload())
		return
	}

	circulation(client, value)
}

func circulation(client mqtt.Client, value bool) {
	if !value {
		return
	}

	// check if INTERVAL passed
	if time.Now().After(nextPossibleRun) {
		nextPossibleRun = time.Now().Add(settings.Interval)
		client.Publish(publishTopic, 0, false, "1")
		time.Sleep(settings.Duration)
		client.Publish(publishTopic, 0, false, "0")
	}
}

func main() {
	broker := flag.String("broker", "tcp://127.0.0.1:1883", "The full url of the MQTT server to connect to ex: tcp://127.0.0.1:1883")
	clientID := flag.String("clientid", "circulation", "A clientid for the connection")
	inTopic := flag.String("inTopic", "evok/input/3/value", "MQTT topic with a current pin state")
	outTopic := flag.String("outTopic", "evok/relay/4/set", "MQTT topic with a relay responsible for circulation pump")
	//settingsTopic := flag.String("settingsTopic", "settings/circulation", "MQTT topic with circulation settings")
	flag.Parse()

	publishTopic = *outTopic

	// Read it from MQTT settings topic
	settings.Interval = 60 * time.Minute
	settings.Duration = 12 * time.Second

	nextPossibleRun = time.Now()
	brokerURL, _ := url.Parse(*broker)
	mqttclient.New(*clientID, brokerURL, *inTopic, onMessage)

	log.Printf("Connected to %s as %s and waiting for messages\n", *broker, *clientID)

	// wait forever
	select {}
}
