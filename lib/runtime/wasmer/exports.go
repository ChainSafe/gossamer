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

package wasmer

import (
	"bytes"
	"fmt"
	"io"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/scale"
	"github.com/ChainSafe/gossamer/lib/transaction"
)

// ValidateTransaction runs the extrinsic through runtime function TaggedTransactionQueue_validate_transaction and returns *Validity
func (in *Instance) ValidateTransaction(e types.Extrinsic) (*transaction.Validity, error) {
	ret, err := in.exec(runtime.TaggedTransactionQueueValidateTransaction, e)
	if err != nil {
		return nil, err
	}

	if ret[0] != 0 {
		return nil, runtime.NewValidateTransactionError(ret)
	}

	v := transaction.NewValidity(0, [][]byte{{}}, [][]byte{{}}, 0, false)
	_, err = scale.Decode(ret[1:], v)

	return v, err
}

// Version calls runtime function Core_Version
func (in *Instance) Version() (runtime.Version, error) {
	// kusama seems to use the legacy version format
	if in.version != nil && bytes.Equal(in.version.SpecName(), []byte("kusama")) {
		return in.version, nil
	}

	version := new(runtime.VersionData)
	res, err := in.exec(runtime.CoreVersion, []byte{})
	if err != nil {
		return nil, err
	}

	err = version.Decode(res)
	if err == io.EOF {
		// kusama seems to use the legacy version format
		lversion := &runtime.LegacyVersionData{}
		err = lversion.Decode(res)
		return lversion, err
	} else if err != nil {
		return nil, err
	}

	return version, nil
}

// Metadata calls runtime function Metadata_metadata
func (in *Instance) Metadata() ([]byte, error) {
	return in.exec(runtime.Metadata, []byte{})
}

// BabeConfiguration gets the configuration data for BABE from the runtime
func (in *Instance) BabeConfiguration() (*types.BabeConfiguration, error) {
	data, err := in.exec(runtime.BabeAPIConfiguration, []byte{})
	if err != nil {
		return nil, err
	}

	bc := new(types.BabeConfiguration)
	_, err = scale.Decode(data, bc)
	if err != nil {
		return nil, err
	}

	return bc, nil
}

// GrandpaAuthorities returns the genesis authorities from the runtime
func (in *Instance) GrandpaAuthorities() ([]*types.Authority, error) {
	ret, err := in.exec(runtime.GrandpaAuthorities, []byte{})
	if err != nil {
		return nil, err
	}

	adr, err := scale.Decode(ret, []*types.GrandpaAuthoritiesRaw{})
	if err != nil {
		return nil, err
	}

	return types.GrandpaAuthoritiesRawToAuthorities(adr.([]*types.GrandpaAuthoritiesRaw))
}

// InitializeBlock calls runtime API function Core_initialise_block
func (in *Instance) InitializeBlock(header *types.Header) error {
	encodedHeader, err := scale.Encode(header)
	if err != nil {
		return fmt.Errorf("cannot encode header: %w", err)
	}

	_, err = in.exec(runtime.CoreInitializeBlock, encodedHeader)
	return err
}

// InherentExtrinsics calls runtime API function BlockBuilder_inherent_extrinsics
func (in *Instance) InherentExtrinsics(data []byte) ([]byte, error) {
	return in.exec(runtime.BlockBuilderInherentExtrinsics, data)
}

// ApplyExtrinsic calls runtime API function BlockBuilder_apply_extrinsic
func (in *Instance) ApplyExtrinsic(data types.Extrinsic) ([]byte, error) {
	return in.exec(runtime.BlockBuilderApplyExtrinsic, data)
}

//nolint
// FinalizeBlock calls runtime API function BlockBuilder_finalize_block
func (in *Instance) FinalizeBlock() (*types.Header, error) {
	data, err := in.exec(runtime.BlockBuilderFinalizeBlock, []byte{})
	if err != nil {
		return nil, err
	}

	bh := new(types.Header)
	_, err = scale.Decode(data, bh)
	if err != nil {
		return nil, err
	}

	return bh, nil
}

// ExecuteBlock calls runtime function Core_execute_block
func (in *Instance) ExecuteBlock(block *types.Block) ([]byte, error) {
	// copy block since we're going to modify it
	b := block.DeepCopy()

	if in.version == nil {
		var err error
		in.version, err = in.Version()
		if err != nil {
			return nil, err
		}
	}

	// remove seal digest only
	b.Header.Digest = types.NewEmptyDigest()
	for _, d := range block.Header.Digest {
		if d.Type() == types.SealDigestType {
			continue
		}

		b.Header.Digest = append(b.Header.Digest, d)
	}

	bdEnc, err := b.Encode()
	if err != nil {
		return nil, err
	}

	return in.exec(runtime.CoreExecuteBlock, bdEnc)
}

// DecodeSessionKeys decodes the given public session keys. Returns a list of raw public keys including their key type.
func (in *Instance) DecodeSessionKeys(enc []byte) ([]byte, error) {
	return in.exec(runtime.DecodeSessionKeys, enc)
}

func (in *Instance) CheckInherents()      {} //nolint
func (in *Instance) RandomSeed()          {} //nolint
func (in *Instance) OffchainWorker()      {} //nolint
func (in *Instance) GenerateSessionKeys() {} //nolint
