package main

import (
	"flag"
	"log"
	"net/url"
	"strconv"
	"strings"
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
	// parse settings
	if strings.HasSuffix(message.Topic(), "/interval") {
		v, err := strconv.ParseInt(string(message.Payload()), 0, 64)
		if err != nil {
			log.Printf("Received incorrect interval payload: '%v'\n", message.Payload())
			return
		}
		settings.Interval = time.Duration(v)
		return
	}
	if strings.HasSuffix(message.Topic(), "/duration") {
		v, err := strconv.ParseInt(string(message.Payload()), 0, 64)
		if err != nil {
			log.Printf("Received incorrect duration payload: '%v'\n", message.Payload())
			return
		}
		settings.Duration = time.Duration(v)
		return
	}

	// parse values
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
		log.Printf("Running circulation loop")
		nextPossibleRun = time.Now().Add(settings.Interval).Add(settings.Duration)
		client.Publish(publishTopic, 0, false, "1")
		time.Sleep(settings.Duration)
		client.Publish(publishTopic, 0, false, "0")
	}
}

func main() {
	broker := flag.String("broker", "tcp://127.0.0.1:1883", "The full url of the MQTT server to connect to ex: tcp://127.0.0.1:1883")
	clientID := flag.String("clientid", "circulation", "A clientid for the connection")
	inTopic := flag.String("inTopic", "circulation/detect", "MQTT topic with a current pin state")
	outTopic := flag.String("outTopic", "circulation/pump", "MQTT topic with a relay responsible for circulation pump")
	settingsTopic := flag.String("settingsTopic", "circulation/settings/+", "MQTT topic(s) with circulation settings")
	flag.Parse()

	// set default values
	settings.Interval = 60 * time.Minute
	settings.Duration = 12 * time.Second

	publishTopic = *outTopic

	nextPossibleRun = time.Now()
	brokerURL, _ := url.Parse(*broker)
	mqttclient.New(*clientID, brokerURL, []string{*inTopic, *settingsTopic}, onMessage)

	log.Printf("Connected to %s as %s and waiting for messages\n", *broker, *clientID)

	// wait forever
	select {}
}
