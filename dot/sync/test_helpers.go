// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/babe"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/transaction"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
)

// BuildBlockRuntime is the runtime interface to interact with
// blocks and extrinsics.
type BuildBlockRuntime interface {
	BabeConfiguration() (*types.BabeConfiguration, error)
	InitializeBlock(header *types.Header) error
	FinalizeBlock() (*types.Header, error)
	InherentExtrinsics(data []byte) ([]byte, error)
	ApplyExtrinsic(data types.Extrinsic) ([]byte, error)
	ValidateTransaction(e types.Extrinsic) (*transaction.Validity, error)
}

// BuildBlock ...
func BuildBlock(t *testing.T, instance BuildBlockRuntime, parent *types.Header, ext types.Extrinsic) *types.Block {
	babeCfg, err := instance.BabeConfiguration()
	require.NoError(t, err)

	timestamp := uint64(time.Now().Unix())
	slotDuration := babeCfg.SlotDuration
	currentSlot := timestamp / slotDuration

	digest := types.NewDigest()
	prd, err := types.NewBabeSecondaryPlainPreDigest(0, currentSlot).ToPreRuntimeDigest()
	require.NoError(t, err)
	err = digest.Add(*prd)
	require.NoError(t, err)
	header := &types.Header{
		ParentHash:     parent.Hash(),
		StateRoot:      common.Hash{},
		ExtrinsicsRoot: common.Hash{},
		Number:         parent.Number + 1,
		Digest:         digest,
	}

	err = instance.InitializeBlock(header)
	require.NoError(t, err)

	idata := types.NewInherentData()
	err = idata.SetInherent(types.Timstap0, timestamp)
	require.NoError(t, err)

	err = idata.SetInherent(types.Babeslot, currentSlot)
	require.NoError(t, err)

	parachainInherent := babe.ParachainInherentData{
		ParentHeader: types.Header{
			ParentHash:     parent.Hash(),
			Number:         parent.Number,
			StateRoot:      parent.StateRoot,
			ExtrinsicsRoot: parent.ExtrinsicsRoot,
		},
	}

	err = idata.SetInherent(types.Parachn0, parachainInherent)
	require.NoError(t, err)

	err = idata.SetInherent(types.Newheads, []byte{0})
	require.NoError(t, err)

	ienc, err := idata.Encode()
	require.NoError(t, err)

	// Call BlockBuilder_inherent_extrinsics which returns the inherents as encoded extrinsics
	inherentExts, err := instance.InherentExtrinsics(ienc)
	require.NoError(t, err)

	// decode inherent extrinsics
	var inExts [][]byte
	err = scale.Unmarshal(inherentExts, &inExts)
	require.NoError(t, err)

	// apply each inherent extrinsic
	for _, inherent := range inExts {
		in, err := scale.Marshal(inherent)
		require.NoError(t, err)

		ret, err := instance.ApplyExtrinsic(in)
		require.NoError(t, err)
		require.Equal(t, ret, []byte{0, 0})
	}

	var exts []types.Extrinsic
	if ext != nil {
		// validate and apply extrinsic
		var ret []byte

		externalExt := types.Extrinsic(append([]byte{byte(types.TxnExternal)}, ext...))
		_, err = instance.ValidateTransaction(externalExt)
		require.NoError(t, err)

		ret, err = instance.ApplyExtrinsic(ext)
		require.NoError(t, err)
		require.Equal(t, ret, []byte{0, 0})

		exts = append(exts, ext)
	}

	res, err := instance.FinalizeBlock()
	require.NoError(t, err)

	body := types.Body(types.BytesArrayToExtrinsics(inExts))
	body = append(body, exts...)

	res.Number = header.Number
	res.Hash()

	return &types.Block{
		Header: *res,
		Body:   body,
	}
}
