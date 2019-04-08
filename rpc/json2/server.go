package json2

import (
	"encoding/json"
	"github.com/ChainSafe/gossamer/rpc"
	"net/http"
	"strings"
)

var JSONVersion = "2.0"

type serverRequest struct {
	Version string `json:"jsonrpc"`
	Method string `json:"method"`
	Params *json.RawMessage `json:"params"`
	Id *json.RawMessage `json:"id"`
}


type serverResponse struct {
	Version string `json:"jsonrpc"`
	Result interface{} `json:"result"`
	Error *Error `json:"error"`
	Id *json.RawMessage `json:"id"`
}


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
		// create error struct
	} else if req.Version != JSONVersion {
		// create error struct
	}

	r.Body.Close()
	return &CodecRequest{request: req, err: err}
}

type CodecRequest struct {
	request *serverRequest
	err error
	encoder rpc.Encoder
	errorMapper func(error) error
}

func(c *CodecRequest) Method() string {
	return strings.Title(strings.Replace(c.request.Method, "_", ".", 1))
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
	encoder := json.NewEncoder(c.encoder.Encode(w))
	encoder.Encode(res)
}

type EmptyResponse struct {}

