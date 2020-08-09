package main

import (
	"flag"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	mqttclient "github.com/automatedhome/common/pkg/mqttclient"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Settings struct {
	Interval time.Duration
	Duration time.Duration
}

var publishTopic string
var nextPossibleRun time.Time
var settings Settings

var (
	circulationStart = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "circulation_last_start_timestamp",
		Help: "Last start of circulation pump",
	})
	circulationStop = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "circulation_last_stop_timestamp",
		Help: "Last stop of circulation pump",
	})
)

func onMessage(client mqtt.Client, message mqtt.Message) {
	// parse settings
	if strings.HasSuffix(message.Topic(), "/interval") {
		v, err := strconv.ParseInt(string(message.Payload()), 0, 64)
		if err != nil {
			log.Printf("Received incorrect interval payload: '%v'\n", message.Payload())
			return
		}
		settings.Interval = time.Duration(v) * time.Minute
		return
	}
	if strings.HasSuffix(message.Topic(), "/duration") {
		v, err := strconv.ParseInt(string(message.Payload()), 0, 64)
		if err != nil {
			log.Printf("Received incorrect duration payload: '%v'\n", message.Payload())
			return
		}
		settings.Duration = time.Duration(v) * time.Second
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
		log.Printf("Running circulation loop with following settings: %+v", settings)
		nextPossibleRun = time.Now().Add(settings.Interval).Add(settings.Duration)
		circulationStart.SetToCurrentTime()
		if err := mqttclient.Publish(client, publishTopic, 0, false, "1"); err != nil {
			return
		}
		time.Sleep(settings.Duration)
		circulationStop.SetToCurrentTime()
		if err := mqttclient.Publish(client, publishTopic, 0, false, "0"); err != nil {
			return
		}
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

	// Expose metrics
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":7003", nil)

	// wait forever
	select {}
}
