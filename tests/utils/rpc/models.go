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

// WebsocketResponse wraps the Websocket response
type WebsocketResponse struct {
	// JSON-RPC Version
	Version string `json:"jsonrpc"`
	// Method name called
	Method string `json:"method"`
	// Resulting values
	Result json.RawMessage `json:"result"`
	// Params values including results
	Params json.RawMessage `json:"params"`
	// Any generated errors
	Error *Error `json:"error"`
	// Request id
	Subscription *json.RawMessage `json:"subscription"`
	// Request id
	ID *json.RawMessage `json:"id"`
}

// Error is a struct that holds the error message and the error code for a error
type Error struct {
	Message   string                 `json:"message"`
	ErrorCode int                    `json:"code"`
	Data      map[string]interface{} `json:"data"`
}
