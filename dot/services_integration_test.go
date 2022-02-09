// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

//go:build integration
// +build integration

package dot

import (
	"flag"
	"net/url"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/grandpa"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

func TestCreateStateService(t *testing.T) {
	cfg := NewTestConfig(t)
	require.NotNil(t, cfg)

	genFile := NewTestGenesisRawFile(t, cfg)

	cfg.Init.Genesis = genFile

	err := InitNode(cfg)
	require.NoError(t, err)

	stateSrvc, err := createStateService(cfg)
	require.NoError(t, err)
	require.NotNil(t, stateSrvc)
}

func TestCreateCoreService(t *testing.T) {
	cfg := NewTestConfig(t)
	require.NotNil(t, cfg)

	genFile := NewTestGenesisRawFile(t, cfg)

	cfg.Core.Roles = types.FullNodeRole
	cfg.Core.BabeAuthority = false
	cfg.Core.GrandpaAuthority = false
	cfg.Init.Genesis = genFile

	err := InitNode(cfg)
	require.NoError(t, err)

	stateSrvc, err := createStateService(cfg)
	require.NoError(t, err)

	ks := keystore.NewGlobalKeystore()
	require.NotNil(t, ks)
	ed25519Keyring, _ := keystore.NewEd25519Keyring()
	ks.Gran.Insert(ed25519Keyring.Alice())

	networkSrvc := &network.Service{}

	dh, err := createDigestHandler(cfg.Log.DigestLvl, stateSrvc)
	require.NoError(t, err)

	coreSrvc, err := createCoreService(cfg, ks, stateSrvc, networkSrvc, dh)
	require.NoError(t, err)
	require.NotNil(t, coreSrvc)
}

func TestCreateBlockVerifier(t *testing.T) {
	cfg := NewTestConfig(t)
	require.NotNil(t, cfg)

	genFile := newTestGenesisFile(t, cfg)

	cfg.Init.Genesis = genFile

	err := InitNode(cfg)
	require.NoError(t, err)

	stateSrvc, err := createStateService(cfg)
	require.NoError(t, err)

	_, err = createBlockVerifier(stateSrvc)
	require.NoError(t, err)
}

func TestCreateSyncService(t *testing.T) {
	cfg := NewTestConfig(t)
	require.NotNil(t, cfg)

	genFile := newTestGenesisFile(t, cfg)

	cfg.Init.Genesis = genFile

	err := InitNode(cfg)
	require.NoError(t, err)

	stateSrvc, err := createStateService(cfg)
	require.NoError(t, err)

	ks := keystore.NewGlobalKeystore()
	require.NotNil(t, ks)

	ver, err := createBlockVerifier(stateSrvc)
	require.NoError(t, err)

	dh, err := createDigestHandler(cfg.Log.DigestLvl, stateSrvc)
	require.NoError(t, err)

	coreSrvc, err := createCoreService(cfg, ks, stateSrvc, &network.Service{}, dh)
	require.NoError(t, err)

	_, err = newSyncService(cfg, stateSrvc, &grandpa.Service{}, ver, coreSrvc, &network.Service{}, nil)
	require.NoError(t, err)
}

func TestCreateNetworkService(t *testing.T) {
	cfg := NewTestConfig(t)
	require.NotNil(t, cfg)

	genFile := NewTestGenesisRawFile(t, cfg)

	cfg.Init.Genesis = genFile

	err := InitNode(cfg)
	require.NoError(t, err)

	stateSrvc, err := createStateService(cfg)
	require.NoError(t, err)

	networkSrvc, err := createNetworkService(cfg, stateSrvc, nil)
	require.NoError(t, err)
	require.NotNil(t, networkSrvc)
}

func TestCreateRPCService(t *testing.T) {
	cfg := NewTestConfig(t)
	require.NotNil(t, cfg)

	genFile := NewTestGenesisRawFile(t, cfg)

	cfg.Core.Roles = types.FullNodeRole
	cfg.Core.BabeAuthority = false
	cfg.Core.GrandpaAuthority = false
	cfg.Init.Genesis = genFile

	err := InitNode(cfg)
	require.NoError(t, err)

	stateSrvc, err := createStateService(cfg)
	require.NoError(t, err)

	networkSrvc := &network.Service{}

	ks := keystore.NewGlobalKeystore()
	ed25519Keyring, _ := keystore.NewEd25519Keyring()
	ks.Gran.Insert(ed25519Keyring.Alice())

	ns, err := createRuntimeStorage(stateSrvc)
	require.NoError(t, err)
	err = loadRuntime(cfg, ns, stateSrvc, ks, networkSrvc)
	require.NoError(t, err)

	dh, err := createDigestHandler(cfg.Log.DigestLvl, stateSrvc)
	require.NoError(t, err)

	coreSrvc, err := createCoreService(cfg, ks, stateSrvc, networkSrvc, dh)
	require.NoError(t, err)

	sysSrvc, err := createSystemService(&cfg.System, stateSrvc)
	require.NoError(t, err)

	rpcSrvc, err := createRPCService(cfg, ns, stateSrvc, coreSrvc, networkSrvc, nil, sysSrvc, nil)
	require.NoError(t, err)
	require.NotNil(t, rpcSrvc)
}

func TestCreateBABEService(t *testing.T) {
	cfg := NewTestConfig(t)

	genFile := NewTestGenesisRawFile(t, cfg)

	cfg.Core.Roles = types.FullNodeRole
	cfg.Init.Genesis = genFile

	err := InitNode(cfg)
	require.NoError(t, err)

	stateSrvc, err := createStateService(cfg)
	require.NoError(t, err)

	ks := keystore.NewGlobalKeystore()
	kr, err := keystore.NewSr25519Keyring()
	require.NoError(t, err)
	ks.Babe.Insert(kr.Alice())

	ns, err := createRuntimeStorage(stateSrvc)
	require.NoError(t, err)
	err = loadRuntime(cfg, ns, stateSrvc, ks, &network.Service{})
	require.NoError(t, err)

	dh, err := createDigestHandler(cfg.Log.DigestLvl, stateSrvc)
	require.NoError(t, err)

	coreSrvc, err := createCoreService(cfg, ks, stateSrvc, &network.Service{}, dh)
	require.NoError(t, err)

	bs, err := createBABEService(cfg, stateSrvc, ks.Babe, coreSrvc, nil)
	require.NoError(t, err)
	require.NotNil(t, bs)
}

func TestCreateGrandpaService(t *testing.T) {
	cfg := NewTestConfig(t)
	require.NotNil(t, cfg)

	genFile := NewTestGenesisRawFile(t, cfg)

	cfg.Core.Roles = types.AuthorityRole
	cfg.Init.Genesis = genFile

	err := InitNode(cfg)
	require.NoError(t, err)

	stateSrvc, err := createStateService(cfg)
	require.NoError(t, err)

	ks := keystore.NewGlobalKeystore()
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)
	ks.Gran.Insert(kr.Alice())

	ns, err := createRuntimeStorage(stateSrvc)
	require.NoError(t, err)

	err = loadRuntime(cfg, ns, stateSrvc, ks, &network.Service{})
	require.NoError(t, err)

	dh, err := createDigestHandler(cfg.Log.DigestLvl, stateSrvc)
	require.NoError(t, err)

	gs, err := createGRANDPAService(cfg, stateSrvc, dh, ks.Gran, &network.Service{}, nil)
	require.NoError(t, err)
	require.NotNil(t, gs)
}

