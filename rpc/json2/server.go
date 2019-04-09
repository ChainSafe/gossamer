package json2

import (
	"encoding/json"
	"github.com/ChainSafe/gossamer/rpc"
	"log"
	"net/http"
	"strings"
)

var JSONVersion = "2.0"

type serverRequest struct {
	// JSON Version
	Version string `json:"jsonrpc"`
	// Method name
	Method string `json:"method"`
	// Method params
	Params *json.RawMessage `json:"params"`
	// Request id, may be int or string
	Id *json.RawMessage `json:"id"`
}


type serverResponse struct {
	// JSON Version
	Version string `json:"jsonrpc"`
	// TODO: Comment
	Result interface{} `json:"result"`
	// Any generated errors
	Error *Error `json:"error"`
	// Request id
	Id *json.RawMessage `json:"id"`
}

// NewCodec creates a Codec instance
func NewCodec() *Codec {
	return &Codec{}
}


type Codec struct {
	// TODO: Is this needed? What is its purpose?
	//errorMapper func(error) error
}

func (c *Codec) NewRequest(r *http.Request) rpc.CodecRequest {
	req := new(serverRequest)
	err := json.NewDecoder(r.Body).Decode(req)
	if err != nil {
		// TODO: create error struct
		err = &Error{
			ErrorCode: ERR_PARSE,
			Message: err.Error(),
		}
	} else if req.Version != JSONVersion {
		// TODO: create error struct
		err = &Error{
			ErrorCode: ERR_PARSE,
			Message: "must be JSON-RPC version " + JSONVersion,
		}
	}

	r.Body.Close()
	return &CodecRequest{request: req, err: err}
}

type CodecRequest struct {
	request *serverRequest
	err error
	errorMapper func(error) error
}

func(c *CodecRequest) Method() (string, error) {
	log.Printf("got: %s modified: %s", c.request.Method, strings.Title(strings.Replace(c.request.Method, "_", ".", 1)))
	if c.err == nil {
		return strings.Title(strings.Replace(c.request.Method, "_", ".", 1)), nil

	}
	return "", c.err
}

// ReadRequest parses the handler params
func (c *CodecRequest) ReadRequest(args interface{}) error {
	// TODO: Check if params is nil?
	if c.err == nil {
		err := json.Unmarshal(*c.request.Params, args)
		if err != nil {
			c.err = &Error{
				Message: err.Error(),
				ErrorCode: ERR_PARSE,
			}
		}
	}
	return c.err
}

func (c *CodecRequest) WriteResponse(w http.ResponseWriter, reply interface{}) {
	res := &serverResponse{
		Version: JSONVersion,
		Result:  reply,
		Id:      c.request.Id,
	}
	c.writeServerResponse(w, res)
}

func (c *CodecRequest) WriteError(w http.ResponseWriter, status int, err error){
	jsonErr, ok := err.(*Error)
	if !ok {
		jsonErr = &Error{
			ErrorCode:    ERR_INTERNAL_ERROR,
			Message: err.Error(),
		}
	}
	res := &serverResponse{
		Version: JSONVersion,
		Error:   jsonErr,
		Id:      c.request.Id,
	}
	c.writeServerResponse(w, res)
}

func (c *CodecRequest) writeServerResponse(w http.ResponseWriter, res *serverResponse) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	encoder := json.NewEncoder(w)
	encoder.Encode(res)
}

type EmptyResponse struct {}

