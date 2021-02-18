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
package rpc

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/gorilla/rpc/v2"
	"github.com/stretchr/testify/require"
)

func TestNewService(t *testing.T) {
	NewService()
}

func TestService_Methods(t *testing.T) {
	qtySystemMethods := 10
	qtyRPCMethods := 1
	qtyAuthorMethods := 7

	rpcService := NewService()
	sysMod := modules.NewSystemModule(nil, nil, nil, nil, nil)
	rpcService.BuildMethodNames(sysMod, "system")
	m := rpcService.Methods()
	require.Equal(t, qtySystemMethods, len(m)) // check to confirm quantity for methods is correct

	rpcMod := modules.NewRPCModule(nil)
	rpcService.BuildMethodNames(rpcMod, "rpc")
	m = rpcService.Methods()
	require.Equal(t, qtySystemMethods+qtyRPCMethods, len(m))

	authMod := modules.NewAuthorModule(nil, nil, nil, nil)
	rpcService.BuildMethodNames(authMod, "author")
	m = rpcService.Methods()
	require.Equal(t, qtySystemMethods+qtyRPCMethods+qtyAuthorMethods, len(m))
}

type MockService struct{}

type MockServiceArrayRequest struct {
	Key   []string
	Bhash []common.Hash
}

type MockServiceArrayResponse struct {
	Key   []string
	Bhash []common.Hash
}

func (t *MockService) ReadArray(r *http.Request, req *MockServiceArrayRequest, res *MockServiceArrayResponse) error {
	res.Key = req.Key
	res.Bhash = req.Bhash
	return nil
}

type MockResponseWriter struct {
	header http.Header
	Status int
	Body   string
}

func NewMockResponseWriter() *MockResponseWriter {
	header := make(http.Header)
	return &MockResponseWriter{header: header}
}

func (w *MockResponseWriter) Header() http.Header {
	return w.header
}

func (w *MockResponseWriter) Write(p []byte) (int, error) {
	w.Body = string(p)
	if w.Status == 0 {
		w.Status = 200
	}
	return len(p), nil
}

func (w *MockResponseWriter) WriteHeader(status int) {
	w.Status = status
}

func TestJson2ReadRequest(t *testing.T) {
	s := rpc.NewServer()
	s.RegisterService(new(MockService), "mockService")
	s.RegisterCodec(NewDotUpCodec(), "application/json")

	testCase := []struct {
		method string
		params string
		isErr  bool
	}{
		{method: `"mockService_readArray"`, params: `[["key1","key2"], ["0x2ea162e982746e9ea1b3133a4e2d5b586740700b239c14d78209ebf96d3b29d4","0x2ea162e982746e9ea1b3133a4e2d5b586740700b239c14d78209ebf96d3b2977"]]`},
		{method: `"mockService_readArray"`, params: `["key1", ["0x2ea162e982746e9ea1b3133a4e2d5b586740700b239c14d78209ebf96d3b29d4"]]`, isErr: true},
	}

	for _, test := range testCase {
		t.Run(test.method, func(t *testing.T) {

			reader := strings.NewReader(fmt.Sprintf(`{"id":1, "jsonrpc":"2.0", "method":%s, "params":%s}`, test.method, test.params))
			req, err := http.NewRequest("POST", "", reader)
			require.NoError(t, err)

			resWriter := NewMockResponseWriter()
			req.Header.Set("Content-Type", "application/json")
			s.ServeHTTP(resWriter, req)

			if test.isErr {
				require.Contains(t, resWriter.Body, "cannot unmarshal")
				return
			}

			var resp MockServiceArrayResponse
			var resMap map[string]json.RawMessage

			err = json.Unmarshal([]byte(resWriter.Body), &resMap)
			require.NoError(t, err)

			result := resMap["result"]
			err = json.Unmarshal(result, &resp)
			require.NoError(t, err)

			require.Equal(t, len(resp.Key), 2)
			require.Equal(t, len(resp.Key), 2)
		})
	}
}
