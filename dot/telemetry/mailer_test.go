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

	wsAddr := strings.ReplaceAll(srv.URL, "http", "ws")
	var testEndpoint1 = &genesis.TelemetryEndpoint{
		Endpoint:  wsAddr,
		Verbosity: 0,
	}

	// instantiate telemetry to connect to websocket (test) server
	testEndpoints := []*genesis.TelemetryEndpoint{testEndpoint1}

	logger := log.New(log.SetWriter(io.Discard))

	mailer, err := BootstrapMailer(context.Background(), testEndpoints, logger)
	require.NoError(t, err)

	return mailer
}

func TestHandler_SendMulti(t *testing.T) {
	t.Parallel()

	firstHash := common.MustHexToHash("0x07b749b6e20fd5f1159153a2e790235018621dd06072a62bcd25e8576f6ff5e6")
	secondHash := common.MustHexToHash("0x5814aec3e28527f81f65841e034872f3a30337cf6c33b2d258bba6071e37e27c")

	expected := [][]byte{
		[]byte(`{"authority":false,"chain":"chain","genesis_hash":"0x07b749b6e20fd5f1159153a2e790235018621dd06072a62bcd25e8576f6ff5e6","implementation":"systemName","name":"nodeName","network_id":"netID","startup_time":"startTime","version":"0.1","msg":"system.connected","ts":`), //nolint:lll
		[]byte(`{"best":"0x07b749b6e20fd5f1159153a2e790235018621dd06072a62bcd25e8576f6ff5e6","height":2,"origin":"NetworkInitialSync","msg":"block.import","ts":`),                                                                                                                      //nolint:lll
		[]byte(`{"bandwidth_download":2,"bandwidth_upload":3,"peers":1,"msg":"system.interval","ts":`),
		[]byte(`{"best":"0x07b749b6e20fd5f1159153a2e790235018621dd06072a62bcd25e8576f6ff5e6","height":32375,"finalized_hash":"0x5814aec3e28527f81f65841e034872f3a30337cf6c33b2d258bba6071e37e27c","finalized_height":32256,"txcount":0,"used_state_cache_size":1234,"msg":"system.interval","ts":`), //nolint:lll
		[]byte(`{"best":"0x07b749b6e20fd5f1159153a2e790235018621dd06072a62bcd25e8576f6ff5e6","height":"32375","msg":"notify.finalized","ts":`),                                                                                                                                                      //nolint:lll
		[]byte(`{"hash":"0x5814aec3e28527f81f65841e034872f3a30337cf6c33b2d258bba6071e37e27c","number":"1","msg":"prepared_block_for_proposing","ts":`),                                                                                                                                              //nolint:lll
		[]byte(`{"ready":1,"future":2,"msg":"txpool.import","ts":`),
		[]byte(`{"authority_id":"authority_id","authority_set_id":"authority_set_id","authorities":"json-stringified-ids-of-authorities","msg":"afg.authority_set","ts`),                       //nolint:lll
		[]byte(`{"hash":"0x07b749b6e20fd5f1159153a2e790235018621dd06072a62bcd25e8576f6ff5e6","number":"1","msg":"afg.finalized_blocks_up_to","ts":`),                                           //nolint:lll
		[]byte(`{"target_hash":"0x5814aec3e28527f81f65841e034872f3a30337cf6c33b2d258bba6071e37e27c","target_number":"1","contains_precommits_signed_by":[],"msg":"afg.received_commit","ts":`), //nolint:lll
		[]byte(`{"target_hash":"0x5814aec3e28527f81f65841e034872f3a30337cf6c33b2d258bba6071e37e27c","target_number":"1","voter":"","msg":"afg.received_precommit","ts":`),                      //nolint:lll
		[]byte(`{"target_hash":"0x5814aec3e28527f81f65841e034872f3a30337cf6c33b2d258bba6071e37e27c","target_number":"1","voter":"","msg":"afg.received_prevote","ts":`),                        //nolint:lll
	}

	messages := []Message{
		NewBandwidth(2, 3, 1),
		NewTxpoolImport(1, 2),
		NewSystemConnected(false, "chain", &firstHash,
			"systemName", "nodeName", "netID", "startTime", "0.1"),
		NewBlockImport(&firstHash, 2, "NetworkInitialSync"),
		NewBlockInterval(&firstHash, 32375, &secondHash,
			32256, big.NewInt(0), big.NewInt(1234)),
		NewAfgAuthoritySet("authority_id", "authority_set_id", "json-stringified-ids-of-authorities"),
		NewAfgFinalizedBlocksUpTo(firstHash, "1"),
		NewAfgReceivedCommit(secondHash, "1", []string{}),
		NewAfgReceivedPrecommit(secondHash, "1", ""),
		NewAfgReceivedPrevote(secondHash, "1", ""),
		NewNotifyFinalized(firstHash, "32375"),
		NewPreparedBlockForProposing(secondHash, "1"),
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
		`[0-9]{2}.[0-9]+Z|([+-][0-9]{2}:[0-9]{2})"}$`)

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
				msg := NewBlockImport(&bestHash, 2, "NetworkInitialSync")
				mailer.SendMessage(msg)
			}
		}()
	}

	doneWait.Wait()
	<-serverHandlerDone
}

