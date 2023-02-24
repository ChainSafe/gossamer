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
func (in *Instance) Version() (version runtime.Version) {
	return in.ctx.Version
}

// version calls runtime function Core_Version and returns the
// decoded version structure.
func (in *Instance) version() (version runtime.Version, err error) {
	res, err := in.Exec(runtime.CoreVersion, []byte{})
	if err != nil {
		return version, err
	}

	version, err = runtime.DecodeVersion(res)
	if err != nil {
		return version, fmt.Errorf("decoding version: %w", err)
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

	//fmt.Println(types.OpaqueKeyOwnershipProof{byte(0)})
	//if encodedKeyOwnershipProof[0] == byte(0) {
	//	return types.OpaqueKeyOwnershipProof{0}, nil
	//}
	return encodedKeyOwnershipProof[1:], nil
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

	//_, err = encoder.Write(keyOwnershipProof)
	//if err != nil {
	//	return fmt.Errorf("encoding key ownership proof: %w", err)
	//}

	// Prints [17 7 0 0 0 0 20 153 1 128 193 0 128 253 162 74 79 189 165 252 62 26 123 19 51 223 21 53 169 217 116 92 72 142 118 97 230 126 230 166 89 111 67 222 237 128 57 137 114 113 236 151 127 184 50 57 23 76 123 53 153 225 249 147 81 102 85 58 39 249 235 161 86 119 75 233 207 115 128 197 82 92 4 231 72 158 191 106 112 88 216 102 124 231 73 231 151 17 1 136 246 16 249 178 199 246 22 164 199 173 173 29 2 128 134 2 128 34 147 82 76 234 71 125 226 137 91 133 61 241 104 171 240 115 130 18 236 185 126 195 234 134 97 219 158 78 180 144 105 128 189 12 199 60 211 249 130 17 234 156 16 187 201 177 20 169 63 170 166 15 101 19 18 1 19 61 69 13 240 68 171 70 128 98 9 112 3 101 29 137 157 239 234 2 221 13 57 115 157 92 120 175 29 40 252 143 236 248 201 232 107 31 221 193 168 128 17 177 5 250 171 121 142 94 127 146 75 235 58 78 233 146 10 196 172 27 204 161 247 55 29 141 183 121 239 78 202 195 172 127 9 97 98 101 128 212 53 147 199 21 253 211 28 97 20 26 189 4 169 159 214 130 44 133 88 133 76 205 227 154 86 132 231 165 109 162 125 16 0 0 0 0 153 1 128 193 0 128 253 162 74 79 189 165 252 62 26 123 19 51 223 21 53 169 217 116 92 72 142 118 97 230 126 230 166 89 111 67 222 237 128 57 137 114 113 236 151 127 184 50 57 23 76 123 53 153 225 249 147 81 102 85 58 39 249 235 161 86 119 75 233 207 115 128 197 82 92 4 231 72 158 191 106 112 88 216 102 124 231 73 231 151 17 1 136 246 16 249 178 199 246 22 164 199 173 173 212 71 0 0 0 0 188 190 93 219 21 121 183 46 132 82 79 194 158 120 96 158 60 175 66 232 90 161 24 235 254 11 10 212 4 181 189 210 95 11 0 64 122 16 243 90 11 0 64 122 16 243 90 0 1 0 0 0]
	fmt.Println(keyOwnershipProof)

	res, err := scale.Marshal(keyOwnershipProof)
	if err != nil {
		return fmt.Errorf("encoding key ownership proof: %w", err)
	}

	// This doesnt seem length encoded, why??
	// Prints [25 7 17 7 0 0 0 0 20 153 1 128 193 0 128 253 162 74 79 189 165 252 62 26 123 19 51 223 21 53 169 217 116 92 72 142 118 97 230 126 230 166 89 111 67 222 237 128 57 137 114 113 236 151 127 184 50 57 23 76 123 53 153 225 249 147 81 102 85 58 39 249 235 161 86 119 75 233 207 115 128 197 82 92 4 231 72 158 191 106 112 88 216 102 124 231 73 231 151 17 1 136 246 16 249 178 199 246 22 164 199 173 173 29 2 128 134 2 128 34 147 82 76 234 71 125 226 137 91 133 61 241 104 171 240 115 130 18 236 185 126 195 234 134 97 219 158 78 180 144 105 128 189 12 199 60 211 249 130 17 234 156 16 187 201 177 20 169 63 170 166 15 101 19 18 1 19 61 69 13 240 68 171 70 128 98 9 112 3 101 29 137 157 239 234 2 221 13 57 115 157 92 120 175 29 40 252 143 236 248 201 232 107 31 221 193 168 128 17 177 5 250 171 121 142 94 127 146 75 235 58 78 233 146 10 196 172 27 204 161 247 55 29 141 183 121 239 78 202 195 172 127 9 97 98 101 128 212 53 147 199 21 253 211 28 97 20 26 189 4 169 159 214 130 44 133 88 133 76 205 227 154 86 132 231 165 109 162 125 16 0 0 0 0 153 1 128 193 0 128 253 162 74 79 189 165 252 62 26 123 19 51 223 21 53 169 217 116 92 72 142 118 97 230 126 230 166 89 111 67 222 237 128 57 137 114 113 236 151 127 184 50 57 23 76 123 53 153 225 249 147 81 102 85 58 39 249 235 161 86 119 75 233 207 115 128 197 82 92 4 231 72 158 191 106 112 88 216 102 124 231 73 231 151 17 1 136 246 16 249 178 199 246 22 164 199 173 173 212 71 0 0 0 0 188 190 93 219 21 121 183 46 132 82 79 194 158 120 96 158 60 175 66 232 90 161 24 235 254 11 10 212 4 181 189 210 95 11 0 64 122 16 243 90 11 0 64 122 16 243 90 0 1 0 0 0]
	fmt.Println(res)

	//keyOwnershipProofOption := &keyOwnershipProof
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

	keyOwnershipProof := types.GrandpaOpaqueKeyOwnershipProof{}
	err = scale.Unmarshal(encodedOpaqueKeyOwnershipProof, &keyOwnershipProof)
	if err != nil {
		return nil, fmt.Errorf("scale decoding: %w", err)
	}

	return keyOwnershipProof, nil
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

func (in *Instance) RandomSeed()          {} //nolint:revive
func (in *Instance) OffchainWorker()      {} //nolint:revive
func (in *Instance) GenerateSessionKeys() {} //nolint:revive
