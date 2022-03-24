// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package rpc

import "encoding/json"

// ServerResponse wraps the RPC response
type ServerResponse struct {
	// JSON-RPC Version
	Version string `json:"jsonrpc"`
	// Resulting values
	Result json.RawMessage `json:"result"`
	// Any generated errors
	Error *Error `json:"error"`
	// Request id
	ID *json.RawMessage `json:"id"`
}

// Error is a struct that holds the error message and the error code for a error
type Error struct {
	Message   string                 `json:"message"`
	ErrorCode int                    `json:"code"`
	Data      map[string]interface{} `json:"data"`
}
