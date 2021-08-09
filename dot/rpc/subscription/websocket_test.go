package subscription

import (
	"fmt"
	"math/big"
	"testing"
	"time"

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

	fmt.Println("ws defined")

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
