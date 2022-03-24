// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package websocket

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

var (
	ErrResponseVersion = errors.New("unexpected response version received")
	ErrResponseError   = errors.New("response error received")
)

// Decode will decode body into target interface.
func Decode(body []byte, target interface{}) error {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()

	var response Response
	err := decoder.Decode(&response)
	if err != nil {
		return fmt.Errorf("cannot decode websocket response: %w", err)
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
		return fmt.Errorf("cannot decode result or params of websocket response: %w", err)
	}

	return nil
}
