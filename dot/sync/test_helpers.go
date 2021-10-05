// Copyright 2019 ChainSafe Systems (ON) Corp.
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

package sync

import (
	"math/big"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/babe"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/transaction"
	"github.com/ChainSafe/gossamer/pkg/scale"

	"github.com/stretchr/testify/require"
)

// BuildBlock ...
func BuildBlock(t *testing.T, instance runtime.Instance, parent *types.Header, ext types.Extrinsic) *types.Block {
	digest := types.NewDigest()
	err := digest.Add(*types.NewBabeSecondaryPlainPreDigest(0, 1).ToPreRuntimeDigest())
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

	// Call BlockBuilder_inherent_extrinsics which returns the inherents as extrinsics
	inherentExts, err := instance.InherentExtrinsics(ienc)
	require.NoError(t, err)

	// decode inherent extrinsics
	var exts [][]byte
	err = scale.Unmarshal(inherentExts, &exts)
	require.NoError(t, err)

	inExt := exts

	var body *types.Body
	if ext != nil {
		var txn *transaction.Validity
		externalExt := types.Extrinsic(append([]byte{byte(types.TxnExternal)}, ext...))
		txn, err = instance.ValidateTransaction(externalExt)
		require.NoError(t, err)

		vtx := transaction.NewValidTransaction(ext, txn)
		_, err = instance.ApplyExtrinsic(ext) // TODO: Determine error for ret
		require.NoError(t, err)

		body, err = babe.ExtrinsicsToBody(inExt, []*transaction.ValidTransaction{vtx})
		require.NoError(t, err)

	} else {
		body = types.NewBody(types.BytesArrayToExtrinsics(exts))
	}

	// apply each inherent extrinsic
	for _, ext := range inExt {
		in, err := scale.Marshal(ext) //nolint
		require.NoError(t, err)

		ret, err := instance.ApplyExtrinsic(in)
		require.NoError(t, err)
		require.Equal(t, ret, []byte{0, 0})
	}

	res, err := instance.FinalizeBlock()
	require.NoError(t, err)
	res.Number = header.Number

	return &types.Block{
		Header: *res,
		Body:   *body,
	}
}
