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

// PostRPC sends a payload using the method, host and params string given.
// It returns the response bytes and an eventual error.
func PostRPC(ctx context.Context, endpoint, method, params string) (data []byte, err error) {
	requestBody := fmt.Sprintf(`{"jsonrpc":"2.0","method":"%s","params":%s,"id":1}`, method, params)
	requestBuffer := bytes.NewBuffer([]byte(requestBody))

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, requestBuffer)
	if err != nil {
		return nil, fmt.Errorf("cannot create HTTP request: %w", err)
	}

	const contentType = "application/json"
	request.Header.Set("Content-Type", contentType)
	request.Header.Set("Accept", contentType)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("cannot do HTTP request: %w", err)
	}

	data, err = io.ReadAll(response.Body)
	if err != nil {
		_ = response.Body.Close()
		return nil, fmt.Errorf("cannot read HTTP response body: %w", err)
	}

	err = response.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("cannot close HTTP response body: %w", err)
	}

	return data, nil
}

// PostRPCWithRetry repeatitively calls `PostRPC` repeatitively
// until it succeeds within the requestWait duration or returns
// the last error if the context is canceled.
func PostRPCWithRetry(ctx context.Context, endpoint, method, params string,
	requestWait time.Duration) (data []byte, err error) {
	try := 0
	for {
		try++

		postRPCCtx, postRPCCancel := context.WithTimeout(ctx, requestWait)

		data, err = PostRPC(postRPCCtx, endpoint, method, params)

		if err == nil {
			postRPCCancel()
			return data, nil
		}

		// wait for full requestWait duration or main context cancelation
		<-postRPCCtx.Done()
		postRPCCancel()

		if ctx.Err() != nil {
			break
		}
	}

	totalTime := time.Duration(try) * requestWait
	tryWord := "try"
	if try > 1 {
		tryWord = "tries"
	}
	return nil, fmt.Errorf("after %d %s totalling %s: %w", try, tryWord, totalTime, err)
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
