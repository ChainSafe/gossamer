// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestMailer(t *testing.T, handler http.HandlerFunc) (mailer *Mailer) {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/", handler)

	srv := httptest.NewServer(mux)
	t.Cleanup(func() {
		srv.Close()
	})

	wsAddr := strings.Replace(srv.URL, "http", "ws", -1)
	var testEndpoint1 = &genesis.TelemetryEndpoint{
		Endpoint:  wsAddr,
		Verbosity: 0,
	}

	// instantiate telemetry to connect to websocket (test) server
	testEndpoints := []*genesis.TelemetryEndpoint{testEndpoint1}

	logger := log.New(log.SetWriter(io.Discard))
	const telemetryEnabled = true

	mailer, err := BootstrapMailer(context.Background(), testEndpoints, telemetryEnabled, logger)
	require.NoError(t, err)

	return mailer
}

func TestHandler_SendMulti(t *testing.T) {
	t.Parallel()

	expected := [][]byte{
		[]byte(`{"authority":false,"chain":"chain","genesis_hash":"0x91b171bb158e2d3848fa23a9f1c25182fb8e20313b2c1eb49219da7a70ce90c3","implementation":"systemName","name":"nodeName","network_id":"netID","startup_time":"startTime","version":"0.1","msg":"system.connected","ts":`), //nolint:lll
		[]byte(`{"best":"0x07b749b6e20fd5f1159153a2e790235018621dd06072a62bcd25e8576f6ff5e6","height":2,"origin":"NetworkInitialSync","msg":"block.import","ts":`),                                                                                                                      //nolint:lll
		[]byte(`{"bandwidth_download":2,"bandwidth_upload":3,"peers":1,"msg":"system.interval","ts":`),
		[]byte(`{"best":"0x07b749b6e20fd5f1159153a2e790235018621dd06072a62bcd25e8576f6ff5e6","height":32375,"finalized_hash":"0x687197c11b4cf95374159843e7f46fbcd63558db981aaef01a8bac2a44a1d6b2","finalized_height":32256,"txcount":0,"used_state_cache_size":1234,"msg":"system.interval","ts":`), //nolint:lll
		[]byte(`{"best":[7,183,73,182,226,15,213,241,21,145,83,162,231,144,35,80,24,98,29,208,96,114,166,43,205,37,232,87,111,111,245,230],"height":"32375","msg":"notify.finalized","ts":`),                                                                                                        //nolint:lll
		[]byte(`{"hash":[88,20,174,195,226,133,39,248,31,101,132,30,3,72,114,243,163,3,55,207,108,51,178,210,88,187,166,7,30,55,226,124],"number":"1","msg":"prepared_block_for_proposing","ts":`),                                                                                                  //nolint:lll
		[]byte(`{"ready":1,"future":2,"msg":"txpool.import","ts":`),
		[]byte(`{"authority_id":"authority_id","authority_set_id":"authority_set_id","authorities":"json-stringified-ids-of-authorities","msg":"afg.authority_set","ts`),                                                                   //nolint:lll
		[]byte(`{"hash":[7,183,73,182,226,15,213,241,21,145,83,162,231,144,35,80,24,98,29,208,96,114,166,43,205,37,232,87,111,111,245,230],"number":"1","msg":"afg.finalized_blocks_up_to","ts":`),                                         //nolint:lll
		[]byte(`{"target_hash":[88,20,174,195,226,133,39,248,31,101,132,30,3,72,114,243,163,3,55,207,108,51,178,210,88,187,166,7,30,55,226,124],"target_number":"1","contains_precommits_signed_by":[],"msg":"afg.received_commit","ts":`), //nolint:lll
		[]byte(`{"target_hash":[88,20,174,195,226,133,39,248,31,101,132,30,3,72,114,243,163,3,55,207,108,51,178,210,88,187,166,7,30,55,226,124],"target_number":"1","voter":"","msg":"afg.received_precommit","ts":`),                      //nolint:lll
		[]byte(`{"target_hash":[88,20,174,195,226,133,39,248,31,101,132,30,3,72,114,243,163,3,55,207,108,51,178,210,88,187,166,7,30,55,226,124],"target_number":"1","voter":"","msg":"afg.received_prevote","ts":`),                        //nolint:lll
	}

	messages := []Message{
		NewBandwidth(2, 3, 1),
		NewTxpoolImport(1, 2),

		func(genesisHash common.Hash) Message {
			return NewSystemConnected(false, "chain", &genesisHash,
				"systemName", "nodeName", "netID", "startTime", "0.1")
		}(common.MustHexToHash("0x91b171bb158e2d3848fa23a9f1c25182fb8e20313b2c1eb49219da7a70ce90c3")),

		func(bh common.Hash) Message {
			return NewBlockImport(&bh, big.NewInt(2), "NetworkInitialSync")
		}(common.MustHexToHash("0x07b749b6e20fd5f1159153a2e790235018621dd06072a62bcd25e8576f6ff5e6")),

		func(bestHash, finalisedHash common.Hash) Message {
			return NewBlockInterval(&bestHash, big.NewInt(32375), &finalisedHash,
				big.NewInt(32256), big.NewInt(0), big.NewInt(1234))
		}(
			common.MustHexToHash("0x07b749b6e20fd5f1159153a2e790235018621dd06072a62bcd25e8576f6ff5e6"),
			common.MustHexToHash("0x687197c11b4cf95374159843e7f46fbcd63558db981aaef01a8bac2a44a1d6b2"),
		),

		NewAfgAuthoritySet("authority_id", "authority_set_id", "json-stringified-ids-of-authorities"),
		NewAfgFinalizedBlocksUpTo(
			common.MustHexToHash("0x07b749b6e20fd5f1159153a2e790235018621dd06072a62bcd25e8576f6ff5e6"), "1"),
		NewAfgReceivedCommit(
			common.MustHexToHash("0x5814aec3e28527f81f65841e034872f3a30337cf6c33b2d258bba6071e37e27c"),
			"1", []string{}),
		NewAfgReceivedPrecommit(
			common.MustHexToHash("0x5814aec3e28527f81f65841e034872f3a30337cf6c33b2d258bba6071e37e27c"),
			"1", ""),
		NewAfgReceivedPrevote(
			common.MustHexToHash("0x5814aec3e28527f81f65841e034872f3a30337cf6c33b2d258bba6071e37e27c"),
			"1", ""),

		NewNotifyFinalized(
			common.MustHexToHash("0x07b749b6e20fd5f1159153a2e790235018621dd06072a62bcd25e8576f6ff5e6"),
			"32375"),
		NewPreparedBlockForProposing(
			common.MustHexToHash("0x5814aec3e28527f81f65841e034872f3a30337cf6c33b2d258bba6071e37e27c"),
			"1"),
	}

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	serverHandlerDone := make(chan struct{})

	handler := func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)

		defer func() {
			wsCloseErr := c.Close()
			assert.NoError(t, wsCloseErr)
			close(serverHandlerDone)
		}()

		actual := make([][]byte, len(messages))
		for idx := 0; idx < len(messages); idx++ {
			_, msg, err := c.ReadMessage()
			require.NoError(t, err)

			actual[idx] = msg
		}

		// sort the actual slice in alphabetical order
		sort.Slice(actual, func(i, j int) bool {
			return bytes.Compare(actual[i], actual[j]) < 0
		})

		// sort the expected slice in alphabetical order
		sort.Slice(expected, func(i, j int) bool {
			return bytes.Compare(expected[i], expected[j]) < 0
		})

		for i := range actual {
			assert.Contains(t, string(actual[i]), string(expected[i]))
		}
	}

	mailer := newTestMailer(t, handler)
	var wg sync.WaitGroup
	for _, message := range messages {
		wg.Add(1)
		go func(msg Message) {
			mailer.SendMessage(msg)
			wg.Done()
		}(message)
	}

	wg.Wait()
	<-serverHandlerDone
}

