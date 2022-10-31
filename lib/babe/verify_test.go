// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package babe

import (
	"errors"
	"fmt"
	"testing"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestHeader(t *testing.T, digest ...scale.VaryingDataTypeValue) *types.Header {
	t.Helper()
	header := types.NewEmptyHeader()
	header.Number = 1
	for _, d := range digest {
		err := header.Digest.Add(d)
		assert.NoError(t, err)
	}

	return header
}

func signAndAddSeal(t *testing.T, kp *sr25519.Keypair, header *types.Header, data []byte) {
	t.Helper()
	sig, err := kp.Sign(data)
	require.NoError(t, err)

	err = header.Digest.Add(types.SealDigest{
		ConsensusEngineID: types.BabeEngineID,
		Data:              sig,
	})
	assert.NoError(t, err)
}

func newEncodedBabeDigest(t *testing.T, value scale.VaryingDataTypeValue) []byte {
	t.Helper()
	babeDigest := types.NewBabeDigest()
	err := babeDigest.Set(value)
	require.NoError(t, err)

	enc, err := scale.Marshal(babeDigest)
	require.NoError(t, err)
	return enc
}

func encodeAndHashHeader(t *testing.T, header *types.Header) common.Hash {
	t.Helper()
	encHeader, err := scale.Marshal(*header)
	require.NoError(t, err)

	hash, err := common.Blake2bHash(encHeader)
	require.NoError(t, err)
	return hash
}

func newTestVerifier(kp *sr25519.Keypair, blockState BlockState,
	threshold *scale.Uint128, secSlots bool) *verifier {
	authority := types.NewAuthority(kp.Public(), uint64(1))
	info := &verifierInfo{
		authorities:    []types.Authority{*authority, *authority},
		randomness:     Randomness{},
		threshold:      threshold,
		secondarySlots: secSlots,
	}
	return newVerifier(blockState, 1, info)
}

func Test_getAuthorityIndex(t *testing.T) {
	digest := types.NewDigest()
	err := digest.Add(types.SealDigest{
		ConsensusEngineID: types.ConsensusEngineID{},
	})
	assert.NoError(t, err)
	headerNoPre := types.NewEmptyHeader()
	headerNoPre.Digest = digest

	digest2 := types.NewDigest()
	err = digest2.Add(types.PreRuntimeDigest{
		ConsensusEngineID: types.BabeEngineID,
		Data:              []byte{1},
	})
	assert.NoError(t, err)
	headerInvalidPre := types.NewEmptyHeader()
	headerInvalidPre.Digest = digest2

	// BabePrimaryPreDigest Case
	babeDigest := types.NewBabeDigest()
	err = babeDigest.Set(types.BabePrimaryPreDigest{AuthorityIndex: 21})
	assert.NoError(t, err)

	bdEnc, err := scale.Marshal(babeDigest)
	require.NoError(t, err)

	digestPrimary := types.NewDigest()
	err = digestPrimary.Add(types.PreRuntimeDigest{
		ConsensusEngineID: types.BabeEngineID,
		Data:              bdEnc,
	})
	assert.NoError(t, err)
	headerPrimary := types.NewEmptyHeader()
	headerPrimary.Digest = digestPrimary

	//BabeSecondaryVRFPreDigest Case
	babeDigest2 := types.NewBabeDigest()
	err = babeDigest2.Set(types.BabeSecondaryVRFPreDigest{AuthorityIndex: 21})
	assert.NoError(t, err)

	bdEnc2, err := scale.Marshal(babeDigest2)
	require.NoError(t, err)

	digestSecondary := types.NewDigest()
	err = digestSecondary.Add(types.PreRuntimeDigest{
		ConsensusEngineID: types.BabeEngineID,
		Data:              bdEnc2,
	})
	assert.NoError(t, err)
	headerSecondary := types.NewEmptyHeader()
	headerSecondary.Digest = digestSecondary

	//BabeSecondaryPlainPreDigest case
	babeDigest3 := types.NewBabeDigest()
	err = babeDigest3.Set(types.BabeSecondaryPlainPreDigest{AuthorityIndex: 21})
	assert.NoError(t, err)

	bdEnc3, err := scale.Marshal(babeDigest3)
	require.NoError(t, err)

	digestSecondaryPlain := types.NewDigest()
	err = digestSecondaryPlain.Add(types.PreRuntimeDigest{
		ConsensusEngineID: types.BabeEngineID,
		Data:              bdEnc3,
	})
	assert.NoError(t, err)
	headerSecondaryPlain := types.NewEmptyHeader()
	headerSecondaryPlain.Digest = digestSecondaryPlain

	type args struct {
		header *types.Header
	}
	tests := []struct {
		name   string
		args   args
		exp    uint32
		expErr error
	}{
		{
			name:   "No Digest",
			args:   args{types.NewEmptyHeader()},
			expErr: fmt.Errorf("for block hash %s: %w", types.NewEmptyHeader().Hash(), errNoDigest),
		},
		{
			name:   "First Digest Invalid Type",
			args:   args{headerNoPre},
			expErr: errors.New("first digest item is not pre-runtime digest"),
		},
		{
			name: "Invalid Preruntime Digest Type",
			args: args{headerInvalidPre},
			expErr: errors.New("cannot decode babe header from pre-digest: decoding struct: unmarshalling field at" +
				" index 0: EOF"),
		},
		{
			name: "BabePrimaryPreDigest Type",
			args: args{headerPrimary},
			exp:  21,
		},
		{
			name: "BabeSecondaryVRFPreDigest Type",
			args: args{headerSecondary},
			exp:  21,
		},
		{
			name: "BabeSecondaryPlainPreDigest Type",
			args: args{headerSecondaryPlain},
			exp:  21,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := getAuthorityIndex(tt.args.header)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.exp, res)
		})
	}
}

