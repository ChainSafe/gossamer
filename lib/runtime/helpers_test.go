package runtime

import (
	"testing"

	"github.com/ChainSafe/gossamer/lib/common/optional"
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

// validate_transaction only works for transfer extrinsics

func TestApplyExtrinsic_IncludeData(t *testing.T) {
	rt := NewTestRuntime(t, POLKADOT_RUNTIME_c768a7e4c70e)

	ext := extrinsic.NewIncludeDataExt([]byte("nootwashere"))
	enc, err := ext.Encode()
	require.NoError(t, err)

	tx := &transaction.ValidTransaction{
		Extrinsic: enc,
		Validity: &transaction.Validity{
			Priority: 1,
			Requires: [][]byte{},
			Provides: [][]byte{},
			Longevity: 1,
			Propagate: false,
		},
	}

	txb, err := tx.Encode()
	require.NoError(t, err)

	t.Log(enc)
	t.Log(txb)

	validity, err := rt.ApplyExtrinsic(txb)
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

func TestApplyExtrinsic_StorageChange_Set(t *testing.T) {
	rt := NewTestRuntime(t, POLKADOT_RUNTIME_c768a7e4c70e)

	ext := extrinsic.NewStorageChangeExt([]byte("testkey"), optional.NewBytes(true, []byte("testvalue")))
	enc, err := ext.Encode()
	require.NoError(t, err)

	tx := &transaction.ValidTransaction{
		Extrinsic: enc,
		Validity: new(transaction.Validity),
	}

	txb, err := tx.Encode()
	require.NoError(t, err)

	validity, err := rt.ApplyExtrinsic(txb)
	require.Nil(t, err)

	// https://github.com/paritytech/substrate/blob/ea2644a235f4b189c8029b9c9eac9d4df64ee91e/core/test-runtime/src/system.rs#L190
	expected := &transaction.Validity{
		Priority: 0xb,
		Requires: [][]byte{},
		Provides:  [][]byte{{0x6e, 0x6f, 0x6f, 0x74, 0x77, 0x61, 0x73, 0x68, 0x65, 0x72, 0x65}},
		Longevity: 1,
		Propagate: false,
	}

	require.Equal(t, expected, validity)
}

func TestApplyExtrinsic_StorageChange_Delete(t *testing.T) {
	rt := NewTestRuntime(t, POLKADOT_RUNTIME_c768a7e4c70e)

	ext := extrinsic.NewStorageChangeExt([]byte("testkey"), optional.NewBytes(false, []byte{}))
	tx, err := ext.Encode()
	require.NoError(t, err)

	validity, err := rt.ApplyExtrinsic(tx)
	require.Nil(t, err)

	// https://github.com/paritytech/substrate/blob/ea2644a235f4b189c8029b9c9eac9d4df64ee91e/core/test-runtime/src/system.rs#L190
	expected := &transaction.Validity{
		Priority: 0xb,
		Requires: [][]byte{},
		Provides:  [][]byte{{0x6e, 0x6f, 0x6f, 0x74, 0x77, 0x61, 0x73, 0x68, 0x65, 0x72, 0x65}},
		Longevity: 1,
		Propagate: false,
	}

	require.Equal(t, expected, validity)
}
