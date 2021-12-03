// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package rpc

import (
	"fmt"
	"net/http"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/ChainSafe/gossamer/dot/rpc/json2"
	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/gorilla/rpc/v2"
)

// DotUpCodec for overridding default jsonCodec
type DotUpCodec struct{}

// NewDotUpCodec for creating instance of DocUpCodec
func NewDotUpCodec() *DotUpCodec {
	return &DotUpCodec{}
}

// NewRequest is overridden to inject our codec handler
func (c *DotUpCodec) NewRequest(r *http.Request) rpc.CodecRequest {
	outerCR := &DotUpCodecRequest{} // Our custom CR
	jsonC := json2.NewCodec()       // json Codec to create json CR
	innerCR := jsonC.NewRequest(r)  // create the json CR, sort of.

	// NOTE - innerCR is of the interface type rpc.CodecRequest.
	// Because innerCR is of the rpc.CR interface type, we need a
	// type assertion in order to assign it to our struct field's type.
	// We defined the source of the interface implementation here, so
	// we can be confident that innerCR will be of the correct underlying type
	outerCR.CodecRequest = innerCR.(*json2.CodecRequest)
	return outerCR
}

// DotUpCodecRequest decodes and encodes a single request. UpCodecRequest
// implements gorilla/rpc.CodecRequest interface primarily by embedding
// the CodecRequest from gorilla/rpc/json. By selectively adding
// CodecRequest methods to UpCodecRequest, we can modify that behaviour
// while maintaining all the other remaining CodecRequest methods from
// gorilla's rpc/json implementation
type DotUpCodecRequest struct {
	*json2.CodecRequest
}

// Method returns the decoded method as a string of the form "Service.Method"
// after checking for, and correcting a underscore and lowercase method name
// By being of lower depth in the struct , Method will replace the implementation
// of Method() on the embedded CodecRequest. Because the request data is part
// of the embedded json.CodecRequest, and unexported, we have to get the
// requested method name via the embedded CR's own method Method().
// Essentially, this just intercepts the return value from the embedded
// gorilla/rpc/json.CodecRequest.Method(), checks/modifies it, and passes it
// on to the calling rpc server.
func (c *DotUpCodecRequest) Method() (string, error) {
	m, err := c.CodecRequest.Method()
	if len(m) > 1 && err == nil {
		if concreteMethod, ok := modules.AliasesMethods[m]; ok {
			m = concreteMethod
		}

		parts := strings.Split(m, "_")
		if len(parts) < 2 {
			return "", fmt.Errorf("rpc error method %s not found", m)
		}
		service, method := parts[0], parts[1]
		r, n := utf8.DecodeRuneInString(method) // get the first rune, and it's length
		if unicode.IsLower(r) {
			upMethod := service + "." + string(unicode.ToUpper(r)) + method[n:]
			return upMethod, err
		}
	}
	return m, err
}