func Test_verifier_verifyPrimarySlotWinner(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockBlockState := NewMockBlockState(ctrl)

	//Generate keys
	kp, err := sr25519.GenerateKeypair()
	assert.NoError(t, err)

	auth := types.NewAuthority(kp.Public(), uint64(1))
	vi := &verifierInfo{
		authorities: []types.Authority{*auth},
		threshold:   &scale.Uint128{},
	}

	vi1 := &verifierInfo{
		authorities: []types.Authority{*auth},
		threshold:   scale.MaxUint128,
	}

	v := newVerifier(mockBlockState, 1, vi)
	v1 := newVerifier(mockBlockState, 1, vi1)

	output, proof, err := kp.VrfSign(makeTranscript(Randomness{}, uint64(1), 1))
	assert.NoError(t, err)

	type args struct {
		authorityIndex uint32
		slot           uint64
		vrfOutput      [sr25519.VRFOutputLength]byte
		vrfProof       [sr25519.VRFProofLength]byte
	}
	tests := []struct {
		name     string
		verifier verifier
		args     args
		exp      bool
		expErr   error
	}{
		{
			name:     "Over threshold",
			verifier: *v,
			args: args{
				slot:      1,
				vrfOutput: [32]byte{},
				vrfProof:  [64]byte{},
			},
			expErr: ErrVRFOutputOverThreshold,
		},
		{
			name:     "VRF not verified",
			verifier: *v1,
			args: args{
				slot:      1,
				vrfOutput: [32]byte{},
				vrfProof:  [64]byte{},
			},
		},
		{
			name:     "VRF verified",
			verifier: *v1,
			args: args{
				slot:      1,
				vrfOutput: output,
				vrfProof:  proof,
			},
			exp: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &tt.verifier
			res, err := b.verifyPrimarySlotWinner(tt.args.authorityIndex, tt.args.slot, tt.args.vrfOutput, tt.args.vrfProof)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.exp, res)
		})
	}
}

func Test_verifier_verifyPreRuntimeDigest(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockBlockState := NewMockBlockState(ctrl)

	//Generate keys
	kp, err := sr25519.GenerateKeypair()
	assert.NoError(t, err)

	//BabePrimaryPreDigest case
	output, proof, err := kp.VrfSign(makeTranscript(Randomness{}, uint64(1), 1))
	assert.NoError(t, err)

	secDigest1 := types.BabePrimaryPreDigest{
		SlotNumber: 1,
		VRFOutput:  output,
		VRFProof:   proof,
	}
	prd1, err := secDigest1.ToPreRuntimeDigest()
	assert.NoError(t, err)

	auth := types.NewAuthority(kp.Public(), uint64(1))
	vi := &verifierInfo{
		authorities: []types.Authority{*auth, *auth},
		threshold:   scale.MaxUint128,
	}

	v := newVerifier(mockBlockState, 1, vi)

	// Invalid
	v2 := newVerifier(mockBlockState, 13, vi)

	// Above threshold case
	vi1 := &verifierInfo{
		authorities: []types.Authority{*auth, *auth},
		threshold:   &scale.Uint128{},
	}

	v1 := newVerifier(mockBlockState, 1, vi1)

	//BabeSecondaryVRFPreDigest case
	secVRFDigest := types.BabeSecondaryVRFPreDigest{
		SlotNumber: 1,
		VrfOutput:  output,
		VrfProof:   proof,
	}

	digestSecondaryVRF := types.NewBabeDigest()
	err = digestSecondaryVRF.Set(secVRFDigest)
	assert.NoError(t, err)

	bdEnc, err := scale.Marshal(digestSecondaryVRF)
	require.NoError(t, err)

	babePRD := types.NewBABEPreRuntimeDigest(bdEnc)

	authVRFSec := types.NewAuthority(kp.Public(), uint64(1))
	viVRFSec := &verifierInfo{
		authorities: []types.Authority{*authVRFSec, *authVRFSec},
		threshold:   scale.MaxUint128,
	}

	viVRFSec2 := &verifierInfo{
		authorities:    []types.Authority{*authVRFSec, *authVRFSec},
		threshold:      scale.MaxUint128,
		secondarySlots: true,
	}

	vVRFSec := newVerifier(mockBlockState, 1, viVRFSec)
	vVRFSec2 := newVerifier(mockBlockState, 1, viVRFSec2)

	//BabeSecondaryPlainPreDigest case
	secDigest := types.BabeSecondaryPlainPreDigest{AuthorityIndex: 0, SlotNumber: uint64(1)}
	prd, err := secDigest.ToPreRuntimeDigest()
	assert.NoError(t, err)

	authSec := types.NewAuthority(kp.Public(), uint64(1))
	viSec := &verifierInfo{
		authorities: []types.Authority{*authSec, *authSec},
		threshold:   scale.MaxUint128,
	}

	viSec2 := &verifierInfo{
		authorities:    []types.Authority{*authSec, *authSec},
		threshold:      scale.MaxUint128,
		secondarySlots: true,
	}

	vSec := newVerifier(mockBlockState, 1, viSec)
	vSec2 := newVerifier(mockBlockState, 1, viSec2)

	type args struct {
		digest *types.PreRuntimeDigest
	}
	tests := []struct {
		name     string
		verifier verifier
		args     args
		exp      scale.VaryingDataTypeValue
		expErr   error
	}{
		{
			name:     "Invalid PreRuntimeDigest",
			verifier: verifier{},
			args:     args{&types.PreRuntimeDigest{Data: []byte{0}}},
			expErr: errors.New(
				"unable to find VaryingDataTypeValue with index: for key 0"),
		},
		{
			name:     "Invalid BlockProducer Index",
			verifier: verifier{},
			args:     args{prd},
			expErr:   ErrInvalidBlockProducerIndex,
		},
		{
			name:     "BabePrimaryPreDigest Case OK",
			verifier: *v,
			args:     args{prd1},
			exp: types.BabePrimaryPreDigest{
				SlotNumber: 1,
				VRFOutput:  output,
				VRFProof:   proof,
			},
		},
		{
			name:     "BabePrimaryPreDigest Case output over threshold",
			verifier: *v1,
			args:     args{prd1},
			expErr:   errors.New("vrf output over threshold"),
		},
		{
			name:     "BabePrimaryPreDigest Case Invalid",
			verifier: *v2,
			args:     args{prd1},
			expErr:   ErrBadSlotClaim,
		},
		{
			name:     "BabeSecondaryPlainPreDigest SecondarySlot false",
			verifier: *vSec,
			args:     args{prd},
			expErr:   ErrBadSlotClaim,
		},
		{
			name:     "BabeSecondaryPlainPreDigest invalid claim",
			verifier: *vSec2,
			args:     args{prd},
			expErr:   errors.New("invalid secondary slot claim"),
		},
		{
			name:     "BabeSecondaryVRFPreDigest SecondarySlot false",
			verifier: *vVRFSec,
			args:     args{babePRD},
			expErr:   ErrBadSlotClaim,
		},
		{
			name:     "BabeSecondaryVRFPreDigest invalid claim",
			verifier: *vVRFSec2,
			args:     args{babePRD},
			expErr:   errors.New("invalid secondary slot claim"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &tt.verifier
			res, err := b.verifyPreRuntimeDigest(tt.args.digest)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.exp, res)
		})
	}
}

