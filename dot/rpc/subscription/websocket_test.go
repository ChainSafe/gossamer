package subscription

import (
	"log"
	"math/big"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}
var wsconn = &WSConn{
	Subscriptions:    make(map[uint]Listener),
	BlockSubChannels: make(map[uint]byte),
}

func handler(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()

	wsconn.Wsconn = c
	wsconn.HandleComm()
}

func TestMain(m *testing.M) {
	http.HandleFunc("/", handler)

	go func() {
		err := http.ListenAndServe("localhost:8546", nil)
		if err != nil {
			log.Fatal("error", err)
		}
	}()
	time.Sleep(time.Millisecond * 100)
	// Start all tests
	os.Exit(m.Run())
}

func TestWSConn_HandleComm(t *testing.T) {
	c, _, err := websocket.DefaultDialer.Dial("ws://localhost:8546", nil) //nolint
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	// test storageChangeListener
	res, err := wsconn.initStorageChangeListener(1, nil)
	require.EqualError(t, err, "error StorageAPI not set")
	require.Equal(t, uint(0), res)
	_, msg, err := c.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, []byte(`{"jsonrpc":"2.0","error":{"code":null,"message":"error StorageAPI not set"},"id":1}`+"\n"), msg)

	wsconn.StorageAPI = new(MockStorageAPI)

	res, err = wsconn.initStorageChangeListener(1, nil)
	require.EqualError(t, err, "unknown parameter type")
	require.Equal(t, uint(0), res)

	res, err = wsconn.initStorageChangeListener(2, []interface{}{})
	require.NoError(t, err)
	require.Equal(t, uint(1), res)
	_, msg, err = c.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, []byte(`{"jsonrpc":"2.0","result":1,"id":2}`+"\n"), msg)

	res, err = wsconn.initStorageChangeListener(3, []interface{}{"0x26aa"})
	require.NoError(t, err)
	require.Equal(t, uint(2), res)
	_, msg, err = c.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, []byte(`{"jsonrpc":"2.0","result":2,"id":3}`+"\n"), msg)

	var testFilters = []interface{}{}
	var testFilter1 = []interface{}{"0x26aa", "0x26a1"}
	res, err = wsconn.initStorageChangeListener(4, append(testFilters, testFilter1))
	require.NoError(t, err)
	require.Equal(t, uint(3), res)
	_, msg, err = c.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, []byte(`{"jsonrpc":"2.0","result":3,"id":4}`+"\n"), msg)

	var testFilterWrongType = []interface{}{"0x26aa", 1}
	res, err = wsconn.initStorageChangeListener(5, append(testFilters, testFilterWrongType))
	require.EqualError(t, err, "unknown parameter type")
	require.Equal(t, uint(0), res)

	res, err = wsconn.initStorageChangeListener(6, []interface{}{1})
	require.EqualError(t, err, "unknown parameter type")
	require.Equal(t, uint(0), res)

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
	res, err = wsconn.initBlockListener(1)
	require.EqualError(t, err, "error BlockAPI not set")
	require.Equal(t, uint(0), res)
	_, msg, err = c.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, []byte(`{"jsonrpc":"2.0","error":{"code":null,"message":"error BlockAPI not set"},"id":1}`+"\n"), msg)

	wsconn.BlockAPI = new(MockBlockAPI)

	res, err = wsconn.initBlockListener(1)
	require.NoError(t, err)
	require.Equal(t, uint(5), res)
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

	res, err = wsconn.initBlockFinalizedListener(1)
	require.EqualError(t, err, "error BlockAPI not set")
	require.Equal(t, uint(0), res)
	_, msg, err = c.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, []byte(`{"jsonrpc":"2.0","error":{"code":null,"message":"error BlockAPI not set"},"id":1}`+"\n"), msg)

	wsconn.BlockAPI = new(MockBlockAPI)

	res, err = wsconn.initBlockFinalizedListener(1)
	require.NoError(t, err)
	require.Equal(t, uint(7), res)
	_, msg, err = c.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, []byte(`{"jsonrpc":"2.0","result":7,"id":1}`+"\n"), msg)

	// test initExtrinsicWatch
	wsconn.CoreAPI = new(MockCoreAPI)
	wsconn.BlockAPI = nil
	res, err = wsconn.initExtrinsicWatch(0, []interface{}{"NotHex"})
	require.EqualError(t, err, "could not byteify non 0x prefixed string")
	require.Equal(t, uint(0), res)

	res, err = wsconn.initExtrinsicWatch(0, []interface{}{"0x26aa"})
	require.EqualError(t, err, "error BlockAPI not set")
	require.Equal(t, uint(0), res)

	wsconn.BlockAPI = new(MockBlockAPI)
	res, err = wsconn.initExtrinsicWatch(0, []interface{}{"0x26aa"})
	require.NoError(t, err)
	require.Equal(t, uint(8), res)

}

