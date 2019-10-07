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
	scale "github.com/ChainSafe/gossamer/codec"
	"github.com/ChainSafe/gossamer/common"
	"github.com/ChainSafe/gossamer/core/types"
)

// gets the configuration data for Babe from the runtime
func (b *Session) configurationFromRuntime() error {
	ret, err := b.rt.Exec(runtime.BabeApiConfiguration, 1, []byte{})
	if err != nil {
		return err
	}

	bc := new(BabeConfiguration)
	bc.GenesisAuthorities = []AuthorityData{}
	_, err = scale.Decode(ret, bc)

	if err != nil {
		return err
	}

	// Directly set the babe session's config
	b.config = bc

	return err
}

// gets the configuration data for Babe from the runtime
func (b *Session) blockHashFromIdFromRuntime(blockId []byte) (*common.Hash, error) {
	var loc int32 = 1000
	b.rt.Store(blockId, loc)

	ret, err := b.rt.Exec("block_hash_from_id", loc, int32(len(blockId)))
	if err != nil {
		return nil, err
	}

	bc := new(common.Hash)
	_, err = scale.Decode(ret, bc)
	return bc, err
}

// gets the configuration data for Babe from the runtime
func (b *Session) initializeBlockFromRuntime(blockHeader []byte) error {
	var loc int32 = 1000
	b.rt.Store(blockHeader, loc)

	_, err := b.rt.Exec("initialze_block", loc, int32(len(blockHeader)))
	if err != nil {
		return err
	}
	return nil
}

// gets the configuration data for Babe from the runtime
func (b *Session) inherentExtrinsicsFromRuntime(blockInherentData []byte) (*[]types.Extrinsic, error) {
	var loc int32 = 1000
	b.rt.Store(blockInherentData, loc)

	ret, err := b.rt.Exec("inherent_extrinsics", loc, int32(len(blockInherentData)))
	if err != nil {
		return nil, err
	}

	ea := new([]types.Extrinsic)
	_, err = scale.Decode(ret, ea)
	return ea, nil
}

// TODO: Figure out return type of apply_extrinsic
func (b *Session) applyExtrinsicFromRuntime(e types.Extrinsic) (*types.BlockBody, error) {
	var loc int32 = 1000
	b.rt.Store(e, loc)

	ret, err := b.rt.Exec("apply_extrinsics", loc, int32(len(e)))
	if err != nil {
		return nil, err
	}

	bb := new(types.BlockBody)
	_, err = scale.Decode(ret, bb)
	return bb, err
}

// gets the configuration data for Babe from the runtime
func (b *Session) finalizeBlockFromRuntime(e types.Extrinsic) (*types.BlockHeader, error) {
	var loc int32 = 1000
	b.rt.Store(e, loc)

	ret, err := b.rt.Exec("finalize_block", loc, int32(len(e)))
	if err != nil {
		return nil, err
	}

	bh := new(types.BlockHeader)
	_, err = scale.Decode(ret, bh)
	return bh, err
}
