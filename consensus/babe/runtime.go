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

package babe

import (
	"errors"

	scale "github.com/ChainSafe/gossamer/codec"
	tx "github.com/ChainSafe/gossamer/common/transaction"
	"github.com/ChainSafe/gossamer/core/types"
	"github.com/ChainSafe/gossamer/runtime"
)

// gets the configuration data for Babe from the runtime
func (b *Session) configurationFromRuntime() error {
	ret, err := b.rt.Exec(runtime.BabeApiConfiguration, 1, []byte{})
	if err != nil {
		return err
	}

	bc := new(BabeConfiguration)
	_, err = scale.Decode(ret, bc)
	if err != nil {
		return err
	}

	// Directly set the babe session's config
	b.config = bc

	return err
}

// calls runtime API function Core_initialize_block
func (b *Session) initializeBlock(blockHeader []byte) error {
	ptr, err := b.rt.Malloc(uint32(len(blockHeader)))
	if err != nil {
		return err
	}

	b.rt.Store(blockHeader, int32(ptr))

	_, err = b.rt.Exec(runtime.CoreInitializeBlock, int32(ptr), blockHeader)
	if err != nil {
		return err
	}

	return b.rt.Free(ptr)
}

// calls runtime API function BlockBuilder_inherent_extrinsics
func (b *Session) inherentExtrinsics(data []byte) ([]byte, error) {
	ptr, err := b.rt.Malloc(uint32(len(data)))
	if err != nil {
		return nil, err
	}

	b.rt.Store(data, int32(ptr))

	ret, err := b.rt.Exec(runtime.BlockBuilderInherentExtrinsics, int32(ptr), data)
	if err != nil {
		return nil, err
	}

	err = b.rt.Free(ptr)
	return ret, err
}

// calls runtime API function BlockBuilder_apply_extrinsic
func (b *Session) applyExtrinsic(data types.Extrinsic) ([]byte, error) {
	ptr, err := b.rt.Malloc(uint32(len(data)))
	if err != nil {
		return nil, err
	}

	b.rt.Store(data, int32(ptr))

	ret, err := b.rt.Exec(runtime.BlockBuilderApplyExtrinsic, int32(ptr), data)
	if err != nil {
		return nil, err
	}

	err = b.rt.Free(ptr)
	return ret, err
}

// calls runtime API function BlockBuilder_finalize_block
func (b *Session) finalizeBlock() (*types.Block, error) {
	ret, err := b.rt.Exec(runtime.BlockBuilderFinalizeBlock, 0, []byte{})
	if err != nil {
		return nil, err
	}

	bh := &types.Block{
		Header: new(types.BlockHeader),
		Body:   new(types.BlockBody),
	}

	_, err = scale.Decode(ret, bh)
	return bh, err
}

// calls runtime API function TaggedTransactionQueue_validate_transaction
func (b *Session) validateTransaction(data types.Extrinsic) (*tx.Validity, error) {
	ptr, err := b.rt.Malloc(uint32(len(data)))
	if err != nil {
		return nil, err
	}

	b.rt.Store(data, int32(ptr))

	ret, err := b.rt.Exec(runtime.TaggedTransactionQueueValidateTransaction, int32(ptr), data)
	if err != nil {
		return nil, err
	}

	if ret[0] != 0 {
		return nil, errors.New("could not validate transaction")
	}

	v := tx.NewValidity(0, [][]byte{{}}, [][]byte{{}}, 0, false)
	_, err = scale.Decode(ret[1:], v)
	if err != nil {
		return nil, err
	}

	err = b.rt.Free(ptr)
	return v, err
}
