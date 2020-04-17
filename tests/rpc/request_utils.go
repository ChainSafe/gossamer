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
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"testing"

	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/stretchr/testify/require"
)

var (
	transport = &http.Transport{
		Dial: (&net.Dialer{
			Timeout: dialTimeout,
		}).Dial,
	}
	httpClient = &http.Client{
		Transport: transport,
		Timeout:   httpClientTimeout,
	}
)

func PostRPC(t *testing.T, method, host string) []byte {

	data := []byte(`{"jsonrpc":"2.0","method":"` + method + `","params":{},"id":1}`)
	buf := &bytes.Buffer{}
	_, err := buf.Write(data)
	require.Nil(t, err)

	r, err := http.NewRequest("POST", host, buf)
	require.Nil(t, err)

	r.Header.Set("Content-Type", ContentTypeJSON)
	r.Header.Set("Accept", ContentTypeJSON)

	resp, err := httpClient.Do(r)
	require.Nil(t, err)
	require.Equal(t, resp.StatusCode, http.StatusOK)

	defer func() {
		_ = resp.Body.Close()
	}()

	respBody, err := ioutil.ReadAll(resp.Body)
	require.Nil(t, err)

	return respBody

}

func DecodeRPC(t *testing.T, body []byte, targetType string) interface{} {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()

	var response ServerResponse
	err := decoder.Decode(&response)
	require.Nil(t, err, "respBody", string(body))

	t.Log("Got payload from RPC request", "serverResponse", response, "string(respBody)", string(body))

	require.Nil(t, response.Error)
	require.Equal(t, response.Version, "2.0")

	decoder = json.NewDecoder(bytes.NewReader(response.Result))
	decoder.DisallowUnknownFields()

	var target interface{}

	switch targetType {
	case "system_health":
		target = new(modules.SystemHealthResponse)
	case "system_networkState":
		target = new(modules.SystemNetworkStateResponse)
	case "system_peers":
		target = new(modules.SystemPeersResponse)
	}

	err = decoder.Decode(&target)
	require.Nil(t, err)

	return target

}
