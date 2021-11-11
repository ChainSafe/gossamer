package utils

import (
	"encoding/json"
	"net"
	"net/http"
	"os"
	"time"
)

//nolint
var (
	MODE = os.Getenv("MODE")

	HOSTNAME = os.Getenv("HOSTNAME")
	PORT     = os.Getenv("PORT")

	LOGLEVEL = os.Getenv("LOG")

	NETWORK_SIZE = os.Getenv("NETWORK_SIZE")

	ContentTypeJSON   = "application/json"
	dialTimeout       = 60 * time.Second
	httpClientTimeout = 120 * time.Second

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

// ErrCode is a int type used for the rpc error codes
type ErrCode int

// Error is a struct that holds the error message and the error code for a error
type Error struct {
	Message   string                 `json:"message"`
	ErrorCode ErrCode                `json:"code"`
	Data      map[string]interface{} `json:"data"`
}

// Error returns the error Message string
func (e *Error) Error() string {
	return e.Message
}
