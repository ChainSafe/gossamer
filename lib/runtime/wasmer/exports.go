// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package wasmer

import (
	"bytes"
	"fmt"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/transaction"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// ValidateTransaction runs the extrinsic through the runtime function
// TaggedTransactionQueue_validate_transaction and returns *transaction.Validity. The error can
// be a VDT of either transaction.InvalidTransaction or transaction.UnknownTransaction, or can represent
// a normal error i.e. unmarshalling error
func (in *Instance) ValidateTransaction(e types.Extrinsic) (
	*transaction.Validity, error) {
	ret, err := in.Exec(runtime.TaggedTransactionQueueValidateTransaction, e)
	if err != nil {
		return nil, err
	}

	return runtime.UnmarshalTransactionValidity(ret)
}

// Version returns the instance version.
// This is cheap to call since the instance version is cached.
// Note the instance version is set at creation and on code update.
func (in *Instance) Version() (runtime.Version, error) {
	if in.ctx.Version != nil {
		return *in.ctx.Version, nil
	}

	err := in.version()
	if err != nil {
		return runtime.Version{}, err
	}

	return *in.ctx.Version, nil
}

// version calls runtime function Core_Version and returns the
// decoded version structure.
func (in *Instance) version() error {
	res, err := in.Exec(runtime.CoreVersion, []byte{})
	if err != nil {
		return err
	}

	version, err := runtime.DecodeVersion(res)
	if err != nil {
		return fmt.Errorf("decoding version: %w", err)
	}

	in.ctx.Version = &version

	return nil
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

// BabeGenerateKeyOwnershipProof returns the babe key ownership proof from the runtime.
func (in *Instance) BabeGenerateKeyOwnershipProof(slot uint64, authorityID [32]byte) (
	types.OpaqueKeyOwnershipProof, error) {

	// scale encoded slot uint64 + scale encoded array of 32 bytes
	const maxBufferLength = 8 + 33
	buffer := bytes.NewBuffer(make([]byte, 0, maxBufferLength))
	encoder := scale.NewEncoder(buffer)
	err := encoder.Encode(slot)
	if err != nil {
		return nil, fmt.Errorf("encoding slot: %w", err)
	}
	err = encoder.Encode(authorityID)
	if err != nil {
		return nil, fmt.Errorf("encoding authority id: %w", err)
	}

	encodedKeyOwnershipProof, err := in.Exec(runtime.BabeAPIGenerateKeyOwnershipProof, buffer.Bytes())
	if err != nil {
		return nil, fmt.Errorf("executing %s: %w", runtime.BabeAPIGenerateKeyOwnershipProof, err)
	}

	var keyOwnershipProof *types.OpaqueKeyOwnershipProof
	err = scale.Unmarshal(encodedKeyOwnershipProof, &keyOwnershipProof)
	if err != nil {
		return nil, fmt.Errorf("scale decoding key ownership proof: %w", err)
	}

	if keyOwnershipProof == nil {
		return nil, nil
	}

	return *keyOwnershipProof, nil
}

// BabeSubmitReportEquivocationUnsignedExtrinsic reports equivocation report to the runtime.
func (in *Instance) BabeSubmitReportEquivocationUnsignedExtrinsic(
	equivocationProof types.BabeEquivocationProof, keyOwnershipProof types.OpaqueKeyOwnershipProof,
) error {
	buffer := bytes.NewBuffer(nil)
	encoder := scale.NewEncoder(buffer)
	err := encoder.Encode(equivocationProof)
	if err != nil {
		return fmt.Errorf("encoding equivocation proof: %w", err)
	}
	err = encoder.Encode(keyOwnershipProof)
	if err != nil {
		return fmt.Errorf("encoding key ownership proof: %w", err)
	}
	_, err = in.Exec(runtime.BabeAPISubmitReportEquivocationUnsignedExtrinsic, buffer.Bytes())
	return err
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
		digestValue, err := d.Value()
		if err != nil {
			return nil, fmt.Errorf("getting digest type value: %w", err)
		}
		switch digestValue.(type) {
		case types.SealDigest:
			continue
		default:
			err = b.Header.Digest.Add(digestValue)
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
func (in *Instance) PaymentQueryInfo(ext []byte) (*types.RuntimeDispatchInfo, error) {
	encLen, err := scale.Marshal(uint32(len(ext)))
	if err != nil {
		return nil, err
	}

	resBytes, err := in.Exec(runtime.TransactionPaymentAPIQueryInfo, append(ext, encLen...))
	if err != nil {
		return nil, err
	}

	dispatchInfo := new(types.RuntimeDispatchInfo)
	if err = scale.Unmarshal(resBytes, dispatchInfo); err != nil {
		return nil, err
	}

	return dispatchInfo, nil
}

// QueryCallInfo returns information of a given extrinsic
func (in *Instance) QueryCallInfo(ext []byte) (*types.RuntimeDispatchInfo, error) {
	encLen, err := scale.Marshal(uint32(len(ext)))
	if err != nil {
		return nil, err
	}

	resBytes, err := in.Exec(runtime.TransactionPaymentCallAPIQueryCallInfo, append(ext, encLen...))
	if err != nil {
		return nil, err
	}

	dispatchInfo := new(types.RuntimeDispatchInfo)
	if err = scale.Unmarshal(resBytes, dispatchInfo); err != nil {
		return nil, err
	}

	return dispatchInfo, nil
}

// QueryCallFeeDetails returns call fee details for given call
func (in *Instance) QueryCallFeeDetails(ext []byte) (*types.FeeDetails, error) {
	encLen, err := scale.Marshal(uint32(len(ext)))
	if err != nil {
		return nil, err
	}

	resBytes, err := in.Exec(runtime.TransactionPaymentCallAPIQueryCallFeeDetails, append(ext, encLen...))
	if err != nil {
		return nil, err
	}

	dispatchInfo := new(types.FeeDetails)
	if err = scale.Unmarshal(resBytes, dispatchInfo); err != nil {
		return nil, err
	}

	return dispatchInfo, nil
}

// CheckInherents checks inherents in the block verification process.
// TODO: use this in block verification process (#1873)
func (in *Instance) CheckInherents() {}

// GrandpaGenerateKeyOwnershipProof returns grandpa key ownership proof from the runtime.
func (in *Instance) GrandpaGenerateKeyOwnershipProof(authSetID uint64, authorityID ed25519.PublicKeyBytes) (
	types.GrandpaOpaqueKeyOwnershipProof, error) {
	const bufferSize = 8 + 32 // authSetID uint64 + ed25519.PublicKeyBytes
	buffer := bytes.NewBuffer(make([]byte, 0, bufferSize))
	encoder := scale.NewEncoder(buffer)
	err := encoder.Encode(authSetID)
	if err != nil {
		return nil, fmt.Errorf("encoding auth set id: %w", err)
	}
	err = encoder.Encode(authorityID)
	if err != nil {
		return nil, fmt.Errorf("encoding authority id: %w", err)
	}
	encodedOpaqueKeyOwnershipProof, err := in.Exec(runtime.GrandpaGenerateKeyOwnershipProof, buffer.Bytes())
	if err != nil {
		return nil, err
	}

	var keyOwnershipProof *types.GrandpaOpaqueKeyOwnershipProof
	err = scale.Unmarshal(encodedOpaqueKeyOwnershipProof, &keyOwnershipProof)
	if err != nil {
		return nil, fmt.Errorf("scale decoding: %w", err)
	}

	if keyOwnershipProof == nil {
		return nil, nil
	}

	return *keyOwnershipProof, nil
}

// GrandpaSubmitReportEquivocationUnsignedExtrinsic reports an equivocation report to the runtime.
func (in *Instance) GrandpaSubmitReportEquivocationUnsignedExtrinsic(
	equivocationProof types.GrandpaEquivocationProof, keyOwnershipProof types.GrandpaOpaqueKeyOwnershipProof,
) error {
	buffer := bytes.NewBuffer(nil)
	encoder := scale.NewEncoder(buffer)
	err := encoder.Encode(equivocationProof)
	if err != nil {
		return fmt.Errorf("encoding equivocation proof: %w", err)
	}
	err = encoder.Encode(keyOwnershipProof)
	if err != nil {
		return fmt.Errorf("encoding key ownership proof: %w", err)
	}
	_, err = in.Exec(runtime.GrandpaSubmitReportEquivocation, buffer.Bytes())
	if err != nil {
		return err
	}
	return nil
}

func (in *Instance) RandomSeed()          {}
func (in *Instance) OffchainWorker()      {}
func (in *Instance) GenerateSessionKeys() {}
