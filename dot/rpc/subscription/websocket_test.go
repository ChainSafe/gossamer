package subscription

import (
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}
var wsconn = &WSConn{
	Subscriptions: make(map[int]Listener),
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
	require.Equal(t, 0, res)
	_, msg, err := c.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, []byte(`{"jsonrpc":"2.0","error":{"code":null,"message":"error StorageAPI not set"},"id":1}`+"\n"), msg)

	wsconn.StorageAPI = new(MockStorageAPI)

	res, err = wsconn.initStorageChangeListener(1, nil)
	require.EqualError(t, err, "unknown parameter type")
	require.Equal(t, 0, res)

	res, err = wsconn.initStorageChangeListener(1, []interface{}{})
	require.NoError(t, err)
	require.Equal(t, 1, res)
	_, msg, err = c.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, []byte(`{"jsonrpc":"2.0","result":1,"id":1}`+"\n"), msg)

	res, err = wsconn.initStorageChangeListener(1, []interface{}{"0x26aa"})
	require.NoError(t, err)
	require.Equal(t, 2, res)
	_, msg, err = c.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, []byte(`{"jsonrpc":"2.0","result":2,"id":1}`+"\n"), msg)

	res, err = wsconn.initStorageChangeListener(1, []interface{}{1})
	require.EqualError(t, err, "unknown parameter type")
	require.Equal(t, 0, res)

	c.WriteMessage(websocket.TextMessage, []byte(`{
    "jsonrpc": "2.0",
    "method": "state_subscribeStorage",
    "params": ["0x26aa394eea5630e07c48ae0c9558cef7b99d880ec681799c0cf30e8886371da9de1e86a9a8c739864cf3cc5ec2bea59fd43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d"],
    "id": 1
}`))
	_, msg, err = c.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, []byte(`{"jsonrpc":"2.0","result":3,"id":1}`+"\n"), msg)
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
