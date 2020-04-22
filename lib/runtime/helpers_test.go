package runtime

import (
	"testing"

	"github.com/ChainSafe/gossamer/lib/runtime/extrinsic"
	"github.com/ChainSafe/gossamer/lib/transaction"

	"github.com/stretchr/testify/require"
)

func TestValidateTransaction_IncludeData(t *testing.T) {
	rt := NewTestRuntime(t, POLKADOT_RUNTIME_c768a7e4c70e)

	ext := extrinsic.NewIncludeDataExt([]byte("nootwashere"))
	tx, err := ext.Encode()
	require.NoError(t, err)

	validity, err := rt.ValidateTransaction(tx)
	require.Nil(t, err)

	// https://github.com/paritytech/substrate/blob/ea2644a235f4b189c8029b9c9eac9d4df64ee91e/core/test-runtime/src/system.rs#L190
	expected := &transaction.Validity{
		Priority:  0xb,
		Requires:  [][]byte{},
		Provides:  [][]byte{{0x6e, 0x6f, 0x6f, 0x74, 0x77, 0x61, 0x73, 0x68, 0x65, 0x72, 0x65}},
		Longevity: 1,
		Propagate: false,
	}

	require.Equal(t, expected, validity)
}
