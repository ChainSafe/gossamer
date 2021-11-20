// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package offchain

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

const defaultTestURI = "http://example.url"

func TestHTTPSetLimit(t *testing.T) {
	t.Parallel()

	set := NewHTTPSet()
	var err error
	for i := 0; i < maxConcurrentRequests+1; i++ {
		_, err = set.StartRequest(http.MethodGet, defaultTestURI)
	}

	require.ErrorIs(t, errIntBufferEmpty, err)
}

func TestHTTPSet_StartRequest_NotAvailableID(t *testing.T) {
	t.Parallel()

	set := NewHTTPSet()
	set.reqs[1] = &Request{}

	_, err := set.StartRequest(http.MethodGet, defaultTestURI)
	require.ErrorIs(t, errRequestIDNotAvailable, err)
}

func TestHTTPSetGet(t *testing.T) {
	t.Parallel()

	set := NewHTTPSet()

	id, err := set.StartRequest(http.MethodGet, defaultTestURI)
	require.NoError(t, err)

	req := set.Get(id)
	require.NotNil(t, req)

	require.Equal(t, http.MethodGet, req.Request.Method)
	require.Equal(t, defaultTestURI, req.Request.URL.String())
}

func TestOffchainRequest_AddHeader(t *testing.T) {
	t.Parallel()

	invalidCtx := context.WithValue(context.Background(), invalidKey, true)
	invalidReq, err := http.NewRequestWithContext(invalidCtx, http.MethodGet, "http://test.com", nil)
	require.NoError(t, err)

	cases := map[string]struct {
		offReq           Request
		err              error
		headerK, headerV string
	}{
		"should return invalid request": {
			offReq: Request{invalidReq},
			err:    errRequestInvalid,
		},
		"should add header": {
			offReq:  Request{Request: &http.Request{Header: make(http.Header)}},
			headerK: "key",
			headerV: "value",
		},
		"should return invalid empty header": {
			offReq:  Request{Request: &http.Request{Header: make(http.Header)}},
			headerK: "",
			headerV: "value",
			err:     fmt.Errorf("%w: %s", errInvalidHeaderKey, "empty header key"),
		},
	}

	for name, tc := range cases {
		tc := tc

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := tc.offReq.AddHeader(tc.headerK, tc.headerV)

			if tc.err != nil {
				require.Error(t, err)
				require.Equal(t, tc.err.Error(), err.Error())
				return
			}

			require.NoError(t, err)

			got := tc.offReq.Request.Header.Get(tc.headerK)
			require.Equal(t, tc.headerV, got)
		})
	}
}
