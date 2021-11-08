// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package rpc

import (
	"flag"
	"log"
	"net/url"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/core"
	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/ChainSafe/gossamer/dot/system"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

var addr = flag.String("addr", "localhost:8546", "http service address")

var testCalls = []struct {
	call     []byte
	expected []byte
}{
	{[]byte(`{"jsonrpc":"2.0","method":"system_name","params":[],"id":1}`), []byte(`{"id":1,"jsonrpc":"2.0","result":"gossamer"}` + "\n")},                                                            // working request
	{[]byte(`{"jsonrpc":"2.0","method":"unknown","params":[],"id":1}`), []byte(`{"error":{"code":-32000,"data":null,"message":"rpc error method unknown not found"},"id":1,"jsonrpc":"2.0"}` + "\n")}, // unknown method
	{[]byte{}, []byte(`{"jsonrpc":"2.0","error":{"code":-32600,"message":"Invalid request"},"id":0}` + "\n")},                                                                                         // empty request
	{[]byte(`{"jsonrpc":"2.0","method":"chain_subscribeNewHeads","params":[],"id":3}`), []byte(`{"jsonrpc":"2.0","result":1,"id":3}` + "\n")},
	{[]byte(`{"jsonrpc":"2.0","method":"state_subscribeStorage","params":[],"id":4}`), []byte(`{"jsonrpc":"2.0","result":2,"id":4}` + "\n")},
	{[]byte(`{"jsonrpc":"2.0","method":"chain_subscribeFinalizedHeads","params":[],"id":5}`), []byte(`{"jsonrpc":"2.0","result":3,"id":5}` + "\n")},
	{[]byte(`{"jsonrpc":"2.0","method":"author_submitAndWatchExtrinsic","params":["0x010203"],"id":6}`), []byte("{\"jsonrpc\":\"2.0\",\"error\":{\"code\":null,\"message\":\"Failed to call the `TaggedTransactionQueue_validate_transaction` exported function.\"},\"id\":6}\n")},
	{[]byte(`{"jsonrpc":"2.0","method":"state_subscribeRuntimeVersion","params":[],"id":7}`), []byte("{\"jsonrpc\":\"2.0\",\"result\":6,\"id\":7}\n")},
}

func TestHTTPServer_ServeHTTP(t *testing.T) {
	coreAPI := core.NewTestService(t, nil)
	si := &types.SystemInfo{
		SystemName: "gossamer",
	}
	sysAPI := system.NewService(si, nil)
	bAPI := modules.NewMockBlockAPI()
	sAPI := modules.NewMockStorageAPI()

	TxStateAPI := modules.NewMockTransactionStateAPI()

	cfg := &HTTPServerConfig{
		Modules:             []string{"system", "chain"},
		RPCExternal:         false,
		RPCPort:             8545,
		WSPort:              8546,
		WS:                  true,
		WSExternal:          false,
		RPCAPI:              NewService(),
		CoreAPI:             coreAPI,
		SystemAPI:           sysAPI,
		BlockAPI:            bAPI,
		StorageAPI:          sAPI,
		TransactionQueueAPI: TxStateAPI,
	}

	s := NewHTTPServer(cfg)
	err := s.Start()
	require.Nil(t, err)

	defer s.Stop()

	time.Sleep(time.Second) // give server a second to start

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/"}
	log.Printf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	for _, item := range testCalls {
		err = c.WriteMessage(websocket.TextMessage, item.call)
		require.Nil(t, err)

		_, message, err := c.ReadMessage()
		require.Nil(t, err)
		require.Equal(t, item.expected, message)
	}
}
