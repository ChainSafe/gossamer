// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package subscription

import (
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/ChainSafe/gossamer/dot/rpc/modules/mocks"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/grandpa"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/runtime/wasmer"
	"github.com/ChainSafe/gossamer/lib/transaction"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockWSConnAPI struct {
	lastMessage BaseResponseJSON
}

func (m *mockWSConnAPI) safeSend(msg interface{}) {
	m.lastMessage = msg.(BaseResponseJSON)
}

func TestStorageObserver_Update(t *testing.T) {
	wsconn, ws, cancel := setupWSConn(t)
	defer cancel()

	storageObserver := StorageObserver{
		id:     0,
		wsconn: wsconn,
	}

	data := []state.KeyValue{{
		Key:   []byte("key"),
		Value: []byte("value"),
	}}
	change := &state.SubscriptionResult{
		Changes: data,
	}

	expected := ChangeResult{
		Block:   change.Hash.String(),
		Changes: make([]Change, len(change.Changes)),
	}
	for i, v := range change.Changes {
		expected.Changes[i] = Change{common.BytesToHex(v.Key), common.BytesToHex(v.Value)}
	}

	expectedResponse := newSubcriptionBaseResponseJSON()
	expectedResponse.Method = stateStorageMethod
	expectedResponse.Params.Result = expected

	storageObserver.Update(change)
	time.Sleep(time.Millisecond * 100)

	_, msg, err := ws.ReadMessage()
	require.NoError(t, err)

	expectedResponseBytes, err := json.Marshal(expectedResponse)
	require.NoError(t, err)

	require.Equal(t, string(expectedResponseBytes)+"\n", string(msg))
}

func TestBlockListener_Listen(t *testing.T) {
	wsconn, ws, cancel := setupWSConn(t)
	defer cancel()

	BlockAPI := new(mocks.BlockAPI)
	BlockAPI.On("FreeImportedBlockNotifierChannel", mock.AnythingOfType("chan *types.Block"))

	wsconn.BlockAPI = BlockAPI

	notifyChan := make(chan *types.Block)
	bl := BlockListener{
		Channel:       notifyChan,
		wsconn:        wsconn,
		cancel:        make(chan struct{}),
		done:          make(chan struct{}),
		cancelTimeout: time.Second * 5,
	}

	//block := types.NewEmptyBlock()
	block := types.NewBlock(*types.NewEmptyHeader(), *new(types.Body))
	block.Header.Number = big.NewInt(1)

	go bl.Listen()
	defer func() {
		require.NoError(t, bl.Stop())
		time.Sleep(time.Millisecond * 10)
		BlockAPI.AssertCalled(t, "FreeImportedBlockNotifierChannel", mock.AnythingOfType("chan *types.Block"))
	}()

	notifyChan <- &block
	time.Sleep(time.Second * 2)

	_, msg, err := ws.ReadMessage()
	require.NoError(t, err)

	head, err := modules.HeaderToJSON(block.Header)
	require.NoError(t, err)

	expectedResposnse := newSubcriptionBaseResponseJSON()
	expectedResposnse.Method = chainNewHeadMethod
	expectedResposnse.Params.Result = head

	expectedResponseBytes, err := json.Marshal(expectedResposnse)
	require.NoError(t, err)

	require.Equal(t, string(expectedResponseBytes)+"\n", string(msg))
}

func TestBlockFinalizedListener_Listen(t *testing.T) {
	wsconn, ws, cancel := setupWSConn(t)
	defer cancel()

	BlockAPI := new(mocks.BlockAPI)
	BlockAPI.On("FreeFinalisedNotifierChannel", mock.AnythingOfType("chan *types.FinalisationInfo"))

	wsconn.BlockAPI = BlockAPI

	notifyChan := make(chan *types.FinalisationInfo)
	bfl := BlockFinalizedListener{
		channel:       notifyChan,
		wsconn:        wsconn,
		cancel:        make(chan struct{}),
		done:          make(chan struct{}),
		cancelTimeout: time.Second * 5,
	}

	header := types.NewEmptyHeader()

	bfl.Listen()
	defer func() {
		require.NoError(t, bfl.Stop())
		time.Sleep(time.Millisecond * 10)
		BlockAPI.AssertCalled(t, "FreeFinalisedNotifierChannel", mock.AnythingOfType("chan *types.FinalisationInfo"))
	}()

	notifyChan <- &types.FinalisationInfo{
		Header: *header,
	}
	time.Sleep(time.Second * 2)

	_, msg, err := ws.ReadMessage()
	require.NoError(t, err)

	head, err := modules.HeaderToJSON(*header)
	if err != nil {
		logger.Errorf("failed to convert header to JSON: %s", err)
	}
	expectedResponse := newSubcriptionBaseResponseJSON()
	expectedResponse.Method = chainFinalizedHeadMethod
	expectedResponse.Params.Result = head

	expectedResponseBytes, err := json.Marshal(expectedResponse)
	require.NoError(t, err)

	require.Equal(t, string(expectedResponseBytes)+"\n", string(msg))
}
func TestExtrinsicSubmitListener_Listen(t *testing.T) {
	wsconn, ws, cancel := setupWSConn(t)
	defer cancel()

	notifyImportedChan := make(chan *types.Block, 100)
	notifyFinalizedChan := make(chan *types.FinalisationInfo, 100)
	txStatusChan := make(chan transaction.Status)

	BlockAPI := new(mocks.BlockAPI)
	BlockAPI.On("FreeImportedBlockNotifierChannel", mock.AnythingOfType("chan *types.Block"))
	BlockAPI.On("FreeFinalisedNotifierChannel", mock.AnythingOfType("chan *types.FinalisationInfo"))

	wsconn.BlockAPI = BlockAPI

	TxStateAPI := modules.NewMockTransactionStateAPI()
	wsconn.TxStateAPI = TxStateAPI

	esl := ExtrinsicSubmitListener{
		importedChan:  notifyImportedChan,
		finalisedChan: notifyFinalizedChan,
		txStatusChan:  txStatusChan,
		wsconn:        wsconn,
		extrinsic:     types.Extrinsic{1, 2, 3},
		cancel:        make(chan struct{}),
		done:          make(chan struct{}),
		cancelTimeout: time.Second * 5,
	}
	header := types.NewEmptyHeader()
	exts := []types.Extrinsic{{1, 2, 3}, {7, 8, 9, 0}, {0xa, 0xb}}

	body := types.NewBody(exts)

	block := &types.Block{
		Header: *header,
		Body:   *body,
	}

	esl.Listen()
	defer func() {
		require.NoError(t, esl.Stop())
		time.Sleep(time.Millisecond * 10)

		BlockAPI.AssertCalled(t, "FreeImportedBlockNotifierChannel", mock.AnythingOfType("chan *types.Block"))
		BlockAPI.AssertCalled(t, "FreeFinalisedNotifierChannel", mock.AnythingOfType("chan *types.FinalisationInfo"))
	}()

	notifyImportedChan <- block
	time.Sleep(time.Second * 2)

	_, msg, err := ws.ReadMessage()
	require.NoError(t, err)
	resImported := map[string]interface{}{"inBlock": block.Header.Hash().String()}
	expectedImportedBytes, err := json.Marshal(
		newSubscriptionResponse(authorExtrinsicUpdatesMethod, esl.subID, resImported))
	require.NoError(t, err)
	require.Equal(t, string(expectedImportedBytes)+"\n", string(msg))

	notifyFinalizedChan <- &types.FinalisationInfo{
		Header: *header,
	}
	time.Sleep(time.Second * 2)

	_, msg, err = ws.ReadMessage()
	require.NoError(t, err)
	resFinalised := map[string]interface{}{"finalised": block.Header.Hash().String()}
	expectedFinalizedBytes, err := json.Marshal(
		newSubscriptionResponse(authorExtrinsicUpdatesMethod, esl.subID, resFinalised))
	require.NoError(t, err)
	require.Equal(t, string(expectedFinalizedBytes)+"\n", string(msg))
}

func TestGrandpaJustification_Listen(t *testing.T) {
	t.Run("When justification doesnt returns error", func(t *testing.T) {
		wsconn, ws, cancel := setupWSConn(t)
		defer cancel()

		mockedJust := grandpa.Justification{
			Round: 1,
			Commit: grandpa.Commit{
				Number:     1,
				Precommits: nil,
			},
		}

		mockedJustBytes, err := scale.Marshal(mockedJust)
		require.NoError(t, err)

		blockStateMock := new(mocks.BlockAPI)
		blockStateMock.On("GetJustification", mock.AnythingOfType("common.Hash")).Return(mockedJustBytes, nil)
		blockStateMock.On("FreeFinalisedNotifierChannel", mock.AnythingOfType("chan *types.FinalisationInfo"))
		wsconn.BlockAPI = blockStateMock

		finchannel := make(chan *types.FinalisationInfo)
		sub := GrandpaJustificationListener{
			subID:         10,
			wsconn:        wsconn,
			cancel:        make(chan struct{}, 1),
			done:          make(chan struct{}, 1),
			finalisedCh:   finchannel,
			cancelTimeout: time.Second * 5,
		}

		sub.Listen()
		finchannel <- &types.FinalisationInfo{
			Header: *types.NewEmptyHeader(),
		}

		time.Sleep(time.Second * 3)

		_, msg, err := ws.ReadMessage()
		require.NoError(t, err)

		expected := `{"jsonrpc":"2.0","method":"grandpa_justifications","params":{"result":"%s","subscription":10}}` + "\n"
		expected = fmt.Sprintf(expected, common.BytesToHex(mockedJustBytes))

		require.Equal(t, string(msg), expected)
		require.NoError(t, sub.Stop())
		wsconn.Wsconn.Close()
	})
}

func setupWSConn(t *testing.T) (*WSConn, *websocket.Conn, func()) {
	t.Helper()

	wskt := new(WSConn)
	var up = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	h := func(w http.ResponseWriter, r *http.Request) {
		c, err := up.Upgrade(w, r, nil)
		if err != nil {
			log.Print("error while setup handler:", err)
			return
		}

		wskt.Wsconn = c
	}

	server := httptest.NewServer(http.HandlerFunc(h))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, r, err := websocket.DefaultDialer.Dial(wsURL, nil)
	defer r.Body.Close()

	require.NoError(t, err)

	cancel := func() {
		server.Close()
		ws.Close()
		wskt.Wsconn.Close()
	}

	return wskt, ws, cancel
}

func TestRuntimeChannelListener_Listen(t *testing.T) {
	notifyChan := make(chan runtime.Version)
	mockConnection := &mockWSConnAPI{}
	rvl := RuntimeVersionListener{
		wsconn:        mockConnection,
		subID:         0,
		runtimeUpdate: notifyChan,
		coreAPI:       modules.NewMockCoreAPI(),
	}

	expectedInitialVersion := modules.StateRuntimeVersionResponse{
		SpecName: "mock-spec",
		Apis:     modules.ConvertAPIs(nil),
	}

	expectedInitialResponse := newSubcriptionBaseResponseJSON()
	expectedInitialResponse.Method = "state_runtimeVersion"
	expectedInitialResponse.Params.Result = expectedInitialVersion

	instance := wasmer.NewTestInstance(t, runtime.NODE_RUNTIME)
	err := runtime.GetRuntimeBlob(runtime.POLKADOT_RUNTIME_FP, runtime.POLKADOT_RUNTIME_URL)
	require.NoError(t, err)
	fp, err := filepath.Abs(runtime.POLKADOT_RUNTIME_FP)
	require.NoError(t, err)
	code, err := os.ReadFile(fp)
	require.NoError(t, err)
	version, err := instance.CheckRuntimeVersion(code)
	require.NoError(t, err)

	expectedUpdatedVersion := modules.StateRuntimeVersionResponse{
		SpecName:           "polkadot",
		ImplName:           "parity-polkadot",
		AuthoringVersion:   0,
		SpecVersion:        25,
		ImplVersion:        0,
		TransactionVersion: 5,
		Apis:               modules.ConvertAPIs(version.APIItems()),
	}

	expectedUpdateResponse := newSubcriptionBaseResponseJSON()
	expectedUpdateResponse.Method = "state_runtimeVersion"
	expectedUpdateResponse.Params.Result = expectedUpdatedVersion

	go rvl.Listen()

	//check initial response
	time.Sleep(time.Millisecond * 10)
	require.Equal(t, expectedInitialResponse, mockConnection.lastMessage)

	// check response after update
	notifyChan <- version
	time.Sleep(time.Millisecond * 10)
	require.Equal(t, expectedUpdateResponse, mockConnection.lastMessage)
}
