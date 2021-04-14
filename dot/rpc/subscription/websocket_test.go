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

	wsconn.StorageAPI = new(MockStorageAPI)

	res, err = wsconn.initStorageChangeListener(1, nil)
	require.EqualError(t, err, "unknow parameter type")
	require.Equal(t, 0, res)

	res, err = wsconn.initStorageChangeListener(1, []interface{}{})
	require.NoError(t, err)
	require.Equal(t, 1, res)
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
