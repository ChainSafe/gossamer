// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"

	"github.com/stretchr/testify/require"
)

// PostRPC utils for sending payload to endpoint and getting []byte back
func PostRPC(method, host, params string) ([]byte, error) {
	data := []byte(`{"jsonrpc":"2.0","method":"` + method + `","params":` + params + `,"id":1}`)
	buf := &bytes.Buffer{}
	_, err := buf.Write(data)
	if err != nil {
		return nil, err
	}

	r, err := http.NewRequest("POST", host, buf)
	if err != nil {
		return nil, err
	}
	r.Header.Set("Content-Type", ContentTypeJSON)
	r.Header.Set("Accept", ContentTypeJSON)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	r = r.WithContext(ctx)

	resp, err := httpClient.Do(r)
	if err != nil {
		return nil, err
	} else if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status code not OK")
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	respBody, err := io.ReadAll(resp.Body)

	return respBody, err

}

// PostRPCWithRetry is a wrapper around `PostRPC` that calls it `retry` number of times.
func PostRPCWithRetry(method, host, params string, retry int) ([]byte, error) {
	count := 0
	for {
		resp, err := PostRPC(method, host, params)
		if err == nil || count >= retry {
			return resp, err
		}
		time.Sleep(200 * time.Millisecond)
		count++
	}
}

// DecodeRPC will decode []body into target interface
func DecodeRPC(t *testing.T, body []byte, target interface{}) error {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()

	var response ServerResponse
	err := decoder.Decode(&response)
	require.Nil(t, err, string(body))
	require.Equal(t, response.Version, "2.0")

	if response.Error != nil {
		return errors.New(response.Error.Message)
	}

	decoder = json.NewDecoder(bytes.NewReader(response.Result))
	decoder.DisallowUnknownFields()

	err = decoder.Decode(target)
	require.Nil(t, err, string(body))
	return nil
}

// DecodeWebsocket will decode body into target interface
func DecodeWebsocket(t *testing.T, body []byte, target interface{}) error {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()

	var response WebsocketResponse
	err := decoder.Decode(&response)
	require.Nil(t, err, string(body))
	require.Equal(t, response.Version, "2.0")

	if response.Error != nil {
		return errors.New(response.Error.Message)
	}

	if response.Result != nil {
		decoder = json.NewDecoder(bytes.NewReader(response.Result))
	} else {
		decoder = json.NewDecoder(bytes.NewReader(response.Params))
	}

	decoder.DisallowUnknownFields()

	err = decoder.Decode(target)
	require.Nil(t, err, string(body))
	return nil
}

// DecodeRPC_NT will decode []body into target interface (NT is Not Test testing required)
func DecodeRPC_NT(body []byte, target interface{}) error { //nolint:revive
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()

	var response ServerResponse
	err := decoder.Decode(&response)
	if err != nil {
		return err
	}

	if response.Error != nil {
		return errors.New(response.Error.Message)
	}

	decoder = json.NewDecoder(bytes.NewReader(response.Result))
	decoder.DisallowUnknownFields()

	err = decoder.Decode(target)
	return err
}

// NewEndpoint will create a new endpoint string based on utils.HOSTNAME and port
func NewEndpoint(port string) string {
	return "http://" + HOSTNAME + ":" + port
}

func rpcLogsToDigest(t *testing.T, logs []string) scale.VaryingDataTypeSlice {
	digest := types.NewDigest()

	for _, l := range logs {
		itemBytes, err := common.HexToBytes(l)
		require.NoError(t, err)

		var di = types.NewDigestItem()
		err = scale.Unmarshal(itemBytes, &di)
		require.NoError(t, err)

		err = digest.Add(di.Value())
		require.NoError(t, err)
	}

	return digest
}
