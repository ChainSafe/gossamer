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
	{
		call:     []byte(`{"jsonrpc":"2.0","method":"system_name","params":[],"id":1}`),
		expected: []byte(`{"id":1,"jsonrpc":"2.0","result":"gossamer"}` + "\n")}, // working request
	{
		call: []byte(`{"jsonrpc":"2.0","method":"unknown","params":[],"id":1}`),
		// unknown method
		expected: []byte(`{"error":{` +
			`"code":-32000,"data":null,` +
			`"message":"rpc error method unknown not found"},` +
			`"id":1,` +
			`"jsonrpc":"2.0"}` + "\n")},
	{
		call: []byte{},
		// empty request
		expected: []byte(`{"jsonrpc":"2.0","error":{"code":-32600,"message":"Invalid request"},"id":0}` + "\n")},
	{
		call:     []byte(`{"jsonrpc":"2.0","method":"chain_subscribeNewHeads","params":[],"id":3}`),
		expected: []byte(`{"jsonrpc":"2.0","result":1,"id":3}` + "\n")},
	{
		call:     []byte(`{"jsonrpc":"2.0","method":"state_subscribeStorage","params":[],"id":4}`),
		expected: []byte(`{"jsonrpc":"2.0","result":2,"id":4}` + "\n")},
	{
		call:     []byte(`{"jsonrpc":"2.0","method":"chain_subscribeFinalizedHeads","params":[],"id":5}`),
		expected: []byte(`{"jsonrpc":"2.0","result":3,"id":5}` + "\n")},
	{
		call: []byte(`{"jsonrpc":"2.0","method":"author_submitAndWatchExtrinsic","params":["0x010203"],"id":6}`),
		expected: []byte(`{"jsonrpc":"2.0","error":{"code":null,"message":"Failed to call the ` +
			"`" + `TaggedTransactionQueue_validate_transaction` + "`" + ` exported function."},"id":6}` + "\n")},
	{
		call:     []byte(`{"jsonrpc":"2.0","method":"state_subscribeRuntimeVersion","params":[],"id":7}`),
		expected: []byte(`{"jsonrpc":"2.0","result":6,"id":7}` + "\n")},
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
