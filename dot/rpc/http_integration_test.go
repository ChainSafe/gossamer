// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

//go:build integration
// +build integration

package rpc

import (
	"bytes"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/system"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/stretchr/testify/require"
)

func TestNewHTTPServer(t *testing.T) {
	coreAPI := newCoreServiceTest(t)
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
	require.NoError(t, err)

	time.Sleep(time.Second) // give server a second to start
	defer s.Stop()

	// Valid request
	client := &http.Client{}
	data := []byte(`{"jsonrpc":"2.0","method":"system_name","params":[],"id":1}`)

	buf := &bytes.Buffer{}
	_, err = buf.Write(data)
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://localhost:%v/", cfg.RPCPort), buf)
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")

	res, err := client.Do(req)
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, "200 OK", res.Status)

	// nil POST
	req, err = http.NewRequest(http.MethodPost, fmt.Sprintf("http://localhost:%v/", cfg.RPCPort), nil)
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json;")

	res, err = client.Do(req)
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, "200 OK", res.Status)

	// GET
	req, err = http.NewRequest(http.MethodGet, fmt.Sprintf("http://localhost:%v/", cfg.RPCPort), nil)
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json;")

	res, err = client.Do(req)
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, "405 Method Not Allowed", res.Status)
}