func TestTelemetryMarshalMessage(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		message  Message
		expected string
	}{
		"AfgAuthoritySet_marshal": {
			message: &AfgAuthoritySet{
				AuthorityID:    "0",
				AuthoritySetID: "0",
				Authorities:    "authorities",
			},
			expected: `^{"authority_id":"0","authority_set_id":"0","authorities"` +
				`:"authorities","msg":"afg.authority_set","ts":"[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}.` +
				`[0-9]+Z|([+-][0-9]{2}:[0-9]{2})"}$`,
		},
		"AfgFinalizedBlocksUpTo_marshal": {
			message: &AfgFinalizedBlocksUpTo{
				Hash:   common.Hash{},
				Number: "0",
			},
			expected: `^{"hash":"0x[0]{64}","number":"0",` +
				`"msg":"afg.finalized_blocks_up_to","ts":"[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:` +
				`[0-9]{2}.[0-9]+Z|([+-][0-9]{2}:[0-9]{2})"}$`,
		},
		"AfgReceivedPrecommit_marshal": {
			message: &AfgReceivedPrecommit{
				TargetHash:   common.Hash{},
				TargetNumber: "0",
				Voter:        "0x0",
			},
			expected: `^{"target_hash":"0x[0]{64}","target_number":"0","voter":"0x0",` +
				`"msg":"afg.received_precommit","ts":"[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:` +
				`[0-9]{2}.[0-9]+Z|([+-][0-9]{2}:[0-9]{2})"}$`,
		},
		"AfgReceivedPrevoteTM_marshal": {
			message: &AfgReceivedPrevote{
				TargetHash:   common.Hash{},
				TargetNumber: "0",
				Voter:        "0x0",
			},
			expected: `^{"target_hash":"0x[0]{64}","target_number":"0","voter":"0x0",` +
				`"msg":"afg.received_prevote","ts":"[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:` +
				`[0-9]{2}.[0-9]+Z|([+-][0-9]{2}:[0-9]{2})"}$`,
		},
		"AfgReceivedCommit_marshal": {
			message: &AfgReceivedCommit{
				TargetHash:                 common.Hash{},
				TargetNumber:               "0",
				ContainsPrecommitsSignedBy: []string{"0x0", "0x1"},
			},
			expected: `^{"target_hash":"0x[0]{64}","target_number":"0",` +
				`"contains_precommits_signed_by":\["0x0","0x1"\],` +
				`"msg":"afg.received_commit","ts":"[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:` +
				`[0-9]{2}.[0-9]+Z|([+-][0-9]{2}:[0-9]{2})"}$`,
		},
		"BlockImport_marshal": {
			message: &BlockImport{
				BestHash: &common.Hash{},
				Origin:   "0x0",
			},
			expected: `^{"best":"0x[0]{64}","height":0,"origin":"0x0",` +
				`"msg":"block.import","ts":"[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:` +
				`[0-9]{2}.[0-9]+Z|([+-][0-9]{2}:[0-9]{2})"}$`,
		},
		"NotifyFinalized_marshal": {
			message: &NotifyFinalized{
				Best:   common.Hash{},
				Height: "0",
			},
			expected: `^{"best":"0x[0]{64}","height":"0",` +
				`"msg":"notify.finalized","ts":"[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:` +
				`[0-9]{2}.[0-9]+Z|([+-][0-9]{2}:[0-9]{2})"}$`,
		},
		"PreparedBlockForProposing_marshal": {
			message: &PreparedBlockForProposing{
				Hash:   common.Hash{},
				Number: "0",
			},
			expected: `^{"hash":"0x[0]{64}","number":"0",` +
				`"msg":"prepared_block_for_proposing","ts":"[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:` +
				`[0-9]{2}.[0-9]+Z|([+-][0-9]{2}:[0-9]{2})"}$`,
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
			expected: `^{"authority":true,"chain":"0x0","genesis_hash":"0x[0]{64}",` +
				`"implementation":"gossamer","name":"gossamer","network_id":"0","startup_time":"0ms",` +
				`"version":"0","msg":"system.connected","ts":"[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:` +
				`[0-9]{2}.[0-9]+Z|([+-][0-9]{2}:[0-9]{2})"}$`,
		},
		"SystemInterval_marshal": {
			message: &SystemInterval{
				BandwidthDownload:  1.5,
				BandwidthUpload:    1.5,
				Peers:              1,
				BestHash:           &common.Hash{},
				FinalisedHash:      &common.Hash{},
				TxCount:            &big.Int{},
				UsedStateCacheSize: &big.Int{},
			},
			expected: `^{"bandwidth_download":1.5,"bandwidth_upload":1.5,"peers":1,` +
				`"best":"0x[0]{64}","finalized_hash":"0x[0]{64}",` +
				`"txcount":0,"used_state_cache_size":0,"msg":"system.interval","ts":` +
				`"[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:` +
				`[0-9]{2}.[0-9]+Z|([+-][0-9]{2}:[0-9]{2})"}$`,
		},
		"TxpoolImport_marshal": {
			message: &TxpoolImport{
				Ready:  11,
				Future: 10,
			},
			expected: `^{"ready":11,"future":10,` +
				`"msg":"txpool.import","ts":"[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:` +
				`[0-9]{2}.[0-9]+Z|([+-][0-9]{2}:[0-9]{2})"}$`,
		},
	}

	for tname, tt := range tests {
		tt := tt
		t.Run(tname, func(t *testing.T) {
			t.Parallel()
			telemetryBytes, err := json.Marshal(tt.message)
			require.NoError(t, err)
			assert.Regexp(t, tt.expected, string(telemetryBytes))
		})
	}
}
