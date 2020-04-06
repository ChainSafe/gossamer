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
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewHTTPServer(t *testing.T) {

	cfg := &HTTPServerConfig{
		Modules: []string{"system"},
		Port:    8545,
	}
	s := NewHTTPServer(cfg)
	err := s.Start()
	require.Nil(t, err)

	// Valid request
	client := &http.Client{}
	data := []byte(`{"jsonrpc":"2.0","method":"system_name","params":[],"id":1}`)

	buf := &bytes.Buffer{}
	_, err = buf.Write(data)
	require.Nil(t, err)
	req, err := http.NewRequest("POST", "http://localhost:8545/", buf)
	require.Nil(t, err)

	req.Header.Set("Content-Type", "application/json;")

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

	require.Equal(t, "400 Bad Request", res.Status)

	// GET
	req, err = http.NewRequest("GET", "http://localhost:8545/", nil)
	require.Nil(t, err)

	req.Header.Set("Content-Type", "application/json;")

	res, err = client.Do(req)
	require.Nil(t, err)
	defer res.Body.Close()

	require.Equal(t, "405 Method Not Allowed", res.Status)
}
