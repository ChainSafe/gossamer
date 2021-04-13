package telemetry

import (
	"log"
	"math/big"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

var upgrader = websocket.Upgrader{}
var lastMessage []byte

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
	GetInstance().AddConnections(append(testEndpoints, testEndpoint1))

	// Start all tests
	code := m.Run()
	os.Exit(code)
}
func TestHandler_SendConnection(t *testing.T) {
	expected := []byte(`{"id":1,"payload":{"authority":false,"chain":"chain","config":"","genesis_hash":"hash","implementation":"systemName","msg":"system.connected","name":"nodeName","network_id":"netID","startup_time":"startTime","version":"version"},"ts":`)
	data := &ConnectionData{
		Authority:     false,
		Chain:         "chain",
		GenesisHash:   "hash",
		SystemName:    "systemName",
		NodeName:      "nodeName",
		SystemVersion: "version",
		NetworkID:     "netID",
		StartTime:     "startTime",
	}
	GetInstance().SendConnection(data)
	time.Sleep(time.Millisecond)
	// note, we only check the first 234 bytes because the remaining bytes are the timestamp, which we can't estimate
	require.Equal(t, expected, lastMessage[:234])
}

func TestHandler_SendBlockImport(t *testing.T) {
	expected := []byte(`{"id":1,"payload":{"best":"hash","height":2,"msg":"block.import","origin":"NetworkInitialSync"},"ts":`)
	GetInstance().SendBlockImport("hash", big.NewInt(2))
	time.Sleep(time.Millisecond)
	// note, we only check the first 101 bytes because the remaining bytes are the timestamp, which we can't estimate
	require.Equal(t, expected, lastMessage[:101])
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
		lastMessage = msg
	}
}
