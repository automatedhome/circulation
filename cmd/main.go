package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
)

type EvokDigitalInput struct {
	Bitvalue    int    `json:"bitvalue"`
	ID          int    `json:"glob_dev_id"`
	Value       int    `json:"value"`
	Circuit     string `json:"circuit"`
	Time        int64  `json:"time"`
	Debounce    int    `json:"debounce"`
	CounterMode bool   `json:"counter_mode"`
	Dev         string `json:"dev"`
}

type Settings struct {
	Interval time.Duration
	Duration time.Duration
}

var (
	lastPass        time.Time
	evokCircuit     string
	evokRelay       string
	evokAddress     string
	settings        Settings
	nextPossibleRun time.Time
)

var (
	circulationStart = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "circulation_last_start_timestamp",
		Help: "Last start of circulation pump",
	})
	circulationStop = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "circulation_last_stop_timestamp",
		Help: "Last stop of circulation pump",
	})
	circualtionRuns = promauto.NewCounter(prometheus.CounterOpts{
		Name: "circulation_runs_total",
		Help: "Total number of circulation pump runs",
	})
	circualtionTicks = promauto.NewCounter(prometheus.CounterOpts{
		Name: "circulation_ticks_total",
		Help: "Total number of circulation requests received from a sensor",
	})
)

func digitalInput(address string, circuit string) {
	addr := "ws://" + address + "/ws"
	fmt.Printf("Connecting to EVOK at %s and handling updates for circuit %s\n", addr, circuit)

	conn, _, _, err := ws.DefaultDialer.Dial(context.TODO(), addr)
	if err != nil {
		panic("Connecting to EVOK websocket API failed: " + err.Error())
	}
	defer conn.Close()

	msg := "{\"cmd\":\"filter\", \"devices\":[\"input\"]}"
	if err = wsutil.WriteClientMessage(conn, ws.OpText, []byte(msg)); err != nil {
		panic("Sending websocket message to EVOK failed: " + err.Error())
	}

	var inputs []EvokDigitalInput
	for {
		payload, err := wsutil.ReadServerText(conn)
		if err != nil {
			log.Errorf("Received incorrect data: %#v", err)
		}

		if err := json.Unmarshal(payload, &inputs); err != nil {
			log.Errorf("Could not parse received data: %#v", err)
		}

		if inputs[0].Circuit == evokCircuit && inputs[0].Value == 1 {
			circualtionTicks.Inc()
			go run()
		}
	}
}

func httpHealthCheck(w http.ResponseWriter, r *http.Request) {
	timeout := time.Duration(1 * time.Minute)
	if lastPass.Add(timeout).After(time.Now()) {
		w.WriteHeader(200)
	} else {
		w.WriteHeader(500)
	}
}

func run() {
	if time.Now().After(nextPossibleRun) {
		nextPossibleRun = time.Now().Add(settings.Interval).Add(settings.Duration)
		circualtionRuns.Inc()
		circulationStart.SetToCurrentTime()
		setRelay(true)
		circulationStop.SetToCurrentTime()
		time.Sleep(settings.Duration)
		setRelay(false)
	}
}

func setRelay(value bool) {
	url := "http://" + evokAddress + "/json/relay/" + evokRelay

	state := "0"
	if value {
		state = "1"
	}

	payload := strings.NewReader("{\"value\":\"" + state + "\"}")
	req, _ := http.NewRequest("POST", url, payload)
	req.Header.Add("content-type", "application/json")
	http.DefaultClient.Do(req)
}

func init() {
	addr := flag.String("evok-api-address", "localhost:8080", "EVOK API address (default: localhost:8080)")
	circuit := flag.Int("evok-circuit", 1, "EVOK digital input circuit to which sensor is connected (default: 1)")
	relay := flag.Int("evok-relay", 1, "EVOK relay to which pump is connected (default: 1)")
	duration := flag.Int("duration", 12, "Duration in seconds for how long circulation pump should work (default: 12)")
	interval := flag.Int("interval", 60, "Interval in minutes between last and next possible circualtion start (default: 60)")

	flag.Parse()

	evokCircuit = strconv.Itoa(*circuit)
	evokAddress = *addr
	evokRelay = strconv.Itoa(*relay)
	settings.Interval = time.Duration(int64(*interval) * int64(time.Minute))
	settings.Duration = time.Duration(int64(*duration) * int64(time.Second))

	nextPossibleRun = time.Now()
}

func main() {
	// Expose metrics
	http.Handle("/metrics", promhttp.Handler())
	// Expose healthcheck
	http.HandleFunc("/health", httpHealthCheck)
	go func() {
		if err := http.ListenAndServe(":7003", nil); err != nil {
			panic("HTTP Server failed: " + err.Error())
		}
	}()

	go digitalInput(evokAddress, evokCircuit)

	for {
		time.Sleep(15 * time.Second)
		lastPass = time.Now()
	}
}
