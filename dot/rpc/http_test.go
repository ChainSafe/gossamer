// Copyright 2020 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package rpc

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
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

	// Valid request
	client := &http.Client{}
	data := []byte(`{"jsonrpc":"2.0","method":"system_name","params":[],"id":1}`)

	buf := &bytes.Buffer{}
	_, err = buf.Write(data)
	require.Nil(t, err)
	req, err := http.NewRequest("POST", "http://localhost:8545/", buf)
	require.Nil(t, err)

	req.Header.Set("Content-Type", "application/json")

	res, err := client.Do(req)
	require.Nil(t, err)
	defer res.Body.Close()

	require.Equal(t, "200 OK", res.Status)

	// nil POST
	req, err = http.NewRequest("POST", "http://localhost:8545/", nil)
	require.Nil(t, err)

	req.Header.Set("Content-Type", "application/json;")

	res, err = client.Do(req)
	require.Nil(t, err)
	defer res.Body.Close()

	require.Equal(t, "200 OK", res.Status)

	// GET
	req, err = http.NewRequest("GET", "http://localhost:8545/", nil)
	require.Nil(t, err)

	req.Header.Set("Content-Type", "application/json;")

	res, err = client.Do(req)
	require.Nil(t, err)
	defer res.Body.Close()

	require.Equal(t, "405 Method Not Allowed", res.Status)
}

func TestUnsafeRPCProtection(t *testing.T) {
	cfg := &HTTPServerConfig{
		Modules: []string{"author"},
		RPCPort: 7878,
		RPCAPI:  NewService(),
	}

	s := NewHTTPServer(cfg)
	err := s.Start()
	require.NoError(t, err)

	time.Sleep(time.Second)

	for _, unsafe := range modules.UNSAFE_METHODS {
		func(method string) {
			t.Run(fmt.Sprintf("Unsafe method %s should not be reachable", method), func(t *testing.T) {
				data := []byte(fmt.Sprintf(`{"jsonrpc":"2.0","method":"%s","params":["0x00"],"id":1}`, method))

				buf := new(bytes.Buffer)
				_, err = buf.Write(data)
				require.NoError(t, err)

				_, resBody := makePOSTRequest(t, "http://localhost:7878/", buf)
				expected := fmt.Sprintf(
					`{"jsonrpc":"2.0","error":{"code":-32000,"message":"unsafe rpc method %s cannot be reachable","data":null},"id":1}`+"\n",
					method,
				)

				require.Equal(t, expected, string(resBody))
			})
		}(unsafe)
	}
}
func TestRPCUnsafeExpose(t *testing.T) {
	unsafeMethod := "system_addReservedPeer"
	data := []byte(fmt.Sprintf(
		`{"jsonrpc":"2.0","method":"%s","params":["%s"],"id":1}`,
		unsafeMethod,
		"/ip4/198.51.100.19/tcp/30333/p2p/QmSk5HQbn6LhUwDiNMseVUjuRYhEtYj4aUZ6WfWoGURpdV"))

	buf := new(bytes.Buffer)
	_, err := buf.Write(data)
	require.NoError(t, err)

	netmock := new(mocks.MockNetworkAPI)
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

	req, err := http.NewRequest(http.MethodPost, "http://localhost:7879/", buf)
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-For", "200.189.72.10")

	res, err := new(http.Client).Do(req)
	require.NoError(t, err)

	resBody, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	require.NoError(t, err)

	expected := `{"jsonrpc":"2.0","result":null,"id":1}` + "\n"
	require.Equal(t, expected, string(resBody))
}

func TestRPCJustToLocalhost(t *testing.T) {
	unsafeMethod := "system_addReservedPeer"
	data := []byte(fmt.Sprintf(
		`{"jsonrpc":"2.0","method":"%s","params":["%s"],"id":1}`,
		unsafeMethod,
		"/ip4/198.51.100.19/tcp/30333/p2p/QmSk5HQbn6LhUwDiNMseVUjuRYhEtYj4aUZ6WfWoGURpdV"))

	buf := new(bytes.Buffer)
	_, err := buf.Write(data)
	require.NoError(t, err)

	netmock := new(mocks.MockNetworkAPI)
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

	ip, err := externalIP()
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://%s:7880/", ip), buf)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	res, err := new(http.Client).Do(req)
	require.NoError(t, err)

	resBody, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	require.NoError(t, err)

	expected := `{"jsonrpc":"2.0","error":{"code":-32000,"message":"external HTTP request refused","data":null},"id":1}` + "\n"
	require.Equal(t, expected, string(resBody))
}

func TestRPCExternalEnableButUnsafeDont(t *testing.T) {
	unsafeMethod := "system_addReservedPeer"
	unsafeData := []byte(fmt.Sprintf(
		`{"jsonrpc":"2.0","method":"%s","params":["%s"],"id":1}`,
		unsafeMethod,
		"/ip4/198.51.100.19/tcp/30333/p2p/QmSk5HQbn6LhUwDiNMseVUjuRYhEtYj4aUZ6WfWoGURpdV"))
	unsafebuf := new(bytes.Buffer)
	unsafebuf.Write(unsafeData)

	safeMethod := "system_localPeerId"
	safeData := []byte(fmt.Sprintf(
		`{"jsonrpc":"2.0","method":"%s","params":[],"id":2}`,
		safeMethod))
	safebuf := new(bytes.Buffer)
	safebuf.Write(safeData)

	netmock := new(mocks.MockNetworkAPI)
	netmock.On("NetworkState").Return(common.NetworkState{
		PeerID: "peer id",
	})

	cfg := &HTTPServerConfig{
		Modules:     []string{"system"},
		RPCPort:     7880,
		RPCAPI:      NewService(),
		RPCUnsafe:   true,
		RPCExternal: true,
		NetworkAPI:  netmock,
	}

	s := NewHTTPServer(cfg)
	err := s.Start()
	require.NoError(t, err)

	time.Sleep(time.Second)

	ip, err := externalIP()
	require.NoError(t, err)

	// safe method should be ok
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://%s:7880/", ip), safebuf)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	res, err := new(http.Client).Do(req)
	require.NoError(t, err)

	resBody, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	require.NoError(t, err)

	encoded := base58.Encode([]byte("peer id"))
	expected := fmt.Sprintf(`{"jsonrpc":"2.0","result":"%s","id":2}`, encoded) + "\n"
	require.Equal(t, expected, string(resBody))

	// unsafe method should not be ok
	req, err = http.NewRequest(http.MethodPost, fmt.Sprintf("http://%s:7880/", ip), unsafebuf)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	res, err = new(http.Client).Do(req)
	require.NoError(t, err)

	resBody, err = ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	require.NoError(t, err)

	expected = `{"jsonrpc":"2.0","error":{"code":-32000,"message":"external HTTP request refused","data":null},"id":1}` + "\n"
	require.Equal(t, expected, string(resBody))
}

func makePOSTRequest(t *testing.T, url string, data io.Reader) (int, []byte) {
	req, err := http.NewRequest(http.MethodPost, "http://localhost:7878/", data)
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")
	res, err := new(http.Client).Do(req)
	require.NoError(t, err)

	defer res.Body.Close()

	resBody, err := ioutil.ReadAll(res.Body)
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
