// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package rpc

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/ChainSafe/gossamer/dot/core"
	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/ChainSafe/gossamer/dot/rpc/modules/mocks"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/runtime/wasmer"
	"github.com/btcsuite/btcutil/base58"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestRegisterModules(t *testing.T) {
	ctrl := gomock.NewController(t)
	rpcapiMocks := NewMockAPI(ctrl)

	mods := []string{
		"system", "author", "chain",
		"state", "rpc", "grandpa",
		"offchain", "childstate", "syncstate",
	}

	for _, modName := range mods {
		rpcapiMocks.EXPECT().BuildMethodNames(gomock.Any(), modName)
	}

	cfg := &HTTPServerConfig{
		Modules: mods,
		RPCAPI:  rpcapiMocks,
	}

	NewHTTPServer(cfg)
}

func TestUnsafeRPCProtection(t *testing.T) {
	cfg := &HTTPServerConfig{
		Modules:           []string{"system", "author", "chain", "state", "rpc", "grandpa", "dev", "syncstate"},
		RPCPort:           7878,
		RPCAPI:            NewService(),
		RPCUnsafeExternal: false,
		RPCUnsafe:         false,
	}

	s := NewHTTPServer(cfg)
	err := s.Start()
	require.NoError(t, err)

	time.Sleep(time.Second)
	defer s.Stop()

	for _, unsafe := range modules.UnsafeMethods {
		t.Run(fmt.Sprintf("Unsafe method %s should not be reachable", unsafe), func(t *testing.T) {
			data := []byte(fmt.Sprintf(`{"jsonrpc":"2.0","method":"%s","params":[],"id":1}`, unsafe))

			buf := new(bytes.Buffer)
			_, err = buf.Write(data)
			require.NoError(t, err)

			_, resBody := PostRequest(t, fmt.Sprintf("http://localhost:%v/", cfg.RPCPort), buf)
			expected := fmt.Sprintf(`{`+
				`"jsonrpc":"2.0",`+
				`"error":{`+
				`"code":-32000,`+
				`"message":"unsafe rpc method %s cannot be reachable",`+
				`"data":null`+
				`},`+
				`"id":1`+
				`}`+"\n",
				unsafe,
			)

			require.Equal(t, expected, string(resBody))
		})
	}
}

func TestRPCUnsafeExpose(t *testing.T) {
	ctrl := gomock.NewController(t)

	data := []byte(fmt.Sprintf(
		`{"jsonrpc":"2.0","method":"%s","params":["%s"],"id":1}`,
		"system_addReservedPeer",
		"/ip4/198.51.100.19/tcp/30333/p2p/QmSk5HQbn6LhUwDiNMseVUjuRYhEtYj4aUZ6WfWoGURpdV"))

	buf := new(bytes.Buffer)
	_, err := buf.Write(data)
	require.NoError(t, err)

	netmock := mocks.NewMockNetworkAPI(ctrl)
	netmock.EXPECT().AddReservedPeers(gomock.Any()).Return(nil)

	cfg := &HTTPServerConfig{
		Modules:           []string{"system"},
		RPCPort:           7879,
		RPCAPI:            NewService(),
		RPCUnsafeExternal: true,
		NetworkAPI:        netmock,
	}

	s := NewHTTPServer(cfg)
	err = s.Start()
	require.NoError(t, err)

	time.Sleep(time.Second)
	defer s.Stop()

	ip, err := externalIP()
	require.NoError(t, err)

	_, resBody := PostRequest(t, fmt.Sprintf("http://%s:%v/", ip, cfg.RPCPort), buf)
	expected := `{"jsonrpc":"2.0","result":null,"id":1}` + "\n"
	require.Equal(t, expected, string(resBody))
}

func TestUnsafeRPCJustToLocalhost(t *testing.T) {
	unsafeMethod := "system_addReservedPeer"
	data := []byte(fmt.Sprintf(
		`{"jsonrpc":"2.0","method":"%s","params":["%s"],"id":1}`,
		unsafeMethod,
		"/ip4/198.51.100.19/tcp/30333/p2p/QmSk5HQbn6LhUwDiNMseVUjuRYhEtYj4aUZ6WfWoGURpdV"))

	buf := new(bytes.Buffer)
	_, err := buf.Write(data)
	require.NoError(t, err)

	cfg := &HTTPServerConfig{
		Modules:           []string{"system"},
		RPCPort:           7880,
		RPCAPI:            NewService(),
		RPCUnsafe:         true,
		RPCUnsafeExternal: false,
	}

	s := NewHTTPServer(cfg)
	err = s.Start()
	require.NoError(t, err)

	time.Sleep(time.Second)
	defer s.Stop()

	ip, err := externalIP()
	require.NoError(t, err)

	_, resBody := PostRequest(t, fmt.Sprintf("http://%s:7880/", ip), buf)
	expected := `{` +
		`"jsonrpc":"2.0",` +
		`"error":{` +
		`"code":-32000,` +
		`"message":"external HTTP request refused",` +
		`"data":null` +
		`},` +
		`"id":1` +
		`}` + "\n"
	require.Equal(t, expected, string(resBody))
}

