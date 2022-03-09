// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package babe

import (
	"errors"
	"fmt"
	"math/big"
	"testing"

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

func newTestVerifier(t *testing.T, kp *sr25519.Keypair, blockState BlockState,
	threshold *scale.Uint128, secSlots bool) *verifier {
	t.Helper()
	authority := types.NewAuthority(kp.Public(), uint64(1))
	info := &verifierInfo{
		authorities:    []types.Authority{*authority, *authority},
		randomness:     Randomness{},
		threshold:      threshold,
		secondarySlots: secSlots,
	}
	verifier, err := newVerifier(blockState, 1, info)
	require.NoError(t, err)
	return verifier
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
			expErr: errors.New("no digest provided"),
		},
		{
			name:   "First Digest Invalid Type",
			args:   args{headerNoPre},
			expErr: errors.New("first digest item is not pre-runtime digest"),
		},
		{
			name:   "Invalid Preruntime Digest Type",
			args:   args{headerInvalidPre},
			expErr: errors.New("cannot decode babe header from pre-digest: EOF, field: 0"),
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

	v, err := newVerifier(mockBlockState, 1, vi)
	assert.NoError(t, err)

	v1, err := newVerifier(mockBlockState, 1, vi1)
	assert.NoError(t, err)

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

	v, err := newVerifier(mockBlockState, 1, vi)
	assert.NoError(t, err)

	// Invalid
	v2, err := newVerifier(mockBlockState, 13, vi)
	assert.NoError(t, err)

	// Above threshold case
	vi1 := &verifierInfo{
		authorities: []types.Authority{*auth, *auth},
		threshold:   &scale.Uint128{},
	}

	v1, err := newVerifier(mockBlockState, 1, vi1)
	assert.NoError(t, err)

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

	vVRFSec, err := newVerifier(mockBlockState, 1, viVRFSec)
	assert.NoError(t, err)

	vVRFSec2, err := newVerifier(mockBlockState, 1, viVRFSec2)
	assert.NoError(t, err)

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

	vSec, err := newVerifier(mockBlockState, 1, viSec)
	assert.NoError(t, err)

	vSec2, err := newVerifier(mockBlockState, 1, viSec2)
	assert.NoError(t, err)

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
			expErr:   errors.New("unable to find VaryingDataTypeValue with index: 0"),
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
	mockBlockStateEquiv1 := NewMockBlockState(ctrl)
	mockBlockStateEquiv2 := NewMockBlockState(ctrl)
	mockBlockStateEquiv3 := NewMockBlockState(ctrl)

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

	mockBlockState.EXPECT().GetAllBlocksAtDepth(gomock.Any()).Return(h1)
	mockBlockState.EXPECT().GetHeader(h).Return(types.NewEmptyHeader(), nil)

	mockBlockStateErr.EXPECT().GetAllBlocksAtDepth(gomock.Any()).Return(h1)
	mockBlockStateErr.EXPECT().GetHeader(h).Return(nil, errors.New("get header error"))

	mockBlockStateEquiv1.EXPECT().GetAllBlocksAtDepth(gomock.Any()).Return(h1)
	mockBlockStateEquiv1.EXPECT().GetHeader(h).Return(testHeaderPrimary, nil)

	mockBlockStateEquiv2.EXPECT().GetAllBlocksAtDepth(gomock.Any()).Return(h1)
	mockBlockStateEquiv2.EXPECT().GetHeader(h).Return(testSecPlainHeader, nil)
	mockBlockStateEquiv3.EXPECT().GetAllBlocksAtDepth(gomock.Any()).Return(h1)
	mockBlockStateEquiv3.EXPECT().GetHeader(h).Return(testSecVrfHeader, nil)

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
	babeVerifier := newTestVerifier(t, kp, mockBlockState, scale.MaxUint128, false)

	// Case 4: Invalid signature - BabePrimaryPreDigest
	babePrd2, err := testBabePrimaryPreDigest.ToPreRuntimeDigest()
	assert.NoError(t, err)
	header4 := newTestHeader(t, *babePrd2)

	signAndAddSeal(t, kp, header4, []byte{1})
	babeVerifier2 := newTestVerifier(t, kp, mockBlockState, scale.MaxUint128, false)

	// Case 5: Invalid signature - BabeSecondaryPlainPreDigest
	babeSecPlainPrd, err := testBabeSecondaryPlainPreDigest.ToPreRuntimeDigest()
	assert.NoError(t, err)
	header5 := newTestHeader(t, *babeSecPlainPrd)

	signAndAddSeal(t, kp, header5, []byte{1})
	babeVerifier3 := newTestVerifier(t, kp, mockBlockState, scale.MaxUint128, true)

	// Case 6: Invalid signature - BabeSecondaryVrfPreDigest
	encSecVrfDigest := newEncodedBabeDigest(t, testBabeSecondaryVRFPreDigest)
	assert.NoError(t, err)
	header6 := newTestHeader(t, *types.NewBABEPreRuntimeDigest(encSecVrfDigest))

	signAndAddSeal(t, kp, header6, []byte{1})
	babeVerifier4 := newTestVerifier(t, kp, mockBlockState, scale.MaxUint128, true)

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
	babeVerifier5 := newTestVerifier(t, kp, mockBlockState, scale.MaxUint128, false)

	//// Case 8: Get header error
	babeVerifier6 := newTestVerifier(t, kp, mockBlockStateErr, scale.MaxUint128, false)

	// Case 9: Equivocate case primary
	babeVerifier7 := newTestVerifier(t, kp, mockBlockStateEquiv1, scale.MaxUint128, false)

	// Case 10: Equivocate case secondary plain
	babeSecPlainPrd2, err := testBabeSecondaryPlainPreDigest.ToPreRuntimeDigest()
	assert.NoError(t, err)
	header8 := newTestHeader(t, *babeSecPlainPrd2)

	hash2 := encodeAndHashHeader(t, header8)
	signAndAddSeal(t, kp, header8, hash2[:])
	babeVerifier8 := newTestVerifier(t, kp, mockBlockStateEquiv2, scale.MaxUint128, true)

	// Case 11: equivocation case secondary VRF
	encVrfDigest := newEncodedBabeDigest(t, testBabeSecondaryVRFPreDigest)
	assert.NoError(t, err)
	header9 := newTestHeader(t, *types.NewBABEPreRuntimeDigest(encVrfDigest))

	hash3 := encodeAndHashHeader(t, header9)
	signAndAddSeal(t, kp, header9, hash3[:])
	babeVerifier9 := newTestVerifier(t, kp, mockBlockStateEquiv3, scale.MaxUint128, true)

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
			expErr:   errors.New("block header is missing digest items"),
		},
		{
			name:     "first digest invalid",
			verifier: verifier{},
			header:   header0,
			expErr:   errors.New("first digest item is not pre-digest"),
		},
		{
			name:     "last digest invalid",
			verifier: verifier{},
			header:   header1,
			expErr:   errors.New("last digest item is not seal"),
		},
		{
			name:     "invalid preruntime digest data",
			verifier: verifier{},
			header:   header2,
			expErr:   errors.New("failed to verify pre-runtime digest: EOF, field: 0"),
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
		},
		{
			name:     "get header err",
			verifier: *babeVerifier6,
			header:   header7,
		},
		{
			name:     "equivocate - primary",
			verifier: *babeVerifier7,
			header:   header7,
			expErr:   ErrProducerEquivocated,
		},
		{
			name:     "equivocate - secondary plain",
			verifier: *babeVerifier8,
			header:   header8,
			expErr:   ErrProducerEquivocated,
		},
		{
			name:     "equivocate - secondary vrf",
			verifier: *babeVerifier9,
			header:   header9,
			expErr:   ErrProducerEquivocated,
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

func TestVerificationManager_getConfigData(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockBlockState := NewMockBlockState(ctrl)
	mockEpochStateEmpty := NewMockEpochState(ctrl)
	mockEpochStateHasErr := NewMockEpochState(ctrl)
	mockEpochStateGetErr := NewMockEpochState(ctrl)

	mockEpochStateEmpty.EXPECT().HasConfigData(uint64(0)).Return(false, nil)
	mockEpochStateHasErr.EXPECT().HasConfigData(uint64(0)).Return(false, errNoConfigData)
	mockEpochStateGetErr.EXPECT().HasConfigData(uint64(0)).Return(true, nil)
	mockEpochStateGetErr.EXPECT().GetConfigData(uint64(0)).Return(nil, errNoConfigData)

	vm0, err := NewVerificationManager(mockBlockState, mockEpochStateEmpty)
	assert.NoError(t, err)
	vm1, err := NewVerificationManager(mockBlockState, mockEpochStateHasErr)
	assert.NoError(t, err)
	vm2, err := NewVerificationManager(mockBlockState, mockEpochStateGetErr)
	assert.NoError(t, err)
	tests := []struct {
		name   string
		vm     *VerificationManager
		epoch  uint64
		exp    *types.ConfigData
		expErr error
	}{
		{
			name:   "cant find ConfigData",
			vm:     vm0,
			expErr: errNoConfigData,
		},
		{
			name:   "hasConfigData error",
			vm:     vm1,
			expErr: errNoConfigData,
		},
		{
			name:   "getConfigData error",
			vm:     vm2,
			expErr: errNoConfigData,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := tt.vm
			res, err := v.getConfigData(tt.epoch, types.NewEmptyHeader())
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.exp, res)
		})
	}
}

