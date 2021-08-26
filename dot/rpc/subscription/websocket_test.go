package subscription

import (
	"errors"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/rpc/modules/mocks"
	modulesmocks "github.com/ChainSafe/gossamer/dot/rpc/modules/mocks"

	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/grandpa"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestWSConn_HandleComm(t *testing.T) {
	wsconn, c, cancel := setupWSConn(t)
	wsconn.Subscriptions = make(map[uint32]Listener)
	defer cancel()

	go wsconn.HandleComm()
	time.Sleep(time.Second * 2)

	// test storageChangeListener
	res, err := wsconn.initStorageChangeListener(1, nil)
	require.Nil(t, res)
	require.Len(t, wsconn.Subscriptions, 0)
	require.EqualError(t, err, "error StorageAPI not set")
	_, msg, err := c.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, []byte(`{"jsonrpc":"2.0","error":{"code":null,"message":"error StorageAPI not set"},"id":1}`+"\n"), msg)

	wsconn.StorageAPI = modules.NewMockStorageAPI()

	res, err = wsconn.initStorageChangeListener(1, nil)
	require.Nil(t, res)
	require.Len(t, wsconn.Subscriptions, 0)
	require.EqualError(t, err, "unknown parameter type")

	res, err = wsconn.initStorageChangeListener(2, []interface{}{})
	require.NotNil(t, res)
	require.NoError(t, err)
	require.Len(t, wsconn.Subscriptions, 1)
	_, msg, err = c.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, []byte(`{"jsonrpc":"2.0","result":1,"id":2}`+"\n"), msg)

	res, err = wsconn.initStorageChangeListener(3, []interface{}{"0x26aa"})
	require.NotNil(t, res)
	require.NoError(t, err)
	require.Len(t, wsconn.Subscriptions, 2)
	_, msg, err = c.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, []byte(`{"jsonrpc":"2.0","result":2,"id":3}`+"\n"), msg)

	var testFilters = []interface{}{}
	var testFilter1 = []interface{}{"0x26aa", "0x26a1"}
	res, err = wsconn.initStorageChangeListener(4, append(testFilters, testFilter1))
	require.NotNil(t, res)
	require.NoError(t, err)
	require.Len(t, wsconn.Subscriptions, 3)
	_, msg, err = c.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, []byte(`{"jsonrpc":"2.0","result":3,"id":4}`+"\n"), msg)

	var testFilterWrongType = []interface{}{"0x26aa", 1}
	res, err = wsconn.initStorageChangeListener(5, append(testFilters, testFilterWrongType))
	require.EqualError(t, err, "unknown parameter type")
	require.Nil(t, res)
	// keep subscriptions len == 3, no additions was made
	require.Len(t, wsconn.Subscriptions, 3)

	res, err = wsconn.initStorageChangeListener(6, []interface{}{1})
	require.EqualError(t, err, "unknown parameter type")
	require.Nil(t, res)
	require.Len(t, wsconn.Subscriptions, 3)

	c.WriteMessage(websocket.TextMessage, []byte(`{
    "jsonrpc": "2.0",
    "method": "state_subscribeStorage",
    "params": ["0x26aa394eea5630e07c48ae0c9558cef7b99d880ec681799c0cf30e8886371da9de1e86a9a8c739864cf3cc5ec2bea59fd43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d"],
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

	wsconn.BlockAPI = modules.NewMockBlockAPI()

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

	wsconn.BlockAPI = modules.NewMockBlockAPI()

	res, err = wsconn.initBlockFinalizedListener(1, nil)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Len(t, wsconn.Subscriptions, 7)
	_, msg, err = c.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, []byte(`{"jsonrpc":"2.0","result":7,"id":1}`+"\n"), msg)

	// test initExtrinsicWatch
	wsconn.CoreAPI = modules.NewMockCoreAPI()
	wsconn.BlockAPI = nil
	res, err = wsconn.initExtrinsicWatch(0, []interface{}{"NotHex"})
	require.EqualError(t, err, "could not byteify non 0x prefixed string")
	require.Nil(t, res)

	res, err = wsconn.initExtrinsicWatch(0, []interface{}{"0x26aa"})
	require.EqualError(t, err, "error BlockAPI not set")
	require.Nil(t, res)

	wsconn.BlockAPI = modules.NewMockBlockAPI()
	res, err = wsconn.initExtrinsicWatch(0, []interface{}{"0x26aa"})
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Len(t, wsconn.Subscriptions, 8)

	_, msg, err = c.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, `{"jsonrpc":"2.0","result":8,"id":0}`+"\n", string(msg))

	_, msg, err = c.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, `{"jsonrpc":"2.0","method":"author_extrinsicUpdate","params":{"result":"ready","subscription":8}}`+"\n", string(msg))

	var fCh chan<- *types.FinalisationInfo
	mockedJust := grandpa.Justification{
		Round: 1,
		Commit: &grandpa.Commit{
			Hash:       common.Hash{},
			Number:     1,
			Precommits: nil,
		},
	}

	mockedJustBytes, err := mockedJust.Encode()
	require.NoError(t, err)

	BlockAPI := new(modulesmocks.BlockAPI)
	BlockAPI.On("RegisterFinalizedChannel", mock.AnythingOfType("chan<- *types.FinalisationInfo")).
		Run(func(args mock.Arguments) {
			ch := args.Get(0).(chan<- *types.FinalisationInfo)
			fCh = ch
		}).
		Return(uint8(4), nil)

	BlockAPI.On("GetJustification", mock.AnythingOfType("common.Hash")).Return(mockedJustBytes, nil)
	BlockAPI.On("UnregisterFinalisedChannel", mock.AnythingOfType("uint8"))

	wsconn.BlockAPI = BlockAPI
	listener, err := wsconn.initGrandpaJustificationListener(0, nil)
	require.NoError(t, err)
	require.NotNil(t, listener)

	_, msg, err = c.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, `{"jsonrpc":"2.0","result":9,"id":0}`+"\n", string(msg))

	listener.Listen()
	header := &types.Header{
		ParentHash: common.Hash{},
		Number:     big.NewInt(1),
	}

	fCh <- &types.FinalisationInfo{
		Header: header,
	}

	time.Sleep(time.Second * 2)

	expected := `{"jsonrpc":"2.0","method":"grandpa_justifications","params":{"result":"%s","subscription":9}}` + "\n"
	expected = fmt.Sprintf(expected, common.BytesToHex(mockedJustBytes))
	_, msg, err = c.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, []byte(expected), msg)

	err = listener.Stop()
	require.NoError(t, err)
}

func TestSubscribeAllHeads(t *testing.T) {
	wsconn, c, cancel := setupWSConn(t)
	wsconn.Subscriptions = make(map[uint32]Listener)
	defer cancel()

	go wsconn.HandleComm()
	time.Sleep(time.Second * 2)

	_, err := wsconn.initAllBlocksListerner(1, nil)
	require.EqualError(t, err, "error BlockAPI not set")
	_, msg, err := c.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, []byte(`{"jsonrpc":"2.0","error":{"code":null,"message":"error BlockAPI not set"},"id":1}`+"\n"), msg)

	mockBlockAPI := new(mocks.BlockAPI)
	mockBlockAPI.On("RegisterImportedChannel", mock.AnythingOfType("chan<- *types.Block")).
		Return(uint8(0), errors.New("some mocked error")).Once()

	wsconn.BlockAPI = mockBlockAPI
	_, err = wsconn.initAllBlocksListerner(1, nil)
	require.Error(t, err, "could not register imported channel")

	_, msg, err = c.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, []byte(`{"jsonrpc":"2.0","error":{"code":null,"message":"could not register imported channel"},"id":1}`+"\n"), msg)

	mockBlockAPI.On("RegisterImportedChannel", mock.AnythingOfType("chan<- *types.Block")).
		Return(uint8(10), nil).Once()
	mockBlockAPI.On("RegisterFinalizedChannel", mock.AnythingOfType("chan<- *types.FinalisationInfo")).
		Return(uint8(0), errors.New("failed")).Once()

	_, err = wsconn.initAllBlocksListerner(1, nil)
	require.Error(t, err, "could not register finalised channel")
	c.ReadMessage()

	importedChanID := uint8(10)
	finalizedChanID := uint8(11)

	var fCh chan<- *types.FinalisationInfo
	var iCh chan<- *types.Block

	mockBlockAPI.On("RegisterImportedChannel", mock.AnythingOfType("chan<- *types.Block")).
		Run(func(args mock.Arguments) {
			ch := args.Get(0).(chan<- *types.Block)
			iCh = ch
		}).Return(importedChanID, nil).Once()

	mockBlockAPI.On("RegisterFinalizedChannel", mock.AnythingOfType("chan<- *types.FinalisationInfo")).
		Run(func(args mock.Arguments) {
			ch := args.Get(0).(chan<- *types.FinalisationInfo)
			fCh = ch
		}).
		Return(finalizedChanID, nil).Once()

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
		`{"jsonrpc":"2.0","method":"chain_allHead","params":{"result":{"parentHash":"%s","number":"0x00","stateRoot":"%s","extrinsicsRoot":"%s","digest":{"logs":["0x064241424504ff"]}},"subscription":1}}`,
		common.EmptyHash,
		common.EmptyHash,
		common.EmptyHash,
	)

	fCh <- &types.FinalisationInfo{
		Header: &types.Header{
			ParentHash:     common.EmptyHash,
			Number:         big.NewInt(0),
			StateRoot:      common.EmptyHash,
			ExtrinsicsRoot: common.EmptyHash,
			Digest:         types.NewDigest(types.NewBABEPreRuntimeDigest([]byte{0xff})),
		},
	}

	time.Sleep(time.Millisecond * 500)
	_, msg, err = c.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, expected+"\n", string(msg))

	iCh <- &types.Block{
		Header: &types.Header{
			ParentHash:     common.EmptyHash,
			Number:         big.NewInt(0),
			StateRoot:      common.EmptyHash,
			ExtrinsicsRoot: common.EmptyHash,
			Digest:         types.NewDigest(types.NewBABEPreRuntimeDigest([]byte{0xff})),
		},
	}
	time.Sleep(time.Millisecond * 500)
	_, msg, err = c.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, []byte(expected+"\n"), msg)

	mockBlockAPI.On("UnregisterImportedChannel", importedChanID)
	mockBlockAPI.On("UnregisterFinalisedChannel", finalizedChanID)

	require.NoError(t, l.Stop())
	mockBlockAPI.AssertCalled(t, "UnregisterImportedChannel", importedChanID)
	mockBlockAPI.AssertCalled(t, "UnregisterFinalisedChannel", finalizedChanID)
}
