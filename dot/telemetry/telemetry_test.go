// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package telemetry

import (
	"bytes"
	"fmt"
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
	expected := [][]byte{
		[]byte(`{"authority":false,"chain":"chain","genesis_hash":"0x91b171bb158e2d3848fa23a9f1c25182fb8e20313b2c1eb49219da7a70ce90c3","implementation":"systemName","msg":"system.connected","name":"nodeName","network_id":"netID","startup_time":"startTime","ts":`), //nolint:lll
		[]byte(`{"best":"0x07b749b6e20fd5f1159153a2e790235018621dd06072a62bcd25e8576f6ff5e6","height":2,"msg":"block.import","origin":"NetworkInitialSync","ts":`),                                                                                                      //nolint:lll
		[]byte(`{"bandwidth_download":2,"bandwidth_upload":3,"msg":"system.interval","peers":1,"ts":`),
		[]byte(`{"best":"0x07b749b6e20fd5f1159153a2e790235018621dd06072a62bcd25e8576f6ff5e6","finalized_hash":"0x687197c11b4cf95374159843e7f46fbcd63558db981aaef01a8bac2a44a1d6b2","finalized_height":32256,"height":32375,"msg":"system.interval","ts":`), //nolint:lll
		[]byte(`{"best":"0x07b749b6e20fd5f1159153a2e790235018621dd06072a62bcd25e8576f6ff5e6","height":"32375","msg":"notify.finalized","ts":`),                                                                                                             //nolint:lll
		[]byte(`{"hash":"0x5814aec3e28527f81f65841e034872f3a30337cf6c33b2d258bba6071e37e27c","msg":"prepared_block_for_proposing","number":"1","ts":`),                                                                                                     //nolint:lll
		[]byte(`{"future":2,"msg":"txpool.import","ready":1,"ts":`),
		[]byte(`{"authorities":"json-stringified-ids-of-authorities","authority_id":"authority_id","authority_set_id":"authority_set_id","msg":"afg.authority_set","ts`),                       //nolint:lll
		[]byte(`{"hash":"0x07b749b6e20fd5f1159153a2e790235018621dd06072a62bcd25e8576f6ff5e6","msg":"afg.finalized_blocks_up_to","number":"1","ts":`),                                           //nolint:lll
		[]byte(`{"contains_precommits_signed_by":[],"msg":"afg.received_commit","target_hash":"0x5814aec3e28527f81f65841e034872f3a30337cf6c33b2d258bba6071e37e27c","target_number":"1","ts":`), //nolint:lll
		[]byte(`{"msg":"afg.received_precommit","target_hash":"0x5814aec3e28527f81f65841e034872f3a30337cf6c33b2d258bba6071e37e27c","target_number":"1","ts":`),                                 //nolint:lll
		[]byte(`{"msg":"afg.received_prevote","target_hash":"0x5814aec3e28527f81f65841e034872f3a30337cf6c33b2d258bba6071e37e27c","target_number":"1","ts":`),                                   //nolint:lll
	}

	messages := []Message{
		NewBandwidthTM(2, 3, 1),
		NewTxpoolImportTM(1, 2),

		func(genesisHash common.Hash) Message {
			return NewSystemConnectedTM(false, "chain", &genesisHash,
				"systemName", "nodeName", "netID", "startTime", "0.1")
		}(common.MustHexToHash("0x91b171bb158e2d3848fa23a9f1c25182fb8e20313b2c1eb49219da7a70ce90c3")),

		func(bh common.Hash) Message {
			return NewBlockImportTM(&bh, big.NewInt(2), "NetworkInitialSync")
		}(common.MustHexToHash("0x07b749b6e20fd5f1159153a2e790235018621dd06072a62bcd25e8576f6ff5e6")),

		func(bestHash, finalisedHash common.Hash) Message {
			return NewBlockIntervalTM(&bestHash, big.NewInt(32375), &finalisedHash,
				big.NewInt(32256), big.NewInt(0), big.NewInt(1234))
		}(
			common.MustHexToHash("0x07b749b6e20fd5f1159153a2e790235018621dd06072a62bcd25e8576f6ff5e6"),
			common.MustHexToHash("0x687197c11b4cf95374159843e7f46fbcd63558db981aaef01a8bac2a44a1d6b2"),
		),

		NewAfgAuthoritySetTM("authority_id", "authority_set_id", "json-stringified-ids-of-authorities"),
		NewAfgFinalizedBlocksUpToTM(
			common.MustHexToHash("0x07b749b6e20fd5f1159153a2e790235018621dd06072a62bcd25e8576f6ff5e6"), "1"),
		NewAfgReceivedCommitTM(
			common.MustHexToHash("0x5814aec3e28527f81f65841e034872f3a30337cf6c33b2d258bba6071e37e27c"),
			"1", []string{}),
		NewAfgReceivedPrecommitTM(
			common.MustHexToHash("0x5814aec3e28527f81f65841e034872f3a30337cf6c33b2d258bba6071e37e27c"),
			"1", ""),
		NewAfgReceivedPrevoteTM(
			common.MustHexToHash("0x5814aec3e28527f81f65841e034872f3a30337cf6c33b2d258bba6071e37e27c"),
			"1", ""),

		NewNotifyFinalizedTM(
			common.MustHexToHash("0x07b749b6e20fd5f1159153a2e790235018621dd06072a62bcd25e8576f6ff5e6"),
			"32375"),
		NewPreparedBlockForProposingTM(
			common.MustHexToHash("0x5814aec3e28527f81f65841e034872f3a30337cf6c33b2d258bba6071e37e27c"),
			"1"),
	}

	resultCh = make(chan []byte)

	var wg sync.WaitGroup
	for _, message := range messages {
		wg.Add(1)
		go func(msg Message) {
			GetInstance().SendMessage(msg)
			wg.Done()
		}(message)
	}

	wg.Wait()

	var actual [][]byte
	for data := range resultCh {
		actual = append(actual, data)
		if len(actual) == len(expected) {
			break
		}
	}

	sort.Slice(expected, func(i, j int) bool {
		return bytes.Compare(expected[i], expected[j]) < 0
	})

	sort.Slice(actual, func(i, j int) bool {
		return bytes.Compare(actual[i], actual[j]) < 0
	})

	for i := range actual {
		require.Contains(t, string(actual[i]), string(expected[i]))
	}
}

