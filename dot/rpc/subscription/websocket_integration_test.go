// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

//go:build integration

package subscription

import (
	"fmt"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/ChainSafe/gossamer/dot/rpc/modules/mocks"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/grandpa"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/transaction"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/golang/mock/gomock"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

func TestWSConn_HandleConn(t *testing.T) {
	ctrl := gomock.NewController(t)

	wsconn, c, cancel := setupWSConn(t)
	wsconn.Subscriptions = make(map[uint32]Listener)
	defer cancel()

	go wsconn.HandleConn()
	time.Sleep(time.Second * 2)

	// test storageChangeListener
	res, err := wsconn.initStorageChangeListener(1, nil)
	require.Nil(t, res)
	require.Len(t, wsconn.Subscriptions, 0)
	require.EqualError(t, err, "error StorageAPI not set")
	_, msg, err := c.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, []byte(`{"jsonrpc":"2.0",`+
		`"error":{"code":null,"message":"error StorageAPI not set"},`+
		`"id":1}`+"\n"), msg)

	wsconn.StorageAPI = modules.NewMockAnyStorageAPI(ctrl)

	res, err = wsconn.initStorageChangeListener(1, nil)
	require.Nil(t, res)
	require.Len(t, wsconn.Subscriptions, 0)
	require.ErrorIs(t, err, errUnexpectedType)
	require.EqualError(t, err, "unexpected type: <nil>, expected type []interface{}")

	res, err = wsconn.initStorageChangeListener(2, []interface{}{})
	require.NotNil(t, res)
	require.NoError(t, err)
	require.Len(t, wsconn.Subscriptions, 1)
	_, msg, err = c.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, []byte(`{"jsonrpc":"2.0","result":1,"id":2}`+"\n"), msg)

	var testFilter0 = []interface{}{"0x26aa"}
	res, err = wsconn.initStorageChangeListener(3, testFilter0)
	require.NotNil(t, res)
	require.NoError(t, err)
	require.Len(t, wsconn.Subscriptions, 2)
	_, msg, err = c.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, []byte(`{"jsonrpc":"2.0","result":2,"id":3}`+"\n"), msg)

	var testFilter1 = []interface{}{[]interface{}{"0x26aa", "0x26a1"}}
	res, err = wsconn.initStorageChangeListener(4, testFilter1)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Len(t, wsconn.Subscriptions, 3)
	_, msg, err = c.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, []byte(`{"jsonrpc":"2.0","result":3,"id":4}`+"\n"), msg)

	var testFilterWrongType = []interface{}{[]int{123}}
	res, err = wsconn.initStorageChangeListener(5, testFilterWrongType)
	require.ErrorIs(t, err, errUnexpectedType)
	require.EqualError(t, err, "unexpected type: []int, expected type string, []string, []interface{}")
	require.Nil(t, res)
	// keep subscriptions len == 3, no additions was made
	require.Len(t, wsconn.Subscriptions, 3)

	res, err = wsconn.initStorageChangeListener(6, []interface{}{1})
	require.ErrorIs(t, err, errUnexpectedType)
	require.EqualError(t, err, "unexpected type: int, expected type string, []string, []interface{}")
	require.Nil(t, res)
	require.Len(t, wsconn.Subscriptions, 3)

	c.WriteMessage(websocket.TextMessage, []byte(`{
    "jsonrpc": "2.0",
    "method": "state_subscribeStorage",
    "params": ["`+
		`0x26aa394eea5630e07c48ae0c9558c`+
		`ef7b99d880ec681799c0cf30e888637`+
		`1da9de1e86a9a8c739864cf3cc5ec2b`+
		`ea59fd43593c715fdd31c61141abd04`+
		`a99fd6822c8558854ccde39a5684e7a`+
		`56da27d"],
    "id": 7}`))
	_, msg, err = c.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, []byte(`{"jsonrpc":"2.0","result":4,"id":7}`+"\n"), msg)

	// test state_unsubscribeStorage
	c.WriteMessage(websocket.TextMessage, []byte(`{
    "jsonrpc": "2.0",
    "method": "state_unsubscribeStorage",
    "params": "foo",
    "id": 7}`))
	_, msg, err = c.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, []byte(`{"jsonrpc":"2.0","error":{"code":-32600,"message":"Invalid request"},"id":7}`+"\n"), msg)

	c.WriteMessage(websocket.TextMessage, []byte(`{
    "jsonrpc": "2.0",
    "method": "state_unsubscribeStorage",
    "params": [],
    "id": 7}`))
	_, msg, err = c.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, []byte(`{"jsonrpc":"2.0","error":{"code":-32600,"message":"Invalid request"},"id":7}`+"\n"), msg)

	c.WriteMessage(websocket.TextMessage, []byte(`{
    "jsonrpc": "2.0",
    "method": "state_unsubscribeStorage",
    "params": ["6"],
    "id": 7}`))
	_, msg, err = c.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, []byte(`{"jsonrpc":"2.0","result":false,"id":7}`+"\n"), msg)

	c.WriteMessage(websocket.TextMessage, []byte(`{
    "jsonrpc": "2.0",
    "method": "state_unsubscribeStorage",
    "params": ["4"],
    "id": 7}`))
	_, msg, err = c.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, []byte(`{"jsonrpc":"2.0","result":true,"id":7}`+"\n"), msg)

	c.WriteMessage(websocket.TextMessage, []byte(`{
    "jsonrpc": "2.0",
    "method": "state_unsubscribeStorage",
    "params": [6],
    "id": 7}`))
	_, msg, err = c.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, []byte(`{"jsonrpc":"2.0","result":false,"id":7}`+"\n"), msg)

	c.WriteMessage(websocket.TextMessage, []byte(`{
    "jsonrpc": "2.0",
    "method": "state_unsubscribeStorage",
    "params": [4],
    "id": 7}`))
	_, msg, err = c.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, []byte(`{"jsonrpc":"2.0","result":true,"id":7}`+"\n"), msg)

	// test initBlockListener
	res, err = wsconn.initBlockListener(1, nil)
	require.EqualError(t, err, "error BlockAPI not set")
	require.Nil(t, res)
	_, msg, err = c.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, []byte(`{"jsonrpc":"2.0","error":{"code":null,"message":"error BlockAPI not set"},"id":1}`+"\n"), msg)

	wsconn.BlockAPI = modules.NewMockAnyBlockAPI(ctrl)

	res, err = wsconn.initBlockListener(1, nil)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Len(t, wsconn.Subscriptions, 5)
	_, msg, err = c.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, []byte(`{"jsonrpc":"2.0","result":5,"id":1}`+"\n"), msg)

	c.WriteMessage(websocket.TextMessage, []byte(`{
		"jsonrpc": "2.0",
		"method": "chain_subscribeNewHeads",
		"params": [],
		"id": 8
	}`))
	_, msg, err = c.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, []byte(`{"jsonrpc":"2.0","result":6,"id":8}`+"\n"), msg)

	// test initBlockFinalizedListener
	wsconn.BlockAPI = nil

	res, err = wsconn.initBlockFinalizedListener(1, nil)
	require.EqualError(t, err, "error BlockAPI not set")
	require.Nil(t, res)
	_, msg, err = c.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, []byte(`{"jsonrpc":"2.0","error":{"code":null,"message":"error BlockAPI not set"},"id":1}`+"\n"), msg)

	wsconn.BlockAPI = modules.NewMockAnyBlockAPI(ctrl)

	res, err = wsconn.initBlockFinalizedListener(1, nil)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Len(t, wsconn.Subscriptions, 7)
	_, msg, err = c.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, []byte(`{"jsonrpc":"2.0","result":7,"id":1}`+"\n"), msg)

	// test initExtrinsicWatch
	wsconn.CoreAPI = modules.NewMockAnyAPI(ctrl)
	wsconn.BlockAPI = nil

	transactionStateAPI := NewMockTransactionStateAPI(ctrl)
	transactionStateAPI.EXPECT().GetStatusNotifierChannel(gomock.Any()).Return(make(chan transaction.Status)).Times(2)
	wsconn.TxStateAPI = transactionStateAPI

	listner, err := wsconn.initExtrinsicWatch(0, []string{"NotHex"})
	require.EqualError(t, err, "could not byteify non 0x prefixed string: NotHex")
	require.Nil(t, listner)

	listner, err = wsconn.initExtrinsicWatch(0, []interface{}{"0x26aa"})
	require.EqualError(t, err, "error BlockAPI not set")
	require.Nil(t, listner)

	wsconn.BlockAPI = modules.NewMockAnyBlockAPI(ctrl)
	listner, err = wsconn.initExtrinsicWatch(0, []interface{}{"0x26aa"})
	require.NoError(t, err)
	require.NotNil(t, listner)
	require.Len(t, wsconn.Subscriptions, 8)

	_, msg, err = c.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, `{"jsonrpc":"2.0","result":8,"id":0}`+"\n", string(msg))

	// test initExtrinsicWatch with invalid transaction
	invalidTransaction := runtime.NewInvalidTransaction()
	err = invalidTransaction.Set(runtime.Future{})
	require.NoError(t, err)

	require.NoError(t, err)
	coreAPI := mocks.NewMockCoreAPI(ctrl)
	coreAPI.EXPECT().HandleSubmittedExtrinsic(gomock.Any()).
		Return(invalidTransaction)
	wsconn.CoreAPI = coreAPI
	listner, err = wsconn.initExtrinsicWatch(0,
		[]interface{}{"0xa9018400d43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d019e91c8d44bf01ffe36d54f9e43dade2b2fc653270a0e002daed1581435c2e1755bc4349f1434876089d99c9dac4d4128e511c2a3e0788a2a74dd686519cb7c83000000000104ab"}) //nolint:lll
	require.Error(t, err)
	require.Nil(t, listner)

	_, msg, err = c.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, `{"jsonrpc":"2.0","method":"author_extrinsicUpdate",`+
		`"params":{"result":"invalid","subscription":9}}`+"\n", string(msg))

	mockedJust := grandpa.Justification{
		Round: 1,
		Commit: grandpa.Commit{
			Number:     1,
			Precommits: nil,
		},
	}

	mockedJustBytes, err := scale.Marshal(mockedJust)
	require.NoError(t, err)

	wsconn.CoreAPI = modules.NewMockAnyAPI(ctrl)
	BlockAPI := mocks.NewMockBlockAPI(ctrl)

	fCh := make(chan *types.FinalisationInfo, 5)
	BlockAPI.EXPECT().GetFinalisedNotifierChannel().Return(fCh)

	BlockAPI.EXPECT().GetJustification(gomock.Any()).Return(mockedJustBytes, nil)
	BlockAPI.EXPECT().FreeFinalisedNotifierChannel(gomock.Any())

	wsconn.BlockAPI = BlockAPI
	listener, err := wsconn.initGrandpaJustificationListener(0, nil)
	require.NoError(t, err)
	require.NotNil(t, listener)

	_, msg, err = c.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, `{"jsonrpc":"2.0","result":10,"id":0}`+"\n", string(msg))

	listener.Listen()
	header := &types.Header{
		Number: 1,
	}

	fCh <- &types.FinalisationInfo{
		Header: *header,
	}

	time.Sleep(time.Second * 2)

	expected := `{"jsonrpc":"2.0","method":"grandpa_justifications","params":{"result":"%s","subscription":10}}` + "\n"
	expected = fmt.Sprintf(expected, common.BytesToHex(mockedJustBytes))
	_, msg, err = c.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, []byte(expected), msg)

	err = listener.Stop()
	require.NoError(t, err)
}

