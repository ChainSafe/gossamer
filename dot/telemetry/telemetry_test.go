package telemetry

import (
	"bytes"
	"log"
	"math/big"
	"net/http"
	"os"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
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
	GetInstance().AddConnections(append(testEndpoints, testEndpoint1))

	// Start all tests
	code := m.Run()
	os.Exit(code)
}

func TestHandler_SendMulti(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(4)

	resultCh = make(chan []byte)

	go func() {
		GetInstance().SendMessage(NewTelemetryMessage(
			NewKeyValue("authority", false),
			NewKeyValue("chain", "chain"),
			NewKeyValue("genesis_hash", "hash"),
			NewKeyValue("implementation", "systemName"),
			NewKeyValue("msg", "system.connected"),
			NewKeyValue("name", "nodeName"),
			NewKeyValue("network_id", "netID"),
			NewKeyValue("startup_time", "startTime"),
			NewKeyValue("version", "version")))
		wg.Done()
	}()

	go func() {
		GetInstance().SendMessage(NewTelemetryMessage(
			NewKeyValue("best", "hash"),
			NewKeyValue("height", big.NewInt(2)),
			NewKeyValue("msg", "block.import"),
			NewKeyValue("origin", "NetworkInitialSync")))
		wg.Done()
	}()

	go func() {
		GetInstance().SendMessage(NewTelemetryMessage(
			NewKeyValue("bandwidth_download", 2),
			NewKeyValue("bandwidth_upload", 3),
			NewKeyValue("msg", "system.interval"),
			NewKeyValue("peers", 1)))
		wg.Done()
	}()

	go func() {
		GetInstance().SendMessage(NewTelemetryMessage(
			NewKeyValue("best", "0x07b749b6e20fd5f1159153a2e790235018621dd06072a62bcd25e8576f6ff5e6"),
			NewKeyValue("finalized_hash", "0x687197c11b4cf95374159843e7f46fbcd63558db981aaef01a8bac2a44a1d6b2"), // nolint
			NewKeyValue("finalized_height", 32256), NewKeyValue("height", 32375),                                // nolint
			NewKeyValue("msg", "system.interval"), NewKeyValue("txcount", 2),
			NewKeyValue("used_state_cache_size", 1886357)))
		wg.Done()
	}()

	wg.Wait()

	expected1 := []byte(`{"id":1,"payload":{"bandwidth_download":2,"bandwidth_upload":3,"msg":"system.interval","peers":1},"ts":`)
	expected2 := []byte(`{"id":1,"payload":{"best":"hash","height":2,"msg":"block.import","origin":"NetworkInitialSync"},"ts":`)
	expected3 := []byte(`{"id":1,"payload":{"authority":false,"chain":"chain","genesis_hash":"hash","implementation":"systemName","msg":"system.connected","name":"nodeName","network_id":"netID","startup_time":"startTime","version":"version"},"ts":`)
	expected4 := []byte(`{"id":1,"payload":{"best":"0x07b749b6e20fd5f1159153a2e790235018621dd06072a62bcd25e8576f6ff5e6","finalized_hash":"0x687197c11b4cf95374159843e7f46fbcd63558db981aaef01a8bac2a44a1d6b2","finalized_height":32256,"height":32375,"msg":"system.interval","txcount":2,"used_state_cache_size":1886357},"ts":`) // nolint

	expected := [][]byte{expected3, expected1, expected4, expected2}

	var actual [][]byte
	for data := range resultCh {
		actual = append(actual, data)
		if len(actual) == 4 {
			break
		}
	}

	sort.Slice(actual, func(i, j int) bool {
		return bytes.Compare(actual[i], actual[j]) < 0
	})
	require.Contains(t, string(actual[0]), string(expected[0]))
	require.Contains(t, string(actual[1]), string(expected[1]))
	require.Contains(t, string(actual[2]), string(expected[2]))
	require.Contains(t, string(actual[3]), string(expected[3]))
}

func TestListenerConcurrency(t *testing.T) {
	const qty = 1000
	var wg sync.WaitGroup
	wg.Add(qty)

	resultCh = make(chan []byte)
	for i := 0; i < qty; i++ {
		go func() {
			GetInstance().SendMessage(NewTelemetryMessage(
				NewKeyValue("best", "hash"),
				NewKeyValue("height", big.NewInt(2)),
				NewKeyValue("msg", "block.import"),
				NewKeyValue("origin", "NetworkInitialSync")))
			wg.Done()
		}()
	}
	wg.Wait()
	counter := 0
	for range resultCh {
		counter++
		if counter == qty {
			break
		}
	}
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