func Test_verifier_verifyAuthorshipRight(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockBlockState := NewMockBlockState(ctrl)
	mockBlockStateErr := NewMockBlockState(ctrl)

	//Generate keys
	kp, err := sr25519.GenerateKeypair()
	assert.NoError(t, err)

	// Create a VRF output and proof
	output, proof, err := kp.VrfSign(makeTranscript(Randomness{}, uint64(1), 1))
	assert.NoError(t, err)

	testBabePrimaryPreDigest := types.BabePrimaryPreDigest{
		SlotNumber: 1,
		VRFOutput:  output,
		VRFProof:   proof,
	}
	testBabeSecondaryPlainPreDigest := types.BabeSecondaryPlainPreDigest{
		AuthorityIndex: 1,
		SlotNumber:     1,
	}
	testBabeSecondaryVRFPreDigest := types.BabeSecondaryVRFPreDigest{
		AuthorityIndex: 1,
		SlotNumber:     1,
		VrfOutput:      output,
		VrfProof:       proof,
	}
	testInvalidSeal := types.SealDigest{
		ConsensusEngineID: types.BabeEngineID,
		Data:              []byte{1},
	}
	testInvalidPreRuntimeDigest := types.PreRuntimeDigest{
		ConsensusEngineID: types.BabeEngineID,
		Data:              []byte{1},
	}

	// Primary Test Header
	encTestDigest := newEncodedBabeDigest(t, types.BabePrimaryPreDigest{AuthorityIndex: 0})
	testDigestPrimary := types.NewDigest()
	err = testDigestPrimary.Add(types.PreRuntimeDigest{
		ConsensusEngineID: types.BabeEngineID,
		Data:              encTestDigest,
	})
	assert.NoError(t, err)

	testHeaderPrimary := types.NewEmptyHeader()
	testHeaderPrimary.Digest = testDigestPrimary

	// Secondary Plain Test Header
	testParentPrd, err := testBabeSecondaryPlainPreDigest.ToPreRuntimeDigest()
	assert.NoError(t, err)

	testParentHeader := newTestHeader(t, *testParentPrd)
	testParentHash := encodeAndHashHeader(t, testParentHeader)
	testSecondaryPrd, err := testBabeSecondaryPlainPreDigest.ToPreRuntimeDigest()
	assert.NoError(t, err)

	testSecPlainHeader := newTestHeader(t, *testSecondaryPrd)
	testSecPlainHeader.ParentHash = testParentHash

	// Secondary Vrf Test Header
	encParentVrfDigest := newEncodedBabeDigest(t, testBabeSecondaryVRFPreDigest)
	testParentVrfHeader := newTestHeader(t, *types.NewBABEPreRuntimeDigest(encParentVrfDigest))

	testVrfParentHash := encodeAndHashHeader(t, testParentVrfHeader)
	encVrfHeader := newEncodedBabeDigest(t, testBabeSecondaryVRFPreDigest)
	testSecVrfHeader := newTestHeader(t, *types.NewBABEPreRuntimeDigest(encVrfHeader))
	testSecVrfHeader.ParentHash = testVrfParentHash

	h := common.MustHexToHash("0x01")
	h1 := []common.Hash{h}

	mockBlockState.EXPECT().GetHeader(h).Return(types.NewEmptyHeader(), nil)
	mockBlockState.EXPECT().GetBlockHashesBySlot(uint64(1)).Return(h1, nil)

	mockBlockStateErr.EXPECT().GetHeader(h).Return(nil, errors.New("get header error"))
	mockBlockStateErr.EXPECT().GetBlockHashesBySlot(uint64(1)).Return(h1, nil)

	// Case 0: First element not preruntime digest
	header0 := newTestHeader(t, testInvalidSeal, testInvalidSeal)

	// Case 1: Last element not seal
	header1 := newTestHeader(t, testInvalidPreRuntimeDigest, testInvalidPreRuntimeDigest)

	// Case 2: Fail to verify preruntime digest
	header2 := newTestHeader(t, testInvalidPreRuntimeDigest, testInvalidSeal)

	// Case 3: Invalid Seal Length
	babePrd, err := testBabePrimaryPreDigest.ToPreRuntimeDigest()
	assert.NoError(t, err)
	header3 := newTestHeader(t, *babePrd, testInvalidSeal)
	babeVerifier := newTestVerifier(kp, mockBlockState, scale.MaxUint128, false)

	// Case 4: Invalid signature - BabePrimaryPreDigest
	babePrd2, err := testBabePrimaryPreDigest.ToPreRuntimeDigest()
	assert.NoError(t, err)
	header4 := newTestHeader(t, *babePrd2)
	signAndAddSeal(t, kp, header4, []byte{1})
	babeVerifier2 := newTestVerifier(kp, mockBlockState, scale.MaxUint128, false)

	// Case 5: Invalid signature - BabeSecondaryPlainPreDigest
	babeSecPlainPrd, err := testBabeSecondaryPlainPreDigest.ToPreRuntimeDigest()
	assert.NoError(t, err)
	header5 := newTestHeader(t, *babeSecPlainPrd)
	signAndAddSeal(t, kp, header5, []byte{1})
	babeVerifier3 := newTestVerifier(kp, mockBlockState, scale.MaxUint128, true)

	// Case 6: Invalid signature - BabeSecondaryVrfPreDigest
	encSecVrfDigest := newEncodedBabeDigest(t, testBabeSecondaryVRFPreDigest)
	assert.NoError(t, err)
	header6 := newTestHeader(t, *types.NewBABEPreRuntimeDigest(encSecVrfDigest))
	signAndAddSeal(t, kp, header6, []byte{1})
	babeVerifier4 := newTestVerifier(kp, mockBlockState, scale.MaxUint128, true)

	// Case 7: GetAuthorityIndex Err
	babeParentPrd, err := testBabePrimaryPreDigest.ToPreRuntimeDigest()
	assert.NoError(t, err)
	babeParentHeader := newTestHeader(t, *babeParentPrd)

	parentHash := encodeAndHashHeader(t, babeParentHeader)
	babePrd3, err := testBabePrimaryPreDigest.ToPreRuntimeDigest()
	assert.NoError(t, err)

	header7 := newTestHeader(t, *babePrd3)
	header7.ParentHash = parentHash

	hash := encodeAndHashHeader(t, header7)
	signAndAddSeal(t, kp, header7, hash[:])
	babeVerifier5 := newTestVerifier(kp, mockBlockState, scale.MaxUint128, false)

	//// Case 8: Get header error
	babeVerifier6 := newTestVerifier(kp, mockBlockStateErr, scale.MaxUint128, false)

	tests := []struct {
		name     string
		verifier verifier
		header   *types.Header
		expErr   error
	}{
		{
			name:     "missing digest",
			verifier: verifier{},
			header:   types.NewEmptyHeader(),
			expErr:   errMissingDigestItems,
		},
		{
			name:     "first digest invalid",
			verifier: verifier{},
			header:   header0,
			expErr:   fmt.Errorf("%w: got types.SealDigest", types.ErrNoFirstPreDigest),
		},
		{
			name:     "last digest invalid",
			verifier: verifier{},
			header:   header1,
			expErr:   fmt.Errorf("%w: got types.PreRuntimeDigest", errLastDigestItemNotSeal),
		},
		{
			name:     "invalid preruntime digest data",
			verifier: verifier{},
			header:   header2,
			expErr: errors.New("failed to verify pre-runtime digest: decoding struct: unmarshalling field at index" +
				" 0: EOF"),
		},
		{
			name:     "invalid seal length",
			verifier: *babeVerifier,
			header:   header3,
			expErr:   errors.New("invalid signature length"),
		},
		{
			name:     "invalid seal signature - primary",
			verifier: *babeVerifier2,
			header:   header4,
			expErr:   ErrBadSignature,
		},
		{
			name:     "invalid seal signature - secondary plain",
			verifier: *babeVerifier3,
			header:   header5,
			expErr:   ErrBadSignature,
		},
		{
			name:     "invalid seal signature - secondary vrf",
			verifier: *babeVerifier4,
			header:   header6,
			expErr:   ErrBadSignature,
		},
		{
			name:     "valid digest items, getAuthorityIndex error",
			verifier: *babeVerifier5,
			header:   header7,
			expErr: fmt.Errorf("could not verify block equivocation: "+
				"failed to get authority index for block %s: for block hash %s: %w",
				h, types.NewEmptyHeader().Hash(), errNoDigest),
		},
		{
			name:     "get header err",
			verifier: *babeVerifier6,
			header:   header7,
			expErr: fmt.Errorf("could not verify block equivocation: "+
				"failed to get header for block %s: get header error", h),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &tt.verifier
			err := b.verifyAuthorshipRight(tt.header)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErr.Error())
			} else {
				assert.NoError(t, err)
			}

		})
	}
}

