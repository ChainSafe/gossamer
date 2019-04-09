package json2

import (
	"encoding/json"
	"errors"
	"io"
	"math/rand"
)

type clientRequest struct {
	Version string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	Id      uint64      `json:"id"`
}

type clientResponse struct {
	Version string           `json:"jsonrpc"`
	Result  *json.RawMessage `json:"result"`
	Error   *json.RawMessage `json:"error"`
}

func EncodeClientRequest(method string, args interface{}) ([]byte, error) {
	c := &clientRequest{
		Version: JSONVersion,
		Method:  method,
		Params:  args,
		Id:      uint64(rand.Int63()),
	}
	return json.Marshal(c)
}

func DecodeClientResponse(r io.Reader, reply interface{}) error {
	var c clientResponse
	if err := json.NewDecoder(r).Decode(&c); err != nil {
		return err
	}
	if c.Error != nil {
		jsonErr := &Error{}
		if err := json.Unmarshal(*c.Error, jsonErr); err != nil {
			return &Error{
				ErrorCode: ERR_INTERNAL_ERROR,
				Message: string(*c.Error),
			}
		}
		return jsonErr
	}

	if c.Result == nil {
		return errors.New("result cannot be null")
	}

	return json.Unmarshal(*c.Result, reply)
}
