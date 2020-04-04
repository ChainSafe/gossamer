// Copyright 2019 ChainSafe Systems (ON) Corp.
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
	"github.com/ChainSafe/gossamer/dot/rpc/json2"
	log "github.com/ChainSafe/log15"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"testing"
	"time"
)

var (
	GOSSAMER_INTEGRATION_TEST_MODE = os.Getenv("GOSSAMER_INTEGRATION_TEST_MODE")

	GOSSAMER_NODE_HOST = os.Getenv("GOSSAMER_NODE_HOST")

	ContentTypeJSON   = "application/json"
	dialTimeout       = 60 * time.Second
	httpClientTimeout = 120 * time.Second
)

//TODO: json2.serverResponse should be exported and re-used instead
type serverResponse struct {
	// JSON-RPC Version
	Version string `json:"jsonrpc"`
	// Resulting values
	Result interface{} `json:"result"`
	// Any generated errors
	Error *json2.Error `json:"error"`
	// Request id
	ID *json.RawMessage `json:"id"`
}

func TestStableRPC(t *testing.T) {
	if GOSSAMER_INTEGRATION_TEST_MODE != "stable" {
		t.Skip("Integration tests are disabled, going to skip.")
	}

	log.Info("Going to run tests",
		"GOSSAMER_INTEGRATION_TEST_MODE", GOSSAMER_INTEGRATION_TEST_MODE,
		"GOSSAMER_NODE_HOST", GOSSAMER_NODE_HOST)

	r, err := http.NewRequest("POST", GOSSAMER_NODE_HOST, nil)
	require.Nil(t, err)

	r.Header.Set("Content-Type", ContentTypeJSON)
	r.Header.Set("Accept", ContentTypeJSON)

	transport := &http.Transport{
		Dial: (&net.Dialer{
			Timeout: dialTimeout,
		}).Dial,
	}
	httpClient := &http.Client{
		Transport: transport,
		Timeout:   httpClientTimeout,
	}

	resp, err := httpClient.Do(r)
	require.Nil(t, err)
	require.Equal(t, resp.StatusCode, http.StatusOK)

	defer func() {
		_ = resp.Body.Close()
	}()

	respBody, err := ioutil.ReadAll(resp.Body)
	require.Nil(t, err)

	decoder := json.NewDecoder(bytes.NewReader(respBody))
	decoder.DisallowUnknownFields()

	var serverResponse serverResponse
	err = decoder.Decode(&serverResponse)
	require.Nil(t, err)

	log.Debug("Got payload from RPC request", "serverResponse", serverResponse)

	require.Equal(t, serverResponse.Version, "2.0")

	//TODO: add further assertions
	//require.Nil(t, serverResponse.Error)

}