type MockStorageAPI struct{}

func (m *MockStorageAPI) GetStorage(_ *common.Hash, key []byte) ([]byte, error) {
	return nil, nil
}
func (m *MockStorageAPI) Entries(_ *common.Hash) (map[string][]byte, error) {
	return nil, nil
}
func (m *MockStorageAPI) GetStorageByBlockHash(_ common.Hash, key []byte) ([]byte, error) {
	return nil, nil
}
func (m *MockStorageAPI) RegisterStorageObserver(observer state.Observer) {
}

func (m *MockStorageAPI) UnregisterStorageObserver(observer state.Observer) {
}
func (m *MockStorageAPI) GetStateRootFromBlock(bhash *common.Hash) (*common.Hash, error) {
	return nil, nil
}
func (m *MockStorageAPI) GetKeysWithPrefix(root *common.Hash, prefix []byte) ([][]byte, error) {
	return nil, nil
}

type MockBlockAPI struct {
}

func (m *MockBlockAPI) GetHeader(hash common.Hash) (*types.Header, error) {
	return nil, nil
}
func (m *MockBlockAPI) BestBlockHash() common.Hash {
	return common.Hash{}
}
func (m *MockBlockAPI) GetBlockByHash(hash common.Hash) (*types.Block, error) {
	return nil, nil
}
func (m *MockBlockAPI) GetBlockHash(blockNumber *big.Int) (*common.Hash, error) {
	return nil, nil
}
func (m *MockBlockAPI) GetFinalizedHash(uint64, uint64) (common.Hash, error) {
	return common.Hash{}, nil
}
func (m *MockBlockAPI) RegisterImportedChannel(ch chan<- *types.Block) (byte, error) {
	return 0, nil
}
func (m *MockBlockAPI) UnregisterImportedChannel(id byte) {
}
func (m *MockBlockAPI) RegisterFinalizedChannel(ch chan<- *types.FinalisationInfo) (byte, error) {
	return 0, nil
}
func (m *MockBlockAPI) UnregisterFinalizedChannel(id byte) {}

func (m *MockBlockAPI) GetJustification(hash common.Hash) ([]byte, error) {
	return make([]byte, 10), nil
}

func (m *MockBlockAPI) HasJustification(hash common.Hash) (bool, error) {
	return true, nil
}

func (m *MockBlockAPI) SubChain(start, end common.Hash) ([]common.Hash, error) {
	return make([]common.Hash, 0), nil
}

type MockCoreAPI struct{}

func (m *MockCoreAPI) InsertKey(kp crypto.Keypair) {}

func (m *MockCoreAPI) HasKey(pubKeyStr string, keyType string) (bool, error) {
	return false, nil
}

func (m *MockCoreAPI) GetRuntimeVersion(bhash *common.Hash) (runtime.Version, error) {
	return nil, nil
}

func (m *MockCoreAPI) IsBlockProducer() bool {
	return false
}

func (m *MockCoreAPI) HandleSubmittedExtrinsic(types.Extrinsic) error {
	return nil
}

func (m *MockCoreAPI) GetMetadata(bhash *common.Hash) ([]byte, error) {
	return nil, nil
}