func TestListenerConcurrency(t *testing.T) {
	const qty = 1000
	var wg sync.WaitGroup
	wg.Add(qty)

	resultCh = make(chan []byte)
	for i := 0; i < qty; i++ {
		go func() {
			bestHash := common.Hash{}
			GetInstance().SendMessage(NewBlockImportTM(&bestHash, big.NewInt(2), "NetworkInitialSync"))

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

func TestDisableInstance(t *testing.T) {
	const qty = 1000
	var wg sync.WaitGroup
	wg.Add(qty)

	resultCh = make(chan []byte)
	for i := 0; i < qty; i++ {
		if i == qty/2 {
			GetInstance().Initialise(false)
		}
		go func() {
			bh := common.MustHexToHash("0x07b749b6e20fd5f1159153a2e790235018621dd06072a62bcd25e8576f6ff5e6")
			GetInstance().SendMessage(NewBlockImportTM(&bh, big.NewInt(2), "NetworkInitialSync"))
			wg.Done()
		}()
	}
	wg.Wait()
	counter := 0
	tk := time.NewTicker(time.Second * 2)
main:
	for {
		select {
		case <-tk.C:
			break main
		case <-resultCh:
			counter++
		}
	}
	tk.Stop()

	require.LessOrEqual(t, counter, qty/2)
}

// TestInfiniteListener starts loop that print out data received on websocket ws://localhost:8001/
//  this can be useful to see what data is sent to telemetry server
func TestInfiniteListener(t *testing.T) {
	t.Skip()
	resultCh = make(chan []byte)
	for data := range resultCh {
		fmt.Printf("Data %s\n", data)
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