func TestVerificationManager_getVerifierInfo(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockBlockState := NewMockBlockState(ctrl)
	mockEpochStateGetErr := NewMockEpochState(ctrl)
	mockEpochStateHasErr := NewMockEpochState(ctrl)
	mockEpochStateThresholdErr := NewMockEpochState(ctrl)
	mockEpochStateOk := NewMockEpochState(ctrl)

	mockEpochStateGetErr.EXPECT().GetEpochData(gomock.Eq(uint64(0))).Return(nil, errNoConfigData)

	mockEpochStateHasErr.EXPECT().GetEpochData(gomock.Eq(uint64(0))).Return(&types.EpochData{}, nil)
	mockEpochStateHasErr.EXPECT().HasConfigData(gomock.Eq(uint64(0))).Return(false, errNoConfigData)

	mockEpochStateThresholdErr.EXPECT().GetEpochData(gomock.Eq(uint64(0))).Return(&types.EpochData{}, nil)
	mockEpochStateThresholdErr.EXPECT().HasConfigData(gomock.Eq(uint64(0))).Return(true, nil)
	mockEpochStateThresholdErr.EXPECT().GetConfigData(gomock.Eq(uint64(0))).
		Return(&types.ConfigData{
			C1: 3,
			C2: 1,
		}, nil)

	mockEpochStateOk.EXPECT().GetEpochData(gomock.Eq(uint64(0))).Return(&types.EpochData{}, nil)
	mockEpochStateOk.EXPECT().HasConfigData(gomock.Eq(uint64(0))).Return(true, nil)
	mockEpochStateOk.EXPECT().GetConfigData(gomock.Eq(uint64(0))).
		Return(&types.ConfigData{
			C1: 1,
			C2: 3,
		}, nil)

	vm0, err := NewVerificationManager(mockBlockState, mockEpochStateGetErr)
	assert.NoError(t, err)
	vm1, err := NewVerificationManager(mockBlockState, mockEpochStateHasErr)
	assert.NoError(t, err)
	vm2, err := NewVerificationManager(mockBlockState, mockEpochStateThresholdErr)
	assert.NoError(t, err)
	vm3, err := NewVerificationManager(mockBlockState, mockEpochStateOk)
	assert.NoError(t, err)

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
			expErr: fmt.Errorf("failed to get epoch data for epoch %d: %w", 0, errNoConfigData),
		},
		{
			name:   "getConfigData error",
			vm:     vm1,
			expErr: fmt.Errorf("failed to get config data: %w", errNoConfigData),
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
			res, err := v.getVerifierInfo(tt.epoch, types.NewEmptyHeader())
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
	testBlockHeaderEmpty.Number = big.NewInt(2)

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
	mockEpochStateNilBlockStateErr := NewMockEpochState(ctrl)
	mockEpochStateVerifyAuthorshipErr := NewMockEpochState(ctrl)

	mockBlockStateCheckFinErr.EXPECT().NumberIsFinalised(gomock.Eq(big.NewInt(1))).Return(false, errFailedFinalisation)

	mockBlockStateNotFinal.EXPECT().NumberIsFinalised(gomock.Eq(big.NewInt(1))).Return(false, nil)

	mockBlockStateNotFinal2.EXPECT().NumberIsFinalised(gomock.Eq(big.NewInt(1))).Return(false, nil)
	mockEpochStateSetSlotErr.EXPECT().SetFirstSlot(gomock.Eq(uint64(1))).Return(errSetFirstSlot)

	mockEpochStateGetEpochErr.EXPECT().GetEpochForBlock(gomock.Eq(testBlockHeaderEmpty)).
		Return(uint64(0), errGetEpoch)

	mockEpochStateSkipVerifyErr.EXPECT().GetEpochForBlock(gomock.Eq(testBlockHeaderEmpty)).Return(uint64(1), nil)
	mockEpochStateSkipVerifyErr.EXPECT().GetEpochData(gomock.Eq(uint64(1))).Return(nil, errGetEpochData)
	mockEpochStateSkipVerifyErr.EXPECT().SkipVerify(gomock.Eq(testBlockHeaderEmpty)).Return(false, errSkipVerify)

	mockEpochStateSkipVerifyTrue.EXPECT().GetEpochForBlock(gomock.Eq(testBlockHeaderEmpty)).Return(uint64(1), nil)
	mockEpochStateSkipVerifyTrue.EXPECT().GetEpochData(gomock.Eq(uint64(1))).Return(nil, errGetEpochData)
	mockEpochStateSkipVerifyTrue.EXPECT().SkipVerify(gomock.Eq(testBlockHeaderEmpty)).Return(true, nil)

	mockEpochStateGetVerifierInfoErr.EXPECT().GetEpochForBlock(gomock.Eq(testBlockHeaderEmpty)).Return(uint64(1), nil)
	mockEpochStateGetVerifierInfoErr.EXPECT().GetEpochData(gomock.Eq(uint64(1))).
		Return(nil, errGetEpochData)
	mockEpochStateGetVerifierInfoErr.EXPECT().SkipVerify(gomock.Eq(testBlockHeaderEmpty)).Return(false, nil)

	mockEpochStateNilBlockStateErr.EXPECT().GetEpochForBlock(gomock.Eq(testBlockHeaderEmpty)).Return(uint64(1), nil)
	mockEpochStateVerifyAuthorshipErr.EXPECT().GetEpochForBlock(gomock.Eq(testBlockHeaderEmpty)).Return(uint64(1), nil)

	block1Header := types.NewEmptyHeader()
	block1Header.Number = big.NewInt(1)

	testBabeSecondaryVRFPreDigest := types.BabeSecondaryVRFPreDigest{
		AuthorityIndex: 1,
		SlotNumber:     uint64(1),
		VrfOutput:      output,
		VrfProof:       proof,
	}
	encVrfDigest := newEncodedBabeDigest(t, testBabeSecondaryVRFPreDigest)
	assert.NoError(t, err)
	block1Header2 := newTestHeader(t, *types.NewBABEPreRuntimeDigest(encVrfDigest))
	block1Header2.Number = big.NewInt(1)

	authority := types.NewAuthority(kp.Public(), uint64(1))
	info := &verifierInfo{
		authorities:    []types.Authority{*authority, *authority},
		threshold:      scale.MaxUint128,
		secondarySlots: true,
	}

	vm0, err := NewVerificationManager(mockBlockStateCheckFinErr, mockEpochStateEmpty)
	assert.NoError(t, err)
	vm1, err := NewVerificationManager(mockBlockStateNotFinal, mockEpochStateEmpty)
	assert.NoError(t, err)
	vm2, err := NewVerificationManager(mockBlockStateNotFinal2, mockEpochStateSetSlotErr)
	assert.NoError(t, err)
	vm3, err := NewVerificationManager(mockBlockStateNotFinal2, mockEpochStateGetEpochErr)
	assert.NoError(t, err)
	vm4, err := NewVerificationManager(mockBlockStateEmpty, mockEpochStateSkipVerifyErr)
	assert.NoError(t, err)
	vm5, err := NewVerificationManager(mockBlockStateEmpty, mockEpochStateSkipVerifyTrue)
	assert.NoError(t, err)
	vm6, err := NewVerificationManager(mockBlockStateEmpty, mockEpochStateGetVerifierInfoErr)
	assert.NoError(t, err)
	vm7 := &VerificationManager{
		epochState: mockEpochStateNilBlockStateErr,
		epochInfo:  make(map[uint64]*verifierInfo),
		onDisabled: make(map[uint64]map[uint32][]*onDisabledInfo),
	}
	vm8, err := NewVerificationManager(mockBlockStateEmpty, mockEpochStateVerifyAuthorshipErr)
	assert.NoError(t, err)

	vm7.epochInfo[1] = info
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
			expErr: fmt.Errorf("failed to check if block 1 is finalised: %w", errFailedFinalisation),
		},
		{
			name:   "get slot from header error",
			vm:     vm1,
			header: block1Header,
			expErr: fmt.Errorf("failed to get slot from block 1: %w", errMissingDigest),
		},
		{
			name:   "set first slot error",
			vm:     vm2,
			header: block1Header2,
			expErr: fmt.Errorf("failed to set current epoch after receiving block 1: %w", errSetFirstSlot),
		},
		{
			name:   "get epoch error",
			vm:     vm3,
			header: testBlockHeaderEmpty,
			expErr: fmt.Errorf("failed to get epoch for block header: %w", errGetEpoch),
		},
		{
			name:   "skip verify err",
			vm:     vm4,
			header: testBlockHeaderEmpty,
			expErr: fmt.Errorf("failed to check if verification can be skipped: %w", errSkipVerify),
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
				"failed to get epoch data for epoch 1: %w", errGetEpochData),
		},
		{
			name:   "nil blockState error",
			vm:     vm7,
			header: testBlockHeaderEmpty,
			expErr: fmt.Errorf("failed to create new BABE verifier: %w", ErrNilBlockState),
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
	testHeader.Number = big.NewInt(2)

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

	mockEpochStateGetEpochErr.EXPECT().GetEpochForBlock(gomock.Eq(types.NewEmptyHeader())).Return(uint64(0), errGetEpoch)

	mockEpochStateGetEpochDataErr.EXPECT().GetEpochForBlock(gomock.Eq(types.NewEmptyHeader())).Return(uint64(0), nil)
	mockEpochStateGetEpochDataErr.EXPECT().GetEpochData(gomock.Eq(uint64(0))).Return(nil, errGetEpochData)

	mockEpochStateIndexLenErr.EXPECT().GetEpochForBlock(gomock.Eq(types.NewEmptyHeader())).Return(uint64(2), nil)

	mockEpochStateSetDisabledProd.EXPECT().GetEpochForBlock(gomock.Eq(types.NewEmptyHeader())).Return(uint64(2), nil)

	mockEpochStateOk.EXPECT().GetEpochForBlock(gomock.Eq(types.NewEmptyHeader())).Return(uint64(2), nil)
	mockBlockStateIsDescendantErr.EXPECT().IsDescendantOf(gomock.Any(), gomock.Any()).Return(false, errDescendant)

	mockEpochStateOk2.EXPECT().GetEpochForBlock(gomock.Eq(testHeader)).Return(uint64(2), nil)
	mockBlockStateAuthorityDisabled.EXPECT().IsDescendantOf(gomock.Any(), gomock.Any()).Return(true, nil)

	mockEpochStateOk3.EXPECT().GetEpochForBlock(gomock.Eq(testHeader)).Return(uint64(2), nil)
	mockBlockStateOk.EXPECT().IsDescendantOf(gomock.Any(), gomock.Any()).Return(false, nil)

	authority := types.NewAuthority(kp.Public(), uint64(1))
	info := &verifierInfo{
		authorities:    []types.Authority{*authority, *authority},
		threshold:      scale.MaxUint128,
		secondarySlots: true,
	}

	disabledInfo := []*onDisabledInfo{
		{
			blockNumber: big.NewInt(2),
		},
	}

	vm0, err := NewVerificationManager(mockBlockStateEmpty, mockEpochStateGetEpochErr)
	assert.NoError(t, err)

	vm1, err := NewVerificationManager(mockBlockStateEmpty, mockEpochStateGetEpochDataErr)
	assert.NoError(t, err)
	vm1.epochInfo[1] = info

	vm2, err := NewVerificationManager(mockBlockStateEmpty, mockEpochStateIndexLenErr)
	assert.NoError(t, err)
	vm2.epochInfo[2] = info

	vm3, err := NewVerificationManager(mockBlockStateEmpty, mockEpochStateSetDisabledProd)
	assert.NoError(t, err)
	vm3.epochInfo[2] = info

	vm4, err := NewVerificationManager(mockBlockStateIsDescendantErr, mockEpochStateOk)
	assert.NoError(t, err)
	vm4.epochInfo[2] = info
	vm4.onDisabled[2] = map[uint32][]*onDisabledInfo{}
	vm4.onDisabled[2][0] = disabledInfo

	vm5, err := NewVerificationManager(mockBlockStateAuthorityDisabled, mockEpochStateOk2)
	assert.NoError(t, err)
	vm5.epochInfo[2] = info
	vm5.onDisabled[2] = map[uint32][]*onDisabledInfo{}
	vm5.onDisabled[2][0] = disabledInfo

	vm6, err := NewVerificationManager(mockBlockStateOk, mockEpochStateOk3)
	assert.NoError(t, err)
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
			expErr: errGetEpoch,
		},
		{
			name: "get epoch data err",
			vm:   vm1,
			args: args{
				header: types.NewEmptyHeader(),
			},
			expErr: fmt.Errorf("failed to get epoch data for epoch %d: %w", 0, errGetEpochData),
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
			expErr: errDescendant,
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
