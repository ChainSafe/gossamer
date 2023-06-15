//go:build integration

// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package subscription

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
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
	gomock "github.com/golang/mock/gomock"
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
	ctrl := gomock.NewController(t)

	wsconn, ws, cancel := setupWSConn(t)
	defer cancel()

	BlockAPI := mocks.NewMockBlockAPI(ctrl)
	BlockAPI.EXPECT().FreeImportedBlockNotifierChannel(gomock.Any())

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
	block.Header.Number = 1

	go bl.Listen()
	defer func() {
		require.NoError(t, bl.Stop())
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
	ctrl := gomock.NewController(t)

	wsconn, ws, cancel := setupWSConn(t)
	defer cancel()

	BlockAPI := mocks.NewMockBlockAPI(ctrl)
	BlockAPI.EXPECT().FreeFinalisedNotifierChannel(gomock.Any())

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
	ctrl := gomock.NewController(t)

	wsconn, ws, cancel := setupWSConn(t)
	defer cancel()

	notifyImportedChan := make(chan *types.Block, 100)
	notifyFinalizedChan := make(chan *types.FinalisationInfo, 100)
	txStatusChan := make(chan transaction.Status)

	BlockAPI := mocks.NewMockBlockAPI(ctrl)
	BlockAPI.EXPECT().FreeImportedBlockNotifierChannel(gomock.Any())
	BlockAPI.EXPECT().FreeFinalisedNotifierChannel(gomock.Any())

	wsconn.BlockAPI = BlockAPI

	TxStateAPI := NewMockTransactionStateAPI(ctrl)
	TxStateAPI.EXPECT().FreeStatusNotifierChannel(gomock.Any())
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
	t.Run("When_justification_doesnt_returns_error", func(t *testing.T) {
		ctrl := gomock.NewController(t)

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

		blockStateMock := mocks.NewMockBlockAPI(ctrl)
		blockStateMock.EXPECT().GetJustification(gomock.Any()).Return(mockedJustBytes, nil)
		blockStateMock.EXPECT().FreeFinalisedNotifierChannel(gomock.Any())
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

func TestRuntimeChannelListener_Listen(t *testing.T) {
	ctrl := gomock.NewController(t)

	notifyChan := make(chan runtime.Version)
	mockConnection := &mockWSConnAPI{}
	rvl := RuntimeVersionListener{
		wsconn:        mockConnection,
		subID:         0,
		runtimeUpdate: notifyChan,
		coreAPI:       modules.NewMockAnyAPI(ctrl),
	}

	expectedInitialVersion := modules.StateRuntimeVersionResponse{
		SpecName: "mock-spec",
		Apis:     []interface{}{},
	}

	expectedInitialResponse := newSubcriptionBaseResponseJSON()
	expectedInitialResponse.Method = "state_runtimeVersion"
	expectedInitialResponse.Params.Result = expectedInitialVersion

	polkadotRuntimeFilepath, err := runtime.GetRuntime(context.Background(), runtime.POLKADOT_RUNTIME_v0929)
	require.NoError(t, err)
	code, err := os.ReadFile(polkadotRuntimeFilepath)
	require.NoError(t, err)
	version, err := wasmer.GetRuntimeVersion(code)
	require.NoError(t, err)

	expectedUpdatedVersion := modules.StateRuntimeVersionResponse{
		SpecName:           "polkadot",
		ImplName:           "parity-polkadot",
		AuthoringVersion:   0,
		SpecVersion:        9290,
		ImplVersion:        0,
		TransactionVersion: 14,
		Apis: []interface{}{
			[]interface{}{"0xdf6acb689907609b", uint32(0x4)},
			[]interface{}{"0x37e397fc7c91f5e4", uint32(0x1)},
			[]interface{}{"0x40fe3ad401f8959a", uint32(0x6)},
			[]interface{}{"0x17a6bc0d0062aeb3", uint32(0x1)},
			[]interface{}{"0xd2bc9897eed08f15", uint32(0x3)},
			[]interface{}{"0xf78b278be53f454c", uint32(0x2)},
			[]interface{}{"0xaf2c0297a23e6d3d", uint32(0x2)},
			[]interface{}{"0x49eaaf1b548a0cb0", uint32(0x1)},
			[]interface{}{"0x91d5df18b0d2cf58", uint32(0x1)},
			[]interface{}{"0xed99c5acb25eedf5", uint32(0x3)},
			[]interface{}{"0xcbca25e39f142387", uint32(0x2)},
			[]interface{}{"0x687ad44ad37f03c2", uint32(0x1)},
			[]interface{}{"0xab3c0572291feb8b", uint32(0x1)},
			[]interface{}{"0xbc9d89904f5b923f", uint32(0x1)},
			[]interface{}{"0x37c8bb1350a9a2a8", uint32(0x1)},
			[]interface{}{"0xf3ff14d5ab527059", uint32(0x1)},
		},
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
