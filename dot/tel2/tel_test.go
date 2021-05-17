package tel2

import (
	"fmt"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"os"
	"testing"
	"time"
)

var upgrader = websocket.Upgrader{}
var resultCh chan []byte

func TestMain(m *testing.M) {
	// start server to listen for websocket connections
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	http.HandleFunc("/", listen)
	go http.ListenAndServe("127.0.0.1:8001", nil)

	time.Sleep(time.Millisecond)
	// instantiate telemetry to connect to websocket (test) server
	var testEndpoints []*genesis.TelemetryEndpoint
	var testEndpoint1 = &genesis.TelemetryEndpoint{
		Endpoint:  "ws://127.0.0.1:8001/",
		Verbosity: 0,
	}
	GetTelInstance().AddConnections(append(testEndpoints, testEndpoint1))

	// Start all tests
	code := m.Run()
	os.Exit(code)
}

func TestTelHandler_SendMessage(t *testing.T) {
	GetTelInstance().SendMessage(NewTelemetryMessage("type", keyValue{
		key:   "k1",
		value: "v1",
	}))
	time.Sleep(time.Second)

	GetTelInstance().SendMessage(NewTelemetryMessage("type", keyValue{
		key:   "k2",
		value: "v2",
	}))
	time.Sleep(time.Second)
}

func TestNewTelemetryMessage(t *testing.T) {
	tm := NewTelemetryMessage("mt", keyValue{
		key:   "key1",
		value: "value1",
	}, keyValue{
		key:   "key2",
		value: "value2",
	})
	fmt.Printf("tm %v\n", len(tm.values))
}

func listen(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error %v\n", err)
	}
	defer c.Close()
	for {
		_, msg, err := c.ReadMessage()
		if err != nil {
			log.Printf("read err %v", err)
			break
		}

		resultCh <- msg
	}
}