func TestSubscribeAllHeads(t *testing.T) {
	ctrl := gomock.NewController(t)

	wsconn, c, cancel := setupWSConn(t)
	wsconn.Subscriptions = make(map[uint32]Listener)
	defer cancel()

	go wsconn.HandleConn()
	time.Sleep(time.Second * 2)

	_, err := wsconn.initAllBlocksListerner(1, nil)
	require.EqualError(t, err, "error BlockAPI not set")
	_, msg, err := c.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, []byte(`{"jsonrpc":"2.0","error":{"code":null,"message":"error BlockAPI not set"},"id":1}`+"\n"), msg)

	mockBlockAPI := mocks.NewMockBlockAPI(ctrl)

	wsconn.BlockAPI = mockBlockAPI

	iCh := make(chan *types.Block)
	mockBlockAPI.EXPECT().GetImportedBlockNotifierChannel().Return(iCh)

	fCh := make(chan *types.FinalisationInfo)
	mockBlockAPI.EXPECT().GetFinalisedNotifierChannel().Return(fCh)

	l, err := wsconn.initAllBlocksListerner(1, nil)
	require.NoError(t, err)
	require.NotNil(t, l)
	require.IsType(t, &AllBlocksListener{}, l)
	require.Len(t, wsconn.Subscriptions, 1)

	_, msg, err = c.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, []byte(`{"jsonrpc":"2.0","result":1,"id":1}`+"\n"), msg)

	l.Listen()
	time.Sleep(time.Millisecond * 500)

	expected := fmt.Sprintf(
		`{"jsonrpc":"2.0","method":"chain_allHead",`+
			`"params":{"result":{"parentHash":"%s","number":"0x00",`+
			`"stateRoot":"%s","extrinsicsRoot":"%s",`+
			`"digest":{"logs":["0x064241424504ff"]}},"subscription":1}}`,
		common.Hash{},
		common.Hash{},
		common.Hash{},
	)

	digest := types.NewDigest()
	err = digest.Add(*types.NewBABEPreRuntimeDigest([]byte{0xff}))
	require.NoError(t, err)
	fCh <- &types.FinalisationInfo{
		Header: types.Header{
			Number: 0,
			Digest: digest,
		},
	}

	time.Sleep(time.Millisecond * 500)
	_, msg, err = c.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, expected+"\n", string(msg))

	digest = types.NewDigest()
	err = digest.Add(*types.NewBABEPreRuntimeDigest([]byte{0xff}))
	require.NoError(t, err)

	iCh <- &types.Block{
		Header: types.Header{
			Number: 0,
			Digest: digest,
		},
	}
	time.Sleep(time.Millisecond * 500)
	_, msg, err = c.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, []byte(expected+"\n"), msg)

	mockBlockAPI.EXPECT().FreeImportedBlockNotifierChannel(gomock.Any())
	mockBlockAPI.EXPECT().FreeFinalisedNotifierChannel(gomock.Any())

	require.NoError(t, l.Stop())
}

