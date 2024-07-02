// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// Post sends a payload using the method, host and params string given.
// It returns the response bytes and an eventual error.
func Post(ctx context.Context, endpoint, method, params string) (data []byte, err error) {
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

var (
	ErrResponseVersion = errors.New("unexpected response version received")
	ErrResponseError   = errors.New("response error received")
)

// Decode decodes []body into the target interface.
// It assumes that the response.Result is an optional JSON-encoded value.
func Decode(body []byte, target interface{}) error {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()

	var response ServerResponse
	err := decoder.Decode(&response)
	if err != nil {
		return fmt.Errorf("cannot decode response: %s: %w",
			string(body), err)
	}

	if response.Version != "2.0" {
		return fmt.Errorf("%w: %s", ErrResponseVersion, response.Version)
	}

	if response.Error != nil {
		return fmt.Errorf("%w: %s (error code %d)",
			ErrResponseError, response.Error.Message, response.Error.ErrorCode)
	}

	jsonRawMessage := response.Result
	if jsonRawMessage == nil {
		jsonRawMessage = response.Params
	}
	decoder = json.NewDecoder(bytes.NewReader(jsonRawMessage))
	decoder.DisallowUnknownFields()

	err = decoder.Decode(target)
	if err != nil {
		return fmt.Errorf("cannot decode response result: %s: %w",
			string(response.Result), err)
	}

	return nil
}

// DecodeScaleUnmarshal decodes []body into the result.
// It assumes that the response.Result is a SCALE-encoded value.
func DecodeScaleUnmarshal(body []byte, target any) error {
	var result string
	err := Decode(body, &result)
	if err != nil {
		return fmt.Errorf("decode body: %w", err)
	}

	resultBytes, err := common.HexToBytes(result)
	if err != nil {
		return fmt.Errorf("convert hex to bytes: %w", err)
	}

	err = scale.Unmarshal(resultBytes, target)
	if err != nil {
		return fmt.Errorf("decode result: %w", err)
	}

	return nil
}

// NewEndpoint returns http://localhost:<port>
func NewEndpoint(port string) string {
	return "http://localhost:" + port
}

func rpcLogsToDigest(logs []string) (digest types.Digest, err error) {
	digest = types.NewDigest()

	for _, l := range logs {
		itemBytes, err := common.HexToBytes(l)
		if err != nil {
			return digest, fmt.Errorf("malformed digest item hex string: %w", err)
		}

		di := types.NewDigestItem()
		err = scale.Unmarshal(itemBytes, &di)
		if err != nil {
			return digest, fmt.Errorf("malformed digest item bytes: %w", err)
		}

		digestValue, err := di.Value()
		if err != nil {
			return digest, err
		}
		err = digest.Add(digestValue)
		if err != nil {
			return digest, fmt.Errorf("cannot add digest item to digest: %w", err)
		}
	}

	return digest, nil
}