func TestListenerConcurrency(t *testing.T) {
	t.Parallel()

	const qty = 5

	readyWait := new(sync.WaitGroup)
	readyWait.Add(qty)

	timerStartedCh := make(chan struct{})

	ctx, cancel := context.WithCancel(context.Background())

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	serverHandlerDone := make(chan struct{})
	expectedResult := regexp.MustCompile(`^{"best":"0x[0]{64}","height":2,"origin":"NetworkInitialSync",` +
		`"msg":"block.import","ts":"[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:` +
		`[0-9]{2}.[0-9]+-[0-9]{2}:[0-9]{2}"}$`)

	handler := func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)

		defer func() {
			wsCloseErr := c.Close()
			assert.NoError(t, wsCloseErr)
		}()
		close(serverHandlerDone)

		for idx := 0; idx < qty; idx++ {
			_, msg, err := c.ReadMessage()
			require.NoError(t, err)
			assert.True(t,
				expectedResult.MatchString(string(msg)))
		}
	}

	go func() {
		const timeout = 50 * time.Millisecond
		readyWait.Wait()
		ctx, cancel = context.WithTimeout(ctx, timeout)
		close(timerStartedCh)
	}()

	defer cancel()

	mailer := newTestMailer(t, handler)

	doneWait := new(sync.WaitGroup)
	for i := 0; i < qty; i++ {
		doneWait.Add(1)

		go func() {
			defer doneWait.Done()

			readyWait.Done()
			readyWait.Wait()

			<-timerStartedCh

			for ctx.Err() == nil {
				bestHash := common.Hash{}
				msg := NewBlockImport(&bestHash, big.NewInt(2), "NetworkInitialSync")
				mailer.SendMessage(msg)
			}
		}()
	}

	doneWait.Wait()
	<-serverHandlerDone
}