func Test_verifier_verifyBlockEquivocation(t *testing.T) {
	ctrl := gomock.NewController(t)

	// Generate keys
	kp, err := sr25519.GenerateKeypair()
	assert.NoError(t, err)

	auth := types.NewAuthority(kp.Public(), uint64(1))
	vi := &verifierInfo{
		authorities: []types.Authority{*auth, *auth},
		threshold:   scale.MaxUint128,
	}

	// Case 1. could not get authority index from header
	verifier1 := newVerifier(NewMockBlockState(ctrl), 1, vi)
	testHeader1 := types.NewEmptyHeader()

	// Case 2. could not get slot from header
	verifier2 := verifier1
	output, proof, err := kp.VrfSign(makeTranscript(Randomness{}, uint64(1), 1))
	assert.NoError(t, err)

	testDigest := types.BabePrimaryPreDigest{
		AuthorityIndex: 1,
		SlotNumber:     1,
		VRFOutput:      output,
		VRFProof:       proof,
	}
	prd, err := testDigest.ToPreRuntimeDigest()
	assert.NoError(t, err)

	testHeader2 := newTestHeader(t, *prd)
	testHeader2.Number = 0

	// Case 3. could not get block hashes by slot
	testHeader3 := newTestHeader(t, *prd)
	testHeader3.Number = 1

	mockBlockState3 := NewMockBlockState(ctrl)
	mockBlockState3.EXPECT().GetBlockHashesBySlot(uint64(1)).Return(
		nil, errors.New("test error"))

	verifier3 := newVerifier(mockBlockState3, 1, vi)

	// Case 4. no equivocation on finding the same block
	testHeader4 := newTestHeader(t, *prd)
	testHeader4.Number = 1
	testHash4 := testHeader4.Hash()
	mockBlockState4 := NewMockBlockState(ctrl)
	mockBlockState4.EXPECT().GetBlockHashesBySlot(uint64(1)).Return(
		[]common.Hash{testHash4}, nil)

	verifier4 := newVerifier(mockBlockState4, 1, vi)

	// Case 5. claiming a slot twice results in equivocation
	testHeader5 := newTestHeader(t, *prd)
	testHeader5.Number = 1

	output5, proof5, err := kp.VrfSign(makeTranscript(Randomness{}, uint64(1), 2))
	assert.NoError(t, err)

	testDigest5 := types.BabePrimaryPreDigest{
		AuthorityIndex: 1,
		SlotNumber:     1,
		VRFOutput:      output5,
		VRFProof:       proof5,
	}
	prd5, err := testDigest5.ToPreRuntimeDigest()
	assert.NoError(t, err)

	existingHeader := newTestHeader(t, *prd5)
	mockBlockState5 := NewMockBlockState(ctrl)
	mockBlockState5.EXPECT().GetBlockHashesBySlot(uint64(1)).Return(
		[]common.Hash{existingHeader.Hash()}, nil)
	mockBlockState5.EXPECT().GetHeader(existingHeader.Hash()).Return(
		existingHeader, nil)

	verifier5 := newVerifier(mockBlockState5, 1, vi)

	tests := []struct {
		name        string
		verifier    verifier
		header      *types.Header
		equivocated bool
		expErr      error
	}{
		{
			name:        "could not get authority index from header",
			verifier:    *verifier1,
			header:      testHeader1,
			equivocated: false,
			expErr:      fmt.Errorf("failed to get authority index: for block hash %s: %w", testHeader1.Hash(), errNoDigest),
		},
		{
			name:        "could not get slot from header",
			verifier:    *verifier2,
			header:      testHeader2,
			equivocated: false,
			expErr:      fmt.Errorf("failed to get slot from header of block %s: %w", testHeader2.Hash(), types.ErrGenesisHeader),
		},
		{
			name:        "could not get block hashes by slot",
			verifier:    *verifier3,
			header:      testHeader3,
			equivocated: false,
			expErr:      fmt.Errorf("failed to get blocks produced in slot: test error"),
		},
		{
			name:        "no equivocation on finding the same block",
			verifier:    *verifier4,
			header:      testHeader4,
			equivocated: false,
			expErr:      nil,
		},
		{
			name:        "claiming same slot twice results in equivocation",
			verifier:    *verifier5,
			header:      testHeader5,
			equivocated: true,
			expErr:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt := tt
			equivocated, err := tt.verifier.verifyBlockEquivocation(tt.header)
			assert.Equal(t, equivocated, tt.equivocated)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_verifier_verifyAuthorshipRightEquivocatory(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockBlockStateEquiv1 := NewMockBlockState(ctrl)
	mockBlockStateEquiv2 := NewMockBlockState(ctrl)
	mockBlockStateEquiv3 := NewMockBlockState(ctrl)

	//Generate keys
	kp, err := sr25519.GenerateKeypair()
	assert.NoError(t, err)

	output, proof, err := kp.VrfSign(makeTranscript(Randomness{}, uint64(1), 1))
	assert.NoError(t, err)

	output2, proof2, err := kp.VrfSign(makeTranscript(Randomness{}, uint64(1), 2))
	assert.NoError(t, err)
	secondDigestExisting := types.BabePrimaryPreDigest{
		AuthorityIndex: 1,
		SlotNumber:     1,
		VRFOutput:      output2,
		VRFProof:       proof2,
	}
	prdExisting, err := secondDigestExisting.ToPreRuntimeDigest()
	assert.NoError(t, err)

	headerExisting := newTestHeader(t, *prdExisting)
	hashExisting := encodeAndHashHeader(t, headerExisting)
	signAndAddSeal(t, kp, headerExisting, hashExisting[:])

	testBabeSecondaryPlainPreDigest := types.BabeSecondaryPlainPreDigest{
		AuthorityIndex: 1,
		SlotNumber:     1,
	}
	testBabeSecondaryVRFPreDigest := types.BabeSecondaryVRFPreDigest{
		AuthorityIndex: 1,
		SlotNumber:     1,
		VrfOutput:      output,
		VrfProof:       proof,
	}

	// BabePrimaryPreDigest case
	secDigest1 := types.BabePrimaryPreDigest{
		AuthorityIndex: 1,
		SlotNumber:     1,
		VRFOutput:      output,
		VRFProof:       proof,
	}
	prd1, err := secDigest1.ToPreRuntimeDigest()
	assert.NoError(t, err)

	auth := types.NewAuthority(kp.Public(), uint64(1))
	vi := &verifierInfo{
		authorities: []types.Authority{*auth, *auth},
		threshold:   scale.MaxUint128,
	}

	verifierEquivocatoryPrimary := newVerifier(mockBlockStateEquiv1, 1, vi)

	headerEquivocatoryPrimary := newTestHeader(t, *prd1)
	hashEquivocatoryPrimary := encodeAndHashHeader(t, headerEquivocatoryPrimary)
	signAndAddSeal(t, kp, headerEquivocatoryPrimary, hashEquivocatoryPrimary[:])

	mockBlockStateEquiv1.EXPECT().GetHeader(hashEquivocatoryPrimary).Return(headerEquivocatoryPrimary, nil)
	mockBlockStateEquiv1.EXPECT().GetBlockHashesBySlot(uint64(1)).Return(
		[]common.Hash{hashEquivocatoryPrimary, hashExisting}, nil)

	// Secondary Plain Test Header
	testParentPrd, err := testBabeSecondaryPlainPreDigest.ToPreRuntimeDigest()
	assert.NoError(t, err)
	testParentHeader := newTestHeader(t, *testParentPrd)

	testParentHash := encodeAndHashHeader(t, testParentHeader)
	testSecondaryPrd, err := testBabeSecondaryPlainPreDigest.ToPreRuntimeDigest()
	assert.NoError(t, err)
	testSecPlainHeader := newTestHeader(t, *testSecondaryPrd)
	testSecPlainHeader.ParentHash = testParentHash

	babeSecPlainPrd2, err := testBabeSecondaryPlainPreDigest.ToPreRuntimeDigest()
	assert.NoError(t, err)
	headerEquivocatorySecondaryPlain := newTestHeader(t, *babeSecPlainPrd2)

	hashEquivocatorySecondaryPlain := encodeAndHashHeader(t, headerEquivocatorySecondaryPlain)
	signAndAddSeal(t, kp, headerEquivocatorySecondaryPlain, hashEquivocatorySecondaryPlain[:])
	babeVerifier8 := newTestVerifier(kp, mockBlockStateEquiv2, scale.MaxUint128, true)

	mockBlockStateEquiv2.EXPECT().GetHeader(hashEquivocatorySecondaryPlain).Return(headerEquivocatorySecondaryPlain, nil)
	mockBlockStateEquiv2.EXPECT().GetBlockHashesBySlot(uint64(1)).Return(
		[]common.Hash{hashEquivocatorySecondaryPlain, hashExisting}, nil)

	// Secondary Vrf Test Header
	encParentVrfDigest := newEncodedBabeDigest(t, testBabeSecondaryVRFPreDigest)
	testParentVrfHeader := newTestHeader(t, *types.NewBABEPreRuntimeDigest(encParentVrfDigest))

	testVrfParentHash := encodeAndHashHeader(t, testParentVrfHeader)
	encVrfHeader := newEncodedBabeDigest(t, testBabeSecondaryVRFPreDigest)
	testSecVrfHeader := newTestHeader(t, *types.NewBABEPreRuntimeDigest(encVrfHeader))
	testSecVrfHeader.ParentHash = testVrfParentHash
	encVrfDigest := newEncodedBabeDigest(t, testBabeSecondaryVRFPreDigest)
	assert.NoError(t, err)
	headerEquivocatorySecondaryVRF := newTestHeader(t, *types.NewBABEPreRuntimeDigest(encVrfDigest))
	hashEquivocatorySecondaryVRF := encodeAndHashHeader(t, headerEquivocatorySecondaryVRF)
	signAndAddSeal(t, kp, headerEquivocatorySecondaryVRF, hashEquivocatorySecondaryVRF[:])
	babeVerifierEquivocatorySecondaryVRF := newTestVerifier(kp, mockBlockStateEquiv3, scale.MaxUint128, true)

	mockBlockStateEquiv3.EXPECT().GetHeader(hashEquivocatorySecondaryVRF).Return(headerEquivocatorySecondaryVRF, nil)
	mockBlockStateEquiv3.EXPECT().GetBlockHashesBySlot(uint64(1)).Return(
		[]common.Hash{hashEquivocatorySecondaryVRF, hashExisting}, nil)

	tests := []struct {
		name     string
		verifier verifier
		header   *types.Header
		expErr   error
	}{
		{
			name:     "equivocate - primary",
			verifier: *verifierEquivocatoryPrimary,
			header:   headerEquivocatoryPrimary,
			expErr:   fmt.Errorf("%w for block header %s", ErrProducerEquivocated, headerEquivocatoryPrimary.Hash()),
		},
		{
			name:     "equivocate - secondary plain",
			verifier: *babeVerifier8,
			header:   headerEquivocatorySecondaryPlain,
			expErr:   fmt.Errorf("%w for block header %s", ErrProducerEquivocated, headerEquivocatorySecondaryPlain.Hash()),
		},
		{
			name:     "equivocate - secondary vrf",
			verifier: *babeVerifierEquivocatorySecondaryVRF,
			header:   headerEquivocatorySecondaryVRF,
			expErr:   fmt.Errorf("%w for block header %s", ErrProducerEquivocated, headerEquivocatorySecondaryVRF.Hash()),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &tt.verifier
			err := b.verifyAuthorshipRight(tt.header)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErr.Error())
			} else {
				assert.NoError(t, err)
			}

		})
	}
}

