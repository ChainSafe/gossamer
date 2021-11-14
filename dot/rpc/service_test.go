// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package rpc

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/gorilla/rpc/v2"
	"github.com/stretchr/testify/require"
)

func TestNewService(t *testing.T) {
	NewService()
}

func TestService_Methods(t *testing.T) {
	qtySystemMethods := 15
	qtyRPCMethods := 1
	qtyAuthorMethods := 8

	rpcService := NewService()
	sysMod := modules.NewSystemModule(nil, nil, nil, nil, nil, nil)
	rpcService.BuildMethodNames(sysMod, "system")
	m := rpcService.Methods()
	require.Equal(t, qtySystemMethods, len(m)) // check to confirm quantity for methods is correct

	rpcMod := modules.NewRPCModule(nil)
	rpcService.BuildMethodNames(rpcMod, "rpc")
	m = rpcService.Methods()
	require.Equal(t, qtySystemMethods+qtyRPCMethods, len(m))

	authMod := modules.NewAuthorModule(log.New(log.SetWriter(io.Discard)), nil, nil)
	rpcService.BuildMethodNames(authMod, "author")
	m = rpcService.Methods()
	require.Equal(t, qtySystemMethods+qtyRPCMethods+qtyAuthorMethods, len(m))
}

type mockService struct{}

// MockServiceArrayRequest must be exported for ReadArray or tests will fail.
type MockServiceArrayRequest struct {
	Key   []string
	Bhash []common.Hash
}

// MockServiceArrayResponse must be exported for ReadArray or tests will fail.
type MockServiceArrayResponse struct {
	Key   []string
	Bhash []common.Hash
}

func (t *mockService) ReadArray(r *http.Request, req *MockServiceArrayRequest, res *MockServiceArrayResponse) error {
	res.Key = req.Key
	res.Bhash = req.Bhash
	return nil
}

type mockResponseWriter struct {
	header http.Header
	Status int
	Body   string
}

func newMockResponseWriter() *mockResponseWriter {
	header := make(http.Header)
	return &mockResponseWriter{header: header}
}

func (w *mockResponseWriter) Header() http.Header {
	return w.header
}

func (w *mockResponseWriter) Write(p []byte) (int, error) {
	w.Body = string(p)
	if w.Status == 0 {
		w.Status = 200
	}
	return len(p), nil
}

func (w *mockResponseWriter) WriteHeader(status int) {
	w.Status = status
}

func TestJson2ReadRequest(t *testing.T) {
	s := rpc.NewServer()
	s.RegisterService(new(mockService), "mockService")
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

			resWriter := newMockResponseWriter()
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