func TestWSConn_CheckWebsocketInvalidData(t *testing.T) {
	wsconn, c, cancel := setupWSConn(t)
	wsconn.Subscriptions = make(map[uint32]Listener)
	defer cancel()

	go wsconn.HandleConn()

	tests := []struct {
		sentMessage []byte
		expected    []byte
	}{
		{
			sentMessage: []byte(`{
			"jsonrpc": "2.0",
			"method": "",
			"id": 0,
			"params": []
			}`),
			expected: []byte(`{"jsonrpc":"2.0","error":{"code":-32600,"message":"Invalid request"},"id":0}` + "\n"),
		},
		{
			sentMessage: []byte(`{
			"jsonrpc": "2.0",
			"params": []
			}`),
			expected: []byte(`{"jsonrpc":"2.0","error":{"code":-32600,"message":"Invalid request"},"id":0}` + "\n"),
		},
		{
			sentMessage: []byte(`{
			"jsonrpc": "2.0",
			"id": "abcdef"
			"method": "some_method_name"
			"params": []
			}`),
			expected: []byte(`{"jsonrpc":"2.0","error":{"code":-32600,"message":"Invalid request"},"id":0}` + "\n"),
		},
	}

	for _, tt := range tests {
		c.WriteMessage(websocket.TextMessage, tt.sentMessage)

		_, msg, err := c.ReadMessage()
		require.NoError(t, err)
		require.Equal(t, tt.expected, msg)
	}
}
