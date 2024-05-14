package networkbridge

import (
	"bytes"
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
)

func TestWireMessage(t *testing.T) {
	// TODO: Add rust code to encode this
	expectedEncoding := []byte{} // fill this.
	wireMessage := NewWireMessage()
	viewUpdate := ViewUpdate(View{
		heads:           []common.Hash{},
		finalizedNumber: 0,
	})
	wireMessage.Set(viewUpdate)

	actualEncoding, err := scale.Marshal(wireMessage)
	require.NoError(t, err)

	require.Equal(t, bytes.Compare(actualEncoding, expectedEncoding), 0)
}