func TestVerificationManager_getVerifierInfo(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockEpochStateGetErr := NewMockEpochState(ctrl)
	mockEpochStateHasErr := NewMockEpochState(ctrl)
	mockEpochStateThresholdErr := NewMockEpochState(ctrl)
	mockEpochStateOk := NewMockEpochState(ctrl)

	testHeader := types.NewEmptyHeader()

	mockEpochStateGetErr.EXPECT().GetEpochData(uint64(0), testHeader).Return(nil, state.ErrEpochNotInMemory)

	mockEpochStateHasErr.EXPECT().GetEpochData(uint64(0), testHeader).Return(&types.EpochData{}, nil)
	mockEpochStateHasErr.EXPECT().GetConfigData(uint64(0), testHeader).Return(&types.ConfigData{}, state.ErrConfigNotFound)

	mockEpochStateThresholdErr.EXPECT().GetEpochData(uint64(0), testHeader).Return(&types.EpochData{}, nil)
	mockEpochStateThresholdErr.EXPECT().GetConfigData(uint64(0), testHeader).
		Return(&types.ConfigData{
			C1: 3,
			C2: 1,
		}, nil)

	mockEpochStateOk.EXPECT().GetEpochData(uint64(0), testHeader).Return(&types.EpochData{}, nil)
	mockEpochStateOk.EXPECT().GetConfigData(uint64(0), testHeader).
		Return(&types.ConfigData{
			C1: 1,
			C2: 3,
		}, nil)

	vm0 := &VerificationManager{epochState: mockEpochStateGetErr}
	vm1 := &VerificationManager{epochState: mockEpochStateHasErr}
	vm2 := &VerificationManager{epochState: mockEpochStateThresholdErr}
	vm3 := &VerificationManager{epochState: mockEpochStateOk}

	tests := []struct {
		name   string
		vm     *VerificationManager
		epoch  uint64
		exp    *verifierInfo
		expErr error
	}{
		{
			name:   "getEpochData error",
			vm:     vm0,
			expErr: fmt.Errorf("failed to get epoch data for epoch %d: %w", 0, state.ErrEpochNotInMemory),
		},
		{
			name:   "getConfigData error",
			vm:     vm1,
			expErr: fmt.Errorf("failed to get config data: %w", state.ErrConfigNotFound),
		},
		{
			name:   "calculate threshold error",
			vm:     vm2,
			expErr: errors.New("failed to calculate threshold: invalid C1/C2: greater than 1"),
		},
		{
			name: "happy path",
			vm:   vm3,
			exp: &verifierInfo{
				threshold: scale.MaxUint128,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := tt.vm
			res, err := v.getVerifierInfo(tt.epoch, testHeader)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.exp, res)
		})
	}
}

