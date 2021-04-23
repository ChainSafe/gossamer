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

	"github.com/ChainSafe/gossamer/lib/common"
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
	require.Contains(t, string(lastMessage), string(expected))
}

func TestHandler_SendBlockImport(t *testing.T) {
	expected := []byte(`{"id":1,"payload":{"best":"hash","height":2,"msg":"block.import","origin":"NetworkInitialSync"},"ts":`)
	GetInstance().SendBlockImport("hash", big.NewInt(2))
	time.Sleep(time.Millisecond)
	require.Contains(t, string(lastMessage), string(expected))
}

var resultCh chan []byte

func TestHandler_SendBlockImport_Race(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(2)

	resultCh = make(chan []byte)

	go func() {
		GetInstance().SendBlockImport("hash", big.NewInt(2))
		// note, we only check the first 101 bytes because the remaining bytes are the timestamp, which we can't estimate
		wg.Done()
	}()

	go func() {
		GetInstance().SendNetworkData(NewNetworkData(1, 2, 3))
		// note, we only check the first 103 bytes because the remaining bytes are the timestamp, which we can't estimate
		wg.Done()
	}()
	wg.Wait()

	expected1 := []byte(`{"id":1,"payload":{"bandwidth_download":2,"bandwidth_upload":3,"msg":"system.interval","peers":1},"ts":`)
	expected2 := []byte(`{"id":1,"payload":{"best":"hash","height":2,"msg":"block.import","origin":"NetworkInitialSync"},"ts":`)

	expected := [][]byte{expected1, expected2}
	var actual [][]byte
	for data := range resultCh {
		actual = append(actual, data)
		if len(actual) == 2 {
			break
		}
	}

	sort.Slice(actual, func(i, j int) bool {
		return bytes.Compare(actual[i], actual[j]) < 0
	})
	require.Contains(t, string(actual[0]), string(expected[0]))
	require.Contains(t, string(actual[1]), string(expected[1]))
}

func TestHandler_SendNetworkData(t *testing.T) {
	expected := []byte(`{"id":1,"payload":{"bandwidth_download":2,"bandwidth_upload":3,"msg":"system.interval","peers":1},"ts":`)
	GetInstance().SendNetworkData(NewNetworkData(1, 2, 3))
	time.Sleep(time.Millisecond)
	require.Contains(t, string(lastMessage), string(expected))
}

func TestHandler_SendBlockIntervalData(t *testing.T) {
	expected := []byte(`{"id":1,"payload":{"best":"0x07b749b6e20fd5f1159153a2e790235018621dd06072a62bcd25e8576f6ff5e6","finalized_hash":"0x687197c11b4cf95374159843e7f46fbcd63558db981aaef01a8bac2a44a1d6b2","finalized_height":32256,"height":32375,"msg":"system.interval","txcount":2,"used_state_cache_size":1886357},"ts":`)
	GetInstance().SendBlockIntervalData(&BlockIntervalData{
		BestHash:           common.MustHexToHash("0x07b749b6e20fd5f1159153a2e790235018621dd06072a62bcd25e8576f6ff5e6"),
		BestHeight:         big.NewInt(32375),
		FinalizedHash:      common.MustHexToHash("0x687197c11b4cf95374159843e7f46fbcd63558db981aaef01a8bac2a44a1d6b2"),
		FinalizedHeight:    big.NewInt(32256),
		TXCount:            2,
		UsedStateCacheSize: 1886357,
	})
	time.Sleep(time.Millisecond)
	require.Contains(t, string(lastMessage), string(expected))
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
		resultCh <- msg
	}
}
