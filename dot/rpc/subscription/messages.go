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
package subscription

import (
	"math/big"
)

// BaseResponseJSON for base json response
type BaseResponseJSON struct {
	Jsonrpc string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  Params `json:"params"`
}

// Params for json param response
type Params struct {
	Result         interface{} `json:"result"`
	SubscriptionID int         `json:"subscription"`
}

func newSubcriptionBaseResponseJSON() BaseResponseJSON {
	return BaseResponseJSON{
		Jsonrpc: "2.0",
	}
}

func newSubscriptionResponse(method string, subID int, result interface{}) BaseResponseJSON {
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
	Result  int     `json:"result"`
	ID      float64 `json:"id"`
}

func newSubscriptionResponseJSON(subID int, reqID float64) ResponseJSON {
	return ResponseJSON{
		Jsonrpc: "2.0",
		Result:  subID,
		ID:      reqID,
	}
}

// ErrorResponseJSON json for error responses
type ErrorResponseJSON struct {
	Jsonrpc string            `json:"jsonrpc"`
	Error   *ErrorMessageJSON `json:"error"`
	ID      float64           `json:"id"`
}

// ErrorMessageJSON json for error messages
type ErrorMessageJSON struct {
	Code    *big.Int `json:"code"`
	Message string   `json:"message"`
}

func (c *WSConn) safeSend(msg interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	err := c.Wsconn.WriteJSON(msg)
	if err != nil {
		logger.Debug("error sending websocket message", "error", err)
	}
}
func (c *WSConn) safeSendError(reqID float64, errorCode *big.Int, message string) {
	res := &ErrorResponseJSON{
		Jsonrpc: "2.0",
		Error: &ErrorMessageJSON{
			Code:    errorCode,
			Message: message,
		},
		ID: reqID,
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	err := c.Wsconn.WriteJSON(res)
	if err != nil {
		logger.Debug("error sending websocket message", "error", err)
	}
}