func TestTelemetryMarshalMessage(t *testing.T) {
	tests := map[string]struct {
		message  Message
		expected *regexp.Regexp
	}{
		"AfgAuthoritySet_marshal": {
			message: &AfgAuthoritySet{
				AuthorityID:    "0",
				AuthoritySetID: "0",
				Authorities:    "authorities",
			},
			expected: regexp.MustCompile(`^{"authority_id":"0","authority_set_id":"0","authorities"` +
				`:"authorities","msg":"afg.authority_set","ts":"[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}.` +
				`[0-9]+-[0-9]{2}:[0-9]{2}"}$`),
		},
		"AfgFinalizedBlocksUpTo_marshal": {
			message: &AfgFinalizedBlocksUpTo{
				Hash:   common.Hash{},
				Number: "0",
			},
			expected: regexp.MustCompile(`^{"hash":\[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,` +
				`0,0,0,0,0,0,0,0,0,0,0,0,0,0,0\],"number":"0",` +
				`"msg":"afg.finalized_blocks_up_to","ts":"[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:` +
				`[0-9]{2}.[0-9]+-[0-9]{2}:[0-9]{2}"}$`),
		},
		"AfgReceivedPrecommit_marshal": {
			message: &AfgReceivedPrecommit{
				TargetHash:   common.Hash{},
				TargetNumber: "0",
				Voter:        "0x0",
			},
			expected: regexp.MustCompile(`^{"target_hash":\[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,` +
				`0,0,0,0,0,0,0,0,0,0,0,0,0,0,0\],"target_number":"0","voter":"0x0",` +
				`"msg":"afg.received_precommit","ts":"[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:` +
				`[0-9]{2}.[0-9]+-[0-9]{2}:[0-9]{2}"}$`),
		},
		"AfgReceivedPrevoteTM_marshal": {
			message: &AfgReceivedPrevote{
				TargetHash:   common.Hash{},
				TargetNumber: "0",
				Voter:        "0x0",
			},
			expected: regexp.MustCompile(`^{"target_hash":\[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,` +
				`0,0,0,0,0,0,0,0,0,0,0,0,0,0,0\],"target_number":"0","voter":"0x0",` +
				`"msg":"afg.received_prevote","ts":"[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:` +
				`[0-9]{2}.[0-9]+-[0-9]{2}:[0-9]{2}"}$`),
		},
		"AfgReceivedCommit_marshal": {
			message: &AfgReceivedCommit{
				TargetHash:                 common.Hash{},
				TargetNumber:               "0",
				ContainsPrecommitsSignedBy: []string{"0x0", "0x1"},
			},
			expected: regexp.MustCompile(`^{"target_hash":\[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,` +
				`0,0,0,0,0,0,0,0,0,0,0,0,0,0,0\],"target_number":"0","contains_precommits_signed_by":\["0x0","0x1"\],` +
				`"msg":"afg.received_commit","ts":"[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:` +
				`[0-9]{2}.[0-9]+-[0-9]{2}:[0-9]{2}"}$`),
		},
		"BlockImport_marshal": {
			message: &BlockImport{
				BestHash: &common.Hash{},
				Height:   &big.Int{},
				Origin:   "0x0",
			},
			expected: regexp.MustCompile(`^{"best":"0x[0]{64}","height":0,"origin":"0x0",` +
				`"msg":"block.import","ts":"[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:` +
				`[0-9]{2}.[0-9]+-[0-9]{2}:[0-9]{2}"}$`),
		},
		"NotifyFinalized_marshal": {
			message: &NotifyFinalized{
				Best:   common.Hash{},
				Height: "0",
			},
			expected: regexp.MustCompile(`^{"best":\[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,` +
				`0,0,0,0,0,0,0,0,0,0,0,0,0,0,0\],"height":"0",` +
				`"msg":"notify.finalized","ts":"[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:` +
				`[0-9]{2}.[0-9]+-[0-9]{2}:[0-9]{2}"}$`),
		},
		"PreparedBlockForProposing_marshal": {
			message: &PreparedBlockForProposing{
				Hash:   common.Hash{},
				Number: "0",
			},
			expected: regexp.MustCompile(`^{"hash":\[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,` +
				`0,0,0,0,0,0,0,0,0,0,0,0,0,0,0\],"number":"0",` +
				`"msg":"prepared_block_for_proposing","ts":"[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:` +
				`[0-9]{2}.[0-9]+-[0-9]{2}:[0-9]{2}"}$`),
		},
		"SystemConnected_marshal": {
			message: &SystemConnected{
				Authority:      true,
				Chain:          "0x0",
				GenesisHash:    &common.Hash{},
				Implementation: "gossamer",
				Name:           "gossamer",
				NetworkID:      "0",
				StartupTime:    "0ms",
				Version:        "0",
			},
			expected: regexp.MustCompile(`^{"authority":true,"chain":"0x0","genesis_hash":"0x[0]{64}",` +
				`"implementation":"gossamer","name":"gossamer","network_id":"0","startup_time":"0ms",` +
				`"version":"0","msg":"system.connected","ts":"[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:` +
				`[0-9]{2}.[0-9]+-[0-9]{2}:[0-9]{2}"}$`),
		},
		"SystemInterval_marshal": {
			message: &SystemInterval{
				BandwidthDownload:  1.5,
				BandwidthUpload:    1.5,
				Peers:              1,
				BestHash:           &common.Hash{},
				BestHeight:         &big.Int{},
				FinalisedHash:      &common.Hash{},
				FinalisedHeight:    &big.Int{},
				TxCount:            &big.Int{},
				UsedStateCacheSize: &big.Int{},
			},
			expected: regexp.MustCompile(`^{"bandwidth_download":1.5,"bandwidth_upload":1.5,"peers":1,` +
				`"best":"0x[0]{64}","height":0,"finalized_hash":"0x[0]{64}","finalized_height":0,` +
				`"txcount":0,"used_state_cache_size":0,"msg":"system.interval","ts":` +
				`"[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:` +
				`[0-9]{2}.[0-9]+-[0-9]{2}:[0-9]{2}"}$`),
		},
		"TxpoolImport_marshal": {
			message: &TxpoolImport{
				Ready:  11,
				Future: 10,
			},
			expected: regexp.MustCompile(`^{"ready":11,"future":10,` +
				`"msg":"txpool.import","ts":"[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:` +
				`[0-9]{2}.[0-9]+-[0-9]{2}:[0-9]{2}"}$`),
		},
	}

	for tname, tt := range tests {
		tt := tt
		t.Run(tname, func(t *testing.T) {
			t.Parallel()

			telemetryBytes, err := json.Marshal(tt.message)
			require.NoError(t, err)
			require.True(t, tt.expected.MatchString(string(telemetryBytes)))
		})
	}
}
