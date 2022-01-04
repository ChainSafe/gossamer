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

	"github.com/ChainSafe/gossamer/dot/core"
	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/ChainSafe/gossamer/dot/rpc/modules/mocks"
	"github.com/ChainSafe/gossamer/dot/system"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/btcsuite/btcutil/base58"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestRegisterModules(t *testing.T) {
	rpcapiMocks := new(mocks.RPCAPI)

	mods := []string{
		"system", "author", "chain",
		"state", "rpc", "grandpa",
		"offchain", "childstate", "syncstate",
	}

	for _, modName := range mods {
		rpcapiMocks.On("BuildMethodNames", mock.Anything, modName).Once()
	}

	cfg := &HTTPServerConfig{
		Modules: mods,
		RPCAPI:  rpcapiMocks,
	}

	NewHTTPServer(cfg)

	for _, modName := range mods {
		rpcapiMocks.AssertCalled(t, "BuildMethodNames", mock.Anything, modName)
	}
}

func TestNewHTTPServer(t *testing.T) {
	coreAPI := core.NewTestService(t, nil)
	si := &types.SystemInfo{
		SystemName: "gossamer",
	}
	sysAPI := system.NewService(si, nil)
	cfg := &HTTPServerConfig{
		Modules:   []string{"system"},
		RPCPort:   8545,
		RPCAPI:    NewService(),
		CoreAPI:   coreAPI,
		SystemAPI: sysAPI,
	}

	s := NewHTTPServer(cfg)
	err := s.Start()
	require.Nil(t, err)

	time.Sleep(time.Second) // give server a second to start
	defer s.Stop()

	// Valid request
	client := &http.Client{}
	data := []byte(`{"jsonrpc":"2.0","method":"system_name","params":[],"id":1}`)

	buf := &bytes.Buffer{}
	_, err = buf.Write(data)
	require.Nil(t, err)
	req, err := http.NewRequest("POST", fmt.Sprintf("http://localhost:%v/", cfg.RPCPort), buf)
	require.Nil(t, err)

	req.Header.Set("Content-Type", "application/json")

	res, err := client.Do(req)
	require.Nil(t, err)
	defer res.Body.Close()

	require.Equal(t, "200 OK", res.Status)

	// nil POST
	req, err = http.NewRequest("POST", fmt.Sprintf("http://localhost:%v/", cfg.RPCPort), nil)
	require.Nil(t, err)

	req.Header.Set("Content-Type", "application/json;")

	res, err = client.Do(req)
	require.Nil(t, err)
	defer res.Body.Close()

	require.Equal(t, "200 OK", res.Status)

	// GET
	req, err = http.NewRequest("GET", fmt.Sprintf("http://localhost:%v/", cfg.RPCPort), nil)
	require.Nil(t, err)

	req.Header.Set("Content-Type", "application/json;")

	res, err = client.Do(req)
	require.Nil(t, err)
	defer res.Body.Close()

	require.Equal(t, "405 Method Not Allowed", res.Status)
}

func TestUnsafeRPCProtection(t *testing.T) {
	cfg := &HTTPServerConfig{
		Modules:           []string{"system", "author", "chain", "state", "rpc", "grandpa", "dev", "syncstate"},
		RPCPort:           7878,
		RPCAPI:            NewService(),
		RPCUnsafe:         false,
		RPCUnsafeExternal: false,
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
	data := []byte(fmt.Sprintf(
		`{"jsonrpc":"2.0","method":"%s","params":["%s"],"id":1}`,
		"system_addReservedPeer",
		"/ip4/198.51.100.19/tcp/30333/p2p/QmSk5HQbn6LhUwDiNMseVUjuRYhEtYj4aUZ6WfWoGURpdV"))

	buf := new(bytes.Buffer)
	_, err := buf.Write(data)
	require.NoError(t, err)

	netmock := new(mocks.NetworkAPI)
	netmock.On("AddReservedPeers", mock.AnythingOfType("string")).Return(nil)

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

	netmock := new(mocks.NetworkAPI)
	netmock.On("AddReservedPeers", mock.AnythingOfType("string")).Return(nil)

	cfg := &HTTPServerConfig{
		Modules:    []string{"system"},
		RPCPort:    7880,
		RPCAPI:     NewService(),
		RPCUnsafe:  true,
		NetworkAPI: netmock,
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

	netmock := new(mocks.NetworkAPI)
	netmock.On("NetworkState").Return(common.NetworkState{
		PeerID: "peer id",
	})

	httpServerConfig := &HTTPServerConfig{
		Modules:           []string{"system"},
		RPCPort:           8786,
		RPCAPI:            NewService(),
		RPCUnsafe:         true,
		RPCUnsafeExternal: false,
		RPCExternal:       true,
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
