// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"math/big"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/pkg/scale"

	"github.com/stretchr/testify/require"
)

// BuildBlock ...
func BuildBlock(t *testing.T, instance runtime.Instance, parent *types.Header, ext types.Extrinsic) *types.Block {
	digest := types.NewDigest()
	prd, err := types.NewBabeSecondaryPlainPreDigest(0, 1).ToPreRuntimeDigest()
	require.NoError(t, err)
	err = digest.Add(*prd)
	require.NoError(t, err)
	header := &types.Header{
		ParentHash: parent.Hash(),
		Number:     big.NewInt(0).Add(parent.Number, big.NewInt(1)),
		Digest:     digest,
	}

	err = instance.InitializeBlock(header)
	require.NoError(t, err)

	idata := types.NewInherentsData()
	err = idata.SetInt64Inherent(types.Timstap0, uint64(time.Now().Unix()))
	require.NoError(t, err)

	err = idata.SetInt64Inherent(types.Babeslot, 1)
	require.NoError(t, err)

	ienc, err := idata.Encode()
	require.NoError(t, err)

	// Call BlockBuilder_inherent_extrinsics which returns the inherents as encoded extrinsics
	inherentExts, err := instance.InherentExtrinsics(ienc)
	require.NoError(t, err)

	// decode inherent extrinsics
	cp := make([]byte, len(inherentExts))
	copy(cp, inherentExts)
	var inExts [][]byte
	err = scale.Unmarshal(cp, &inExts)
	require.NoError(t, err)

	// apply each inherent extrinsic
	for _, inherent := range inExts {
		in, err := scale.Marshal(inherent)
		require.NoError(t, err)

		ret, err := instance.ApplyExtrinsic(in)
		require.NoError(t, err)
		require.Equal(t, ret, []byte{0, 0})
	}

	body := types.Body(types.BytesArrayToExtrinsics(inExts))

	if ext != nil {
		// validate and apply extrinsic
		var ret []byte

		externalExt := types.Extrinsic(append([]byte{byte(types.TxnExternal)}, ext...))
		_, err = instance.ValidateTransaction(externalExt)
		require.NoError(t, err)

		ret, err = instance.ApplyExtrinsic(ext)
		require.NoError(t, err)
		require.Equal(t, ret, []byte{0, 0})

		body = append(body, ext)
	}

	res, err := instance.FinalizeBlock()
	require.NoError(t, err)
	res.Number = header.Number
	res.Hash()

	return &types.Block{
		Header: *res,
		Body:   body,
	}
}