func TestRPCExternalEnable_UnsafeExternalNotEnabled(t *testing.T) {
	ctrl := gomock.NewController(t)

	unsafeData := []byte(fmt.Sprintf(
		`{"jsonrpc":"2.0","method":"%s","params":["%s"],"id":1}`,
		"system_addReservedPeer",
		"/ip4/198.51.100.19/tcp/30333/p2p/QmSk5HQbn6LhUwDiNMseVUjuRYhEtYj4aUZ6WfWoGURpdV"))
	unsafebuf := new(bytes.Buffer)
	unsafebuf.Write(unsafeData)

	safeData := []byte(fmt.Sprintf(
		`{"jsonrpc":"2.0","method":"%s","params":[],"id":2}`,
		"system_localPeerId"))
	safebuf := new(bytes.Buffer)
	safebuf.Write(safeData)

	netmock := mocks.NewMockNetworkAPI(ctrl)
	netmock.EXPECT().NetworkState().Return(common.NetworkState{
		PeerID: "peer id",
	})

	httpServerConfig := &HTTPServerConfig{
		Modules:           []string{"system"},
		RPCPort:           8786,
		RPCAPI:            NewService(),
		RPCExternal:       true,
		RPCUnsafeExternal: false,
		RPCUnsafe:         true,
		NetworkAPI:        netmock,
	}

	s := NewHTTPServer(httpServerConfig)
	err := s.Start()
	require.NoError(t, err)

	time.Sleep(time.Second)
	defer s.Stop()

	ip, err := externalIP()
	require.NoError(t, err)

	_, resBody := PostRequest(t, fmt.Sprintf("http://%s:%v/", ip, httpServerConfig.RPCPort), safebuf)
	encoded := base58.Encode([]byte("peer id"))
	expected := fmt.Sprintf(`{"jsonrpc":"2.0","result":"%s","id":2}`, encoded) + "\n"
	require.Equal(t, expected, string(resBody))

	// unsafe method should not be ok
	_, resBody = PostRequest(t, fmt.Sprintf("http://%s:%v/", ip, httpServerConfig.RPCPort), unsafebuf)
	expected = `{` +
		`"jsonrpc":"2.0",` +
		`"error":{` +
		`"code":-32000,` +
		`"message":"external HTTP request refused",` +
		`"data":null` +
		`},` +
		`"id":1` +
		`}` + "\n"
	require.Equal(t, expected, string(resBody))
}

func PostRequest(t *testing.T, url string, data io.Reader) (int, []byte) {
	t.Helper()

	req, err := http.NewRequest(http.MethodPost, url, data)
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")
	res, err := new(http.Client).Do(req)
	require.NoError(t, err)

	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	responseData := new(bytes.Buffer)
	_, err = responseData.Write(resBody)
	require.NoError(t, err)

	return res.StatusCode, responseData.Bytes()
}

func externalIP() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			return ip.String(), nil
		}
	}
	return "", errors.New("are you connected to the network?")
}

func newCoreServiceTest(t *testing.T) *core.Service {
	t.Helper()

	testDatadirPath := t.TempDir()

	gen, genesisTrie, genesisHeader := newWestendDevGenesisWithTrieAndHeader(t)

	ctrl := gomock.NewController(t)
	telemetryMock := NewMockTelemetry(ctrl)
	telemetryMock.EXPECT().SendMessage(gomock.Any()).AnyTimes()

	config := state.Config{
		Path:      testDatadirPath,
		LogLevel:  log.Debug,
		Telemetry: telemetryMock,
	}

	stateSrvc := state.NewService(config)
	stateSrvc.UseMemDB()

	err := stateSrvc.Initialise(&gen, &genesisHeader, &genesisTrie)
	require.NoError(t, err)

	err = stateSrvc.SetupBase()
	require.NoError(t, err)

	err = stateSrvc.Start()
	require.NoError(t, err)

	cfg := &core.Config{
		LogLvl:               log.Warn,
		BlockState:           stateSrvc.Block,
		StorageState:         stateSrvc.Storage,
		TransactionState:     stateSrvc.Transaction,
		CodeSubstitutedState: stateSrvc.Base,
	}

	cfg.Keystore = keystore.NewGlobalKeystore()
	kp, err := sr25519.GenerateKeypair()
	require.NoError(t, err)

	err = cfg.Keystore.Acco.Insert(kp)
	require.NoError(t, err)

	var rtCfg wasmer.Config

	rtCfg.Storage = rtstorage.NewTrieState(&genesisTrie)

	rtCfg.CodeHash, err = cfg.StorageState.(*state.StorageState).LoadCodeHash(nil)
	require.NoError(t, err)

	nodeStorage := runtime.NodeStorage{
		BaseDB: stateSrvc.Base,
	}
	rtCfg.NodeStorage = nodeStorage

	cfg.Runtime, err = wasmer.NewRuntimeFromGenesis(rtCfg)
	require.NoError(t, err)

	cfg.BlockState.StoreRuntime(cfg.BlockState.BestBlockHash(), cfg.Runtime)

	net := NewMockNetwork(ctrl)
	net.EXPECT().
		GossipMessage(gomock.AssignableToTypeOf(new(network.TransactionMessage))).
		AnyTimes()
	net.EXPECT().IsSynced().Return(true).AnyTimes()
	net.EXPECT().ReportPeer(
		gomock.AssignableToTypeOf(new(peerset.ReputationChange)),
		gomock.AssignableToTypeOf(peer.ID(""))).
		AnyTimes()
	cfg.Network = net

	cfg.CodeSubstitutes = make(map[common.Hash]string)

	genesisData, err := cfg.CodeSubstitutedState.(*state.BaseState).LoadGenesisData()
	require.NoError(t, err)

	for k, v := range genesisData.CodeSubstitutes {
		cfg.CodeSubstitutes[common.MustHexToHash(k)] = v
	}

	s, err := core.NewService(cfg)
	require.NoError(t, err)

	return s
}
