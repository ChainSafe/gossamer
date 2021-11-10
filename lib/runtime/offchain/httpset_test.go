package offchain

import (
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
	set.reqs[1] = &http.Request{}

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

	require.Equal(t, http.MethodGet, req.Method)
	require.Equal(t, defaultTestURI, req.URL.String())
}