func TestVerificationManager_VerifyBlock(t *testing.T) {
	//Generate keys
	kp, err := sr25519.GenerateKeypair()
	assert.NoError(t, err)

	// Create a VRF output and proof
	output, proof, err := kp.VrfSign(makeTranscript(Randomness{}, uint64(1), 1))
	assert.NoError(t, err)

	testBlockHeaderEmpty := types.NewEmptyHeader()
	testBlockHeaderEmpty.Number = 2

	ctrl := gomock.NewController(t)
	mockBlockStateEmpty := NewMockBlockState(ctrl)
	mockBlockStateCheckFinErr := NewMockBlockState(ctrl)
	mockBlockStateNotFinal := NewMockBlockState(ctrl)
	mockBlockStateNotFinal2 := NewMockBlockState(ctrl)

	mockEpochStateEmpty := NewMockEpochState(ctrl)
	mockEpochStateSetSlotErr := NewMockEpochState(ctrl)
	mockEpochStateGetEpochErr := NewMockEpochState(ctrl)
	mockEpochStateSkipVerifyErr := NewMockEpochState(ctrl)
	mockEpochStateSkipVerifyTrue := NewMockEpochState(ctrl)
	mockEpochStateGetVerifierInfoErr := NewMockEpochState(ctrl)
	mockEpochStateVerifyAuthorshipErr := NewMockEpochState(ctrl)

	errTestNumberIsFinalised := errors.New("test number is finalised error")
	mockBlockStateCheckFinErr.EXPECT().NumberIsFinalised(uint(1)).Return(false, errTestNumberIsFinalised)

	mockBlockStateNotFinal.EXPECT().NumberIsFinalised(uint(1)).Return(false, nil)

	mockBlockStateNotFinal2.EXPECT().NumberIsFinalised(uint(1)).Return(false, nil)
	errTestSetFirstSlot := errors.New("test set first slot error")
	mockEpochStateSetSlotErr.EXPECT().SetFirstSlot(uint64(1)).Return(errTestSetFirstSlot)

	errTestGetEpoch := errors.New("test get epoch error")
	mockEpochStateGetEpochErr.EXPECT().GetEpochForBlock(testBlockHeaderEmpty).
		Return(uint64(0), errTestGetEpoch)

	mockEpochStateSkipVerifyErr.EXPECT().GetEpochForBlock(testBlockHeaderEmpty).Return(uint64(1), nil)
	errTestGetEpochData := errors.New("test get epoch data error")
	mockEpochStateSkipVerifyErr.EXPECT().GetEpochData(uint64(1), testBlockHeaderEmpty).Return(nil, errTestGetEpochData)
	errTestSkipVerify := errors.New("test skip verify error")
	mockEpochStateSkipVerifyErr.EXPECT().SkipVerify(testBlockHeaderEmpty).Return(false, errTestSkipVerify)

	mockEpochStateSkipVerifyTrue.EXPECT().GetEpochForBlock(testBlockHeaderEmpty).Return(uint64(1), nil)
	mockEpochStateSkipVerifyTrue.EXPECT().GetEpochData(uint64(1), testBlockHeaderEmpty).Return(nil, errTestGetEpochData)
	mockEpochStateSkipVerifyTrue.EXPECT().SkipVerify(testBlockHeaderEmpty).Return(true, nil)

	mockEpochStateGetVerifierInfoErr.EXPECT().GetEpochForBlock(testBlockHeaderEmpty).Return(uint64(1), nil)
	mockEpochStateGetVerifierInfoErr.EXPECT().GetEpochData(uint64(1), testBlockHeaderEmpty).
		Return(nil, errTestGetEpochData)
	mockEpochStateGetVerifierInfoErr.EXPECT().SkipVerify(testBlockHeaderEmpty).Return(false, nil)

	mockEpochStateVerifyAuthorshipErr.EXPECT().GetEpochForBlock(testBlockHeaderEmpty).Return(uint64(1), nil)

	block1Header := types.NewEmptyHeader()
	block1Header.Number = 1

	testBabeSecondaryVRFPreDigest := types.BabeSecondaryVRFPreDigest{
		AuthorityIndex: 1,
		SlotNumber:     uint64(1),
		VrfOutput:      output,
		VrfProof:       proof,
	}
	encVrfDigest := newEncodedBabeDigest(t, testBabeSecondaryVRFPreDigest)
	assert.NoError(t, err)
	block1Header2 := newTestHeader(t, *types.NewBABEPreRuntimeDigest(encVrfDigest))

	authority := types.NewAuthority(kp.Public(), uint64(1))
	info := &verifierInfo{
		authorities:    []types.Authority{*authority, *authority},
		threshold:      scale.MaxUint128,
		secondarySlots: true,
	}

	vm0 := NewVerificationManager(mockBlockStateCheckFinErr, mockEpochStateEmpty)
	vm1 := NewVerificationManager(mockBlockStateNotFinal, mockEpochStateEmpty)
	vm2 := NewVerificationManager(mockBlockStateNotFinal2, mockEpochStateSetSlotErr)
	vm3 := NewVerificationManager(mockBlockStateNotFinal2, mockEpochStateGetEpochErr)
	vm4 := NewVerificationManager(mockBlockStateEmpty, mockEpochStateSkipVerifyErr)
	vm5 := NewVerificationManager(mockBlockStateEmpty, mockEpochStateSkipVerifyTrue)
	vm6 := NewVerificationManager(mockBlockStateEmpty, mockEpochStateGetVerifierInfoErr)
	vm8 := NewVerificationManager(mockBlockStateEmpty, mockEpochStateVerifyAuthorshipErr)
	vm8.epochInfo[1] = info

	tests := []struct {
		name   string
		vm     *VerificationManager
		header *types.Header
		expErr error
	}{
		{
			name:   "fail to check block 1 finalisation",
			vm:     vm0,
			header: block1Header,
			expErr: fmt.Errorf("failed to check if block 1 is finalised: %w", errTestNumberIsFinalised),
		},
		{
			name:   "get slot from header error",
			vm:     vm1,
			header: block1Header,
			expErr: fmt.Errorf("failed to get slot from header of block 1: %w", types.ErrChainHeadMissingDigest),
		},
		{
			name:   "set first slot error",
			vm:     vm2,
			header: block1Header2,
			expErr: fmt.Errorf("failed to set current epoch after receiving block 1: %w", errTestSetFirstSlot),
		},
		{
			name:   "get epoch error",
			vm:     vm3,
			header: testBlockHeaderEmpty,
			expErr: fmt.Errorf("failed to get epoch for block header: %w", errTestGetEpoch),
		},
		{
			name:   "skip verify err",
			vm:     vm4,
			header: testBlockHeaderEmpty,
			expErr: fmt.Errorf("failed to check if verification can be skipped: %w", errTestSkipVerify),
		},
		{
			name:   "skip verify true",
			vm:     vm5,
			header: testBlockHeaderEmpty,
		},
		{
			name:   "get verifierInfo err",
			vm:     vm6,
			header: testBlockHeaderEmpty,
			expErr: fmt.Errorf("failed to get verifier info for block 2: "+
				"failed to get epoch data for epoch 1: %w", errTestGetEpochData),
		},
		{
			name:   "verify block authorship err",
			vm:     vm8,
			header: testBlockHeaderEmpty,
			expErr: errMissingDigestItems,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := tt.vm
			err := v.VerifyBlock(tt.header)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestVerificationManager_SetOnDisabled(t *testing.T) {
	//Generate keys
	kp, err := sr25519.GenerateKeypair()
	assert.NoError(t, err)

	testHeader := types.NewEmptyHeader()
	testHeader.Number = 2

	ctrl := gomock.NewController(t)
	mockBlockStateEmpty := NewMockBlockState(ctrl)
	mockBlockStateIsDescendantErr := NewMockBlockState(ctrl)
	mockBlockStateAuthorityDisabled := NewMockBlockState(ctrl)
	mockBlockStateOk := NewMockBlockState(ctrl)

	mockEpochStateGetEpochErr := NewMockEpochState(ctrl)
	mockEpochStateGetEpochDataErr := NewMockEpochState(ctrl)
	mockEpochStateIndexLenErr := NewMockEpochState(ctrl)
	mockEpochStateSetDisabledProd := NewMockEpochState(ctrl)
	mockEpochStateOk := NewMockEpochState(ctrl)
	mockEpochStateOk2 := NewMockEpochState(ctrl)
	mockEpochStateOk3 := NewMockEpochState(ctrl)

	errTestGetEpoch := errors.New("test get epoch error")
	mockEpochStateGetEpochErr.EXPECT().GetEpochForBlock(types.NewEmptyHeader()).Return(uint64(0), errTestGetEpoch)

	mockEpochStateGetEpochDataErr.EXPECT().GetEpochForBlock(types.NewEmptyHeader()).Return(uint64(0), nil)
	errTestGetEpochData := errors.New("test get epoch data error")
	mockEpochStateGetEpochDataErr.EXPECT().GetEpochData(uint64(0), types.NewEmptyHeader()).Return(nil, errTestGetEpochData)

	mockEpochStateIndexLenErr.EXPECT().GetEpochForBlock(types.NewEmptyHeader()).Return(uint64(2), nil)

	mockEpochStateSetDisabledProd.EXPECT().GetEpochForBlock(types.NewEmptyHeader()).Return(uint64(2), nil)

	mockEpochStateOk.EXPECT().GetEpochForBlock(types.NewEmptyHeader()).Return(uint64(2), nil)
	errTestDescendant := errors.New("test descendant error")
	mockBlockStateIsDescendantErr.EXPECT().IsDescendantOf(gomock.Any(), gomock.Any()).Return(false, errTestDescendant)

	mockEpochStateOk2.EXPECT().GetEpochForBlock(testHeader).Return(uint64(2), nil)
	mockBlockStateAuthorityDisabled.EXPECT().IsDescendantOf(gomock.Any(), gomock.Any()).Return(true, nil)

	mockEpochStateOk3.EXPECT().GetEpochForBlock(testHeader).Return(uint64(2), nil)
	mockBlockStateOk.EXPECT().IsDescendantOf(gomock.Any(), gomock.Any()).Return(false, nil)

	authority := types.NewAuthority(kp.Public(), uint64(1))
	info := &verifierInfo{
		authorities:    []types.Authority{*authority, *authority},
		threshold:      scale.MaxUint128,
		secondarySlots: true,
	}

	disabledInfo := []*onDisabledInfo{
		{
			blockNumber: 2,
		},
	}

	vm0 := NewVerificationManager(mockBlockStateEmpty, mockEpochStateGetEpochErr)
	vm1 := NewVerificationManager(mockBlockStateEmpty, mockEpochStateGetEpochDataErr)
	vm1.epochInfo[1] = info

	vm2 := NewVerificationManager(mockBlockStateEmpty, mockEpochStateIndexLenErr)
	vm2.epochInfo[2] = info

	vm3 := NewVerificationManager(mockBlockStateEmpty, mockEpochStateSetDisabledProd)
	vm3.epochInfo[2] = info

	vm4 := NewVerificationManager(mockBlockStateIsDescendantErr, mockEpochStateOk)
	vm4.epochInfo[2] = info
	vm4.onDisabled[2] = map[uint32][]*onDisabledInfo{}
	vm4.onDisabled[2][0] = disabledInfo

	vm5 := NewVerificationManager(mockBlockStateAuthorityDisabled, mockEpochStateOk2)
	vm5.epochInfo[2] = info
	vm5.onDisabled[2] = map[uint32][]*onDisabledInfo{}
	vm5.onDisabled[2][0] = disabledInfo

	vm6 := NewVerificationManager(mockBlockStateOk, mockEpochStateOk3)
	vm6.epochInfo[2] = info
	vm6.onDisabled[2] = map[uint32][]*onDisabledInfo{}
	vm6.onDisabled[2][0] = disabledInfo

	type args struct {
		index  uint32
		header *types.Header
	}
	tests := []struct {
		name   string
		vm     *VerificationManager
		args   args
		expErr error
	}{
		{
			name: "get epoch err",
			vm:   vm0,
			args: args{
				header: types.NewEmptyHeader(),
			},
			expErr: errTestGetEpoch,
		},
		{
			name: "get epoch data err",
			vm:   vm1,
			args: args{
				header: types.NewEmptyHeader(),
			},
			expErr: fmt.Errorf("failed to get epoch data for epoch %d: %w", 0, errTestGetEpochData),
		},
		{
			name: "index length error",
			vm:   vm2,
			args: args{
				index:  10000,
				header: types.NewEmptyHeader(),
			},
			expErr: ErrInvalidBlockProducerIndex,
		},
		{
			name: "set disabled producers",
			vm:   vm3,
			args: args{
				header: types.NewEmptyHeader(),
			},
		},
		{
			name: "is Descendant of err",
			vm:   vm4,
			args: args{
				header: types.NewEmptyHeader(),
			},
			expErr: errTestDescendant,
		},
		{
			name: "authority already disabled",
			vm:   vm5,
			args: args{
				header: testHeader,
			},
			expErr: ErrAuthorityAlreadyDisabled,
		},
		{
			name: "happy path",
			vm:   vm6,
			args: args{
				header: testHeader,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := tt.vm
			err := v.SetOnDisabled(tt.args.index, tt.args.header)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