var addr = flag.String("addr", "localhost:8546", "http service address")
var testCalls = []struct {
	call     []byte
	expected []byte
}{
	{[]byte(`{"jsonrpc":"2.0","method":"system_name","params":[],"id":1}`),
		[]byte(`{"id":1,"jsonrpc":"2.0","result":"gossamer"}` + "\n")}, // working request
	{[]byte(`{"jsonrpc":"2.0","method":"unknown","params":[],"id":2}`),
		[]byte(`{"error":{"code":-32000,"data":null,"message":"rpc error method unknown not found"},"id":2,
"jsonrpc":"2.0"}` + "\n")}, // unknown method
	{[]byte{},
		[]byte(`{"jsonrpc":"2.0","error":{"code":-32600,"message":"Invalid request"},"id":0}` + "\n")}, // empty request
	{[]byte(`{"jsonrpc":"2.0","method":"chain_subscribeNewHeads","params":[],"id":3}`),
		[]byte(`{"jsonrpc":"2.0","result":1,"id":3}` + "\n")},
	{[]byte(`{"jsonrpc":"2.0","method":"state_subscribeStorage","params":[],"id":4}`),
		[]byte(`{"jsonrpc":"2.0","result":2,"id":4}` + "\n")},
}

func TestNewWebSocketServer(t *testing.T) {
	cfg := NewTestConfig(t)
	require.NotNil(t, cfg)

	genFile := NewTestGenesisRawFile(t, cfg)

	cfg.Core.Roles = types.FullNodeRole
	cfg.Core.BabeAuthority = false
	cfg.Core.GrandpaAuthority = false
	cfg.Init.Genesis = genFile
	cfg.RPC.External = false
	cfg.RPC.WS = true
	cfg.RPC.WSExternal = false
	cfg.System.SystemName = "gossamer"

	err := InitNode(cfg)
	require.NoError(t, err)

	stateSrvc, err := createStateService(cfg)
	require.NoError(t, err)

	networkSrvc := &network.Service{}

	ks := keystore.NewGlobalKeystore()
	ed25519Keyring, _ := keystore.NewEd25519Keyring()
	ks.Gran.Insert(ed25519Keyring.Alice())

	ns, err := createRuntimeStorage(stateSrvc)
	require.NoError(t, err)
	err = loadRuntime(cfg, ns, stateSrvc, ks, networkSrvc)
	require.NoError(t, err)

	dh, err := createDigestHandler(cfg.Log.DigestLvl, stateSrvc)
	require.NoError(t, err)

	coreSrvc, err := createCoreService(cfg, ks, stateSrvc, networkSrvc, dh)
	require.NoError(t, err)

	sysSrvc, err := createSystemService(&cfg.System, stateSrvc)
	require.NoError(t, err)

	rpcSrvc, err := createRPCService(cfg, ns, stateSrvc, coreSrvc, networkSrvc, nil, sysSrvc, nil)
	require.NoError(t, err)
	err = rpcSrvc.Start()
	require.NoError(t, err)

	time.Sleep(time.Second) // give server a second to start

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/"}

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	require.NoError(t, err)
	defer c.Close()

	for _, item := range testCalls {
		err = c.WriteMessage(websocket.TextMessage, item.call)
		require.NoError(t, err)

		_, message, err := c.ReadMessage()
		require.NoError(t, err)
		require.Equal(t, item.expected, message)
	}
}
