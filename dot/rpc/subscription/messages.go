// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package subscription

// BaseResponseJSON for base json response
type BaseResponseJSON struct {
	Jsonrpc string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  Params `json:"params"`
}

// Params for json param response
type Params struct {
	Result         interface{} `json:"result"`
	SubscriptionID uint32      `json:"subscription"`
}

// InvalidRequestCode error code returned for invalid request parameters, value derived from Substrate node output
const InvalidRequestCode = -32600

// InvalidRequestMessage error message for invalid request parameters
const InvalidRequestMessage = "Invalid request"

func newSubcriptionBaseResponseJSON() BaseResponseJSON {
	return BaseResponseJSON{
		Jsonrpc: "2.0",
	}
}

func newSubscriptionResponse(method string, subID uint32, result interface{}) BaseResponseJSON {
	return BaseResponseJSON{
		Jsonrpc: "2.0",
		Method:  method,
		Params: Params{
			Result:         result,
			SubscriptionID: subID,
		},
	}
}

// ResponseJSON for json subscription responses
type ResponseJSON struct {
	Jsonrpc string  `json:"jsonrpc"`
	Result  uint32  `json:"result"`
	ID      float64 `json:"id"`
}

// NewSubscriptionResponseJSON builds a Response JSON object
func NewSubscriptionResponseJSON(subID uint32, reqID float64) ResponseJSON {
	return ResponseJSON{
		Jsonrpc: "2.0",
		Result:  subID,
		ID:      reqID,
	}
}

// BooleanResponse for responses that return boolean values
type BooleanResponse struct {
	JSONRPC string  `json:"jsonrpc"`
	Result  bool    `json:"result"`
	ID      float64 `json:"id"`
}

func newBooleanResponseJSON(value bool, reqID float64) BooleanResponse {
	return BooleanResponse{
		JSONRPC: "2.0",
		Result:  value,
		ID:      reqID,
	}
}
