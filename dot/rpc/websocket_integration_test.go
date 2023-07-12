//go:build integration

// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package rpc

import (
	"flag"
	"log"
	"net/url"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/ChainSafe/gossamer/dot/system"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/transaction"
	"github.com/golang/mock/gomock"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
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
		call: []byte(`{"jsonrpc":"2.0","method":"author_submitAndWatchExtrinsic","params":["0x4102840062136` +
			`fef1fa4136a3932066121c623417022c4447ebfb3d0cddbfc1bc68057070148356becc7653f6120ae247cfd757f36dd4e53` +
			`0c9f9c1a9370b034324665727ef3e3c8e7b3d2b432bb93ed01a17c86a5f8e01cf200583c2a22281dec21a6348605021c000` +
			`4070000ac81b86d05495a73dd7e98d33fb5bf55a837c6b87e3da0bf45618fed00be6d070010a5d4e8"],"id":6}`),
		expected: []byte(`{"jsonrpc":"2.0","method":"author_extrinsicUpdate",` +
			`"params":{"result":"invalid","subscription":4}}` + "\n")},
	{
		call:     []byte(`{"jsonrpc":"2.0","method":"state_subscribeRuntimeVersion","params":[],"id":7}`),
		expected: []byte(`{"jsonrpc":"2.0","result":6,"id":7}` + "\n")},
}

func TestHTTPServer_ServeHTTP(t *testing.T) {
	ctrl := gomock.NewController(t)

	coreAPI := newCoreServiceTest(t)
	si := &types.SystemInfo{
		SystemName: "gossamer",
	}
	sysAPI := system.NewService(si, nil)
	bAPI := modules.NewMockAnyBlockAPI(ctrl)
	sAPI := modules.NewMockAnyStorageAPI(ctrl)

	TxStateAPI := NewMockTransactionStateAPI(ctrl)
	TxStateAPI.EXPECT().GetStatusNotifierChannel(gomock.Any()).Return(make(chan transaction.Status))

	cfg := &HTTPServerConfig{
		Modules:             []string{"system", "chain"},
		RPCExternal:         false,
		RPCPort:             8545,
		WSPort:              8546,
		WSExternal:          true,
		RPCAPI:              NewService(),
		CoreAPI:             coreAPI,
		SystemAPI:           sysAPI,
		BlockAPI:            bAPI,
		StorageAPI:          sAPI,
		TransactionQueueAPI: TxStateAPI,
	}

	s := NewHTTPServer(cfg)
	err := s.Start()
	require.NoError(t, err)

	defer s.Stop()

	time.Sleep(time.Second) // give server a second to start

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/"}
	log.Printf("connecting to %s", u.String())

	c, response, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer func() {
		err := response.Body.Close()
		assert.NoError(t, err)
	}()
	defer c.Close()

	for _, item := range testCalls {
		err = c.WriteMessage(websocket.TextMessage, item.call)
		require.NoError(t, err)

		_, message, err := c.ReadMessage()
		require.NoError(t, err)
		require.Equal(t, string(item.expected), string(message))
	}
}
