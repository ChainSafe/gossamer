// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package wasmer

import (
	"fmt"
	"github.com/ChainSafe/gossamer/lib/transaction"
	"strings"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/runtime"
	txnvalidity "github.com/ChainSafe/gossamer/lib/runtime/transaction_validity"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// ValidateTransaction runs the extrinsic through the runtime function
// TaggedTransactionQueue_validate_transaction and returns *Validity
func (in *Instance) ValidateTransaction(e types.Extrinsic) (
	*transaction.Validity, *txnvalidity.TransactionValidityError, error) {
	fmt.Println("In validate Transaction")
	ret, err := in.exec(runtime.TaggedTransactionQueueValidateTransaction, e)
	if err != nil {
		return nil, nil, err
	}
	return txnvalidity.UnmarshalTransactionValidity(ret)
}

// Version calls runtime function Core_Version
func (in *Instance) Version() (runtime.Version, error) {
	res, err := in.Exec(runtime.CoreVersion, []byte{})
	if err != nil {
		return nil, err
	}

	version := &runtime.VersionData{}
	err = version.Decode(res)
	// error comes from scale now, so do a string check
	if err != nil {
		if strings.Contains(err.Error(), "EOF") {
			// TODO: kusama seems to use the legacy version format
			lversion := &runtime.LegacyVersionData{}
			err = lversion.Decode(res)
			return lversion, err
		}
		return nil, err
	}

	return version, nil
}

// Metadata calls runtime function Metadata_metadata
func (in *Instance) Metadata() ([]byte, error) {
	return in.Exec(runtime.Metadata, []byte{})
}

// BabeConfiguration gets the configuration data for BABE from the runtime
func (in *Instance) BabeConfiguration() (*types.BabeConfiguration, error) {
	data, err := in.Exec(runtime.BabeAPIConfiguration, []byte{})
	if err != nil {
		return nil, err
	}

	bc := new(types.BabeConfiguration)
	err = scale.Unmarshal(data, bc)
	if err != nil {
		return nil, err
	}

	return bc, nil
}

// GrandpaAuthorities returns the genesis authorities from the runtime
func (in *Instance) GrandpaAuthorities() ([]types.Authority, error) {
	ret, err := in.Exec(runtime.GrandpaAuthorities, []byte{})
	if err != nil {
		return nil, err
	}

	var gar []types.GrandpaAuthoritiesRaw
	err = scale.Unmarshal(ret, &gar)
	if err != nil {
		return nil, err
	}

	return types.GrandpaAuthoritiesRawToAuthorities(gar)
}

// InitializeBlock calls runtime API function Core_initialise_block
func (in *Instance) InitializeBlock(header *types.Header) error {
	encodedHeader, err := scale.Marshal(*header)
	if err != nil {
		return fmt.Errorf("cannot encode header: %w", err)
	}

	_, err = in.Exec(runtime.CoreInitializeBlock, encodedHeader)
	return err
}

// InherentExtrinsics calls runtime API function BlockBuilder_inherent_extrinsics
func (in *Instance) InherentExtrinsics(data []byte) ([]byte, error) {
	return in.Exec(runtime.BlockBuilderInherentExtrinsics, data)
}

// ApplyExtrinsic calls runtime API function BlockBuilder_apply_extrinsic
func (in *Instance) ApplyExtrinsic(data types.Extrinsic) ([]byte, error) {
	return in.Exec(runtime.BlockBuilderApplyExtrinsic, data)
}

// FinalizeBlock calls runtime API function BlockBuilder_finalize_block
func (in *Instance) FinalizeBlock() (*types.Header, error) {
	data, err := in.Exec(runtime.BlockBuilderFinalizeBlock, []byte{})
	if err != nil {
		return nil, err
	}

	bh := types.NewEmptyHeader()
	err = scale.Unmarshal(data, bh)
	if err != nil {
		return nil, err
	}

	return bh, nil
}

// ExecuteBlock calls runtime function Core_execute_block
func (in *Instance) ExecuteBlock(block *types.Block) ([]byte, error) {
	// copy block since we're going to modify it
	b, err := block.DeepCopy()
	if err != nil {
		return nil, err
	}

	b.Header.Digest = types.NewDigest()

	// remove seal digest only
	for _, d := range block.Header.Digest.Types {
		switch d.Value().(type) {
		case types.SealDigest:
			continue
		default:
			err = b.Header.Digest.Add(d.Value())
			if err != nil {
				return nil, err
			}
		}
	}

	bdEnc, err := b.Encode()
	if err != nil {
		return nil, err
	}

	return in.Exec(runtime.CoreExecuteBlock, bdEnc)
}

// DecodeSessionKeys decodes the given public session keys. Returns a list of raw public keys including their key type.
func (in *Instance) DecodeSessionKeys(enc []byte) ([]byte, error) {
	return in.Exec(runtime.DecodeSessionKeys, enc)
}

// PaymentQueryInfo returns information of a given extrinsic
func (in *Instance) PaymentQueryInfo(ext []byte) (*types.TransactionPaymentQueryInfo, error) {
	encLen, err := scale.Marshal(uint32(len(ext)))
	if err != nil {
		return nil, err
	}

	resBytes, err := in.Exec(runtime.TransactionPaymentAPIQueryInfo, append(ext, encLen...))
	if err != nil {
		return nil, err
	}

	i := new(types.TransactionPaymentQueryInfo)
	if err = scale.Unmarshal(resBytes, i); err != nil {
		return nil, err
	}

	return i, nil
}

func (in *Instance) CheckInherents()      {} //nolint:revive
func (in *Instance) RandomSeed()          {} //nolint:revive
func (in *Instance) OffchainWorker()      {} //nolint:revive
func (in *Instance) GenerateSessionKeys() {} //nolint:revive
