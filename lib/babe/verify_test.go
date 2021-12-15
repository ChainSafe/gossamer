// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package babe

import (
	"errors"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/babe/mocks"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"testing"
)

func Test_getAuthorityIndex(t *testing.T) {
	digest := types.NewDigest()
	err := digest.Add(types.SealDigest{
		ConsensusEngineID: types.ConsensusEngineID{},
		Data:              nil,
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
	mockBlockState := mocks.NewMockBlockState(ctrl)

	//Generate keys
	kp, err := sr25519.GenerateKeypair()
	assert.NoError(t, err)

	auth := types.NewAuthority(kp.Public(), uint64(1))
	vi := &verifierInfo{
		authorities:    []types.Authority{*auth},
		randomness:     Randomness{},
		threshold:      &scale.Uint128{},
		secondarySlots: false,
	}

	vi1 := &verifierInfo{
		authorities:    []types.Authority{*auth},
		randomness:     Randomness{},
		threshold:      scale.MaxUint128,
		secondarySlots: false,
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
				authorityIndex: 0,
				slot:           uint64(1),
				vrfOutput:      [32]byte{},
				vrfProof:       [64]byte{},
			},
			expErr: ErrVRFOutputOverThreshold,
		},
		{
			name:     "VRF not verified",
			verifier: *v1,
			args: args{
				authorityIndex: 0,
				slot:           uint64(1),
				vrfOutput:      [32]byte{},
				vrfProof:       [64]byte{},
			},
		},
		{
			name:     "VRF verified",
			verifier: *v1,
			args: args{
				authorityIndex: 0,
				slot:           uint64(1),
				vrfOutput:      output,
				vrfProof:       proof,
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
	mockBlockState := mocks.NewMockBlockState(ctrl)

	//Generate keys
	kp, err := sr25519.GenerateKeypair()
	assert.NoError(t, err)

	//BabePrimaryPreDigest case
	output, proof, err := kp.VrfSign(makeTranscript(Randomness{}, uint64(1), 1))
	assert.NoError(t, err)

	secDigest1 := types.BabePrimaryPreDigest{
		AuthorityIndex: 0,
		SlotNumber:     uint64(1),
		VRFOutput:      output,
		VRFProof:       proof,
	}
	prd1, err := secDigest1.ToPreRuntimeDigest()
	assert.NoError(t, err)

	auth := types.NewAuthority(kp.Public(), uint64(1))
	vi := &verifierInfo{
		authorities:    []types.Authority{*auth, *auth},
		randomness:     Randomness{},
		threshold:      scale.MaxUint128,
		secondarySlots: false,
	}

	v, err := newVerifier(mockBlockState, 1, vi)
	assert.NoError(t, err)

	// Invalid
	v2, err := newVerifier(mockBlockState, 13, vi)
	assert.NoError(t, err)

	// Above threshold case
	vi1 := &verifierInfo{
		authorities:    []types.Authority{*auth, *auth},
		randomness:     Randomness{},
		threshold:      &scale.Uint128{},
		secondarySlots: false,
	}

	v1, err := newVerifier(mockBlockState, 1, vi1)
	assert.NoError(t, err)

	//BabeSecondaryVRFPreDigest case
	secVRFDigest := types.BabeSecondaryVRFPreDigest{
		AuthorityIndex: 0,
		SlotNumber:     uint64(1),
		VrfOutput:      output,
		VrfProof:       proof,
	}

	digestSecondaryVRF := types.NewBabeDigest()
	err = digestSecondaryVRF.Set(secVRFDigest)
	assert.NoError(t, err)

	bdEnc, err := scale.Marshal(digestSecondaryVRF)
	require.NoError(t, err)

	babePRD := types.NewBABEPreRuntimeDigest(bdEnc)

	authVRFSec := types.NewAuthority(kp.Public(), uint64(1))
	viVRFSec := &verifierInfo{
		authorities:    []types.Authority{*authVRFSec, *authVRFSec},
		randomness:     Randomness{},
		threshold:      scale.MaxUint128,
		secondarySlots: false,
	}

	viVRFSec2 := &verifierInfo{
		authorities:    []types.Authority{*authVRFSec, *authVRFSec},
		randomness:     Randomness{},
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
		authorities:    []types.Authority{*authSec, *authSec},
		randomness:     Randomness{},
		threshold:      scale.MaxUint128,
		secondarySlots: false,
	}

	viSec2 := &verifierInfo{
		authorities:    []types.Authority{*authSec, *authSec},
		randomness:     Randomness{},
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
				AuthorityIndex: 0,
				SlotNumber:     uint64(1),
				VRFOutput:      output,
				VRFProof:       proof,
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

func createNewTestHeader(t *testing.T, digest ...scale.VaryingDataTypeValue) *types.Header {
	header := types.NewEmptyHeader()
	for _, d := range digest {
		err := header.Digest.Add(d)
		assert.NoError(t, err)
	}

	return header
}

func signAndAddSeal(t *testing.T, kp *sr25519.Keypair, header *types.Header, data []byte) error {
	t.Helper()
	sig, err := kp.Sign(data)
	assert.NoError(t, err)

	return header.Digest.Add(types.SealDigest{
		ConsensusEngineID: types.BabeEngineID,
		Data:              sig,
	})

}

func newEncodedBabeDigest(t *testing.T, value scale.VaryingDataTypeValue) ([]byte, error) {
	t.Helper()
	babeDigest := types.NewBabeDigest()
	err := babeDigest.Set(value)
	assert.NoError(t, err)
	return scale.Marshal(babeDigest)
}

func encodeAndHashHeader(t *testing.T, header *types.Header) (common.Hash, error) {
	t.Helper()
	encHeader, err := scale.Marshal(*header)
	assert.NoError(t, err)

	return common.Blake2bHash(encHeader)
}

func newTestVerifier(t *testing.T, kp *sr25519.Keypair, blockState BlockState, threshold *scale.Uint128, secSlots bool) (*verifier, error) {
	t.Helper()
	authority := types.NewAuthority(kp.Public(), uint64(1))
	info := &verifierInfo{
		authorities:    []types.Authority{*authority, *authority},
		randomness:     Randomness{},
		threshold:      threshold,
		secondarySlots: secSlots,
	}
	return newVerifier(blockState, 1, info)
}

/*
TODO for this test:
- Can I clean this test up? Helper funcs?
- think about why we dont handle errors
*/
func Test_verifier_verifyAuthorshipRight(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockBlockState := mocks.NewMockBlockState(ctrl)
	mockBlockStateErr := mocks.NewMockBlockState(ctrl)
	mockBlockStateEquiv1 := mocks.NewMockBlockState(ctrl)
	mockBlockStateEquiv2 := mocks.NewMockBlockState(ctrl)
	mockBlockStateEquiv3 := mocks.NewMockBlockState(ctrl)

	//Generate keys
	kp, err := sr25519.GenerateKeypair()
	assert.NoError(t, err)

	// Create a VRF output and proof
	output, proof, err := kp.VrfSign(makeTranscript(Randomness{}, uint64(1), 1))
	assert.NoError(t, err)

	testBabePrimaryPreDigest := types.BabePrimaryPreDigest{
		AuthorityIndex: 0,
		SlotNumber:     uint64(1),
		VRFOutput:      output,
		VRFProof:       proof,
	}
	testBabeSecondaryPlainPreDigest := types.BabeSecondaryPlainPreDigest{
		AuthorityIndex: 1,
		SlotNumber:     uint64(1),
	}
	testBabeSecondaryVRFPreDigest := types.BabeSecondaryVRFPreDigest{
		AuthorityIndex: 1,
		SlotNumber:     uint64(1),
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
	encTestDigest, err := newEncodedBabeDigest(t, types.BabePrimaryPreDigest{AuthorityIndex: 0})
	assert.NoError(t, err)

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
	testParentHeader := createNewTestHeader(t, *testParentPrd)

	testParentHash, err := encodeAndHashHeader(t, testParentHeader)
	assert.NoError(t, err)

	testSecondaryPrd, err := testBabeSecondaryPlainPreDigest.ToPreRuntimeDigest()
	assert.NoError(t, err)
	testSecPlainHeader := createNewTestHeader(t, *testSecondaryPrd)
	testSecPlainHeader.ParentHash = testParentHash

	// Secondary Vrf Test Header
	encParentVrfDigest, err := newEncodedBabeDigest(t, testBabeSecondaryVRFPreDigest)
	assert.NoError(t, err)
	testParentVrfHeader := createNewTestHeader(t, *types.NewBABEPreRuntimeDigest(encParentVrfDigest))

	testVrfParentHash, err := encodeAndHashHeader(t, testParentVrfHeader)
	assert.NoError(t, err)

	encVrfHeader, err := newEncodedBabeDigest(t, testBabeSecondaryVRFPreDigest)
	assert.NoError(t, err)
	testSecVrfHeader := createNewTestHeader(t, *types.NewBABEPreRuntimeDigest(encVrfHeader))
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
	header0 := createNewTestHeader(t, testInvalidSeal, testInvalidSeal)

	// Case 1: Last element not seal
	header1 := createNewTestHeader(t, testInvalidPreRuntimeDigest, testInvalidPreRuntimeDigest)

	// Case 2: Fail to verify preruntime digest
	header2 := createNewTestHeader(t, testInvalidPreRuntimeDigest, testInvalidSeal)

	// Case 3: Invalid Seal Length
	babePrd, err := testBabePrimaryPreDigest.ToPreRuntimeDigest()
	assert.NoError(t, err)
	header3 := createNewTestHeader(t, *babePrd, testInvalidSeal)

	babeVerifier, err := newTestVerifier(t, kp, mockBlockState, scale.MaxUint128, false)
	assert.NoError(t, err)

	// Case 4: Invalid signature - BabePrimaryPreDigest
	babePrd2, err := testBabePrimaryPreDigest.ToPreRuntimeDigest()
	assert.NoError(t, err)
	header4 := createNewTestHeader(t, *babePrd2)

	err = signAndAddSeal(t, kp, header4, []byte{1})
	assert.NoError(t, err)

	babeVerifier2, err := newTestVerifier(t, kp, mockBlockState, scale.MaxUint128, false)
	assert.NoError(t, err)

	// Case 5: Invalid signature - BabeSecondaryPlainPreDigest
	babeSecPlainPrd, err := testBabeSecondaryPlainPreDigest.ToPreRuntimeDigest()
	assert.NoError(t, err)
	header5 := createNewTestHeader(t, *babeSecPlainPrd)

	err = signAndAddSeal(t, kp, header5, []byte{1})
	assert.NoError(t, err)

	babeVerifier3, err := newTestVerifier(t, kp, mockBlockState, scale.MaxUint128, true)
	assert.NoError(t, err)

	// Case 6: Invalid signature - BabeSecondaryVrfPreDigest
	encSecVrfDigest, err := newEncodedBabeDigest(t, testBabeSecondaryVRFPreDigest)
	assert.NoError(t, err)
	header6 := createNewTestHeader(t, *types.NewBABEPreRuntimeDigest(encSecVrfDigest))

	err = signAndAddSeal(t, kp, header6, []byte{1})
	assert.NoError(t, err)

	babeVerifier4, err := newTestVerifier(t, kp, mockBlockState, scale.MaxUint128, true)
	assert.NoError(t, err)

	// Case 7: GetAuthorityIndex Err
	babeParentPrd, err := testBabePrimaryPreDigest.ToPreRuntimeDigest()
	assert.NoError(t, err)
	babeParentHeader := createNewTestHeader(t, *babeParentPrd)

	parentHash, err := encodeAndHashHeader(t, babeParentHeader)
	assert.NoError(t, err)

	babePrd3, err := testBabePrimaryPreDigest.ToPreRuntimeDigest()
	assert.NoError(t, err)

	header7 := createNewTestHeader(t, *babePrd3)
	header7.ParentHash = parentHash

	hash, err := encodeAndHashHeader(t, header7)
	assert.NoError(t, err)

	err = signAndAddSeal(t, kp, header7, hash[:])
	assert.NoError(t, err)

	babeVerifier5, err := newTestVerifier(t, kp, mockBlockState, scale.MaxUint128, false)
	assert.NoError(t, err)

	//// Case 8: Get header error
	babeVerifier6, err := newTestVerifier(t, kp, mockBlockStateErr, scale.MaxUint128, false)
	assert.NoError(t, err)

	// Case 9: Equivocate case primary
	babeVerifier7, err := newTestVerifier(t, kp, mockBlockStateEquiv1, scale.MaxUint128, false)
	assert.NoError(t, err)

	// Case 10: Equivocate case secondary
	babeSecPlainPrd2, err := testBabeSecondaryPlainPreDigest.ToPreRuntimeDigest()
	assert.NoError(t, err)
	header8 := createNewTestHeader(t, *babeSecPlainPrd2)

	hash2, err := encodeAndHashHeader(t, header8)
	assert.NoError(t, err)

	err = signAndAddSeal(t, kp, header8, hash2[:])
	assert.NoError(t, err)

	babeVerifier8, err := newTestVerifier(t, kp, mockBlockStateEquiv2, scale.MaxUint128, true)
	assert.NoError(t, err)

	// Case 11: equivocation case for secondary VRF
	encVrfDigest, err := newEncodedBabeDigest(t, testBabeSecondaryVRFPreDigest)
	assert.NoError(t, err)
	header9 := createNewTestHeader(t, *types.NewBABEPreRuntimeDigest(encVrfDigest))

	hash3, err := encodeAndHashHeader(t, header9)
	assert.NoError(t, err)

	err = signAndAddSeal(t, kp, header9, hash3[:])
	assert.NoError(t, err)

	babeVerifier9, err := newTestVerifier(t, kp, mockBlockStateEquiv3, scale.MaxUint128, true)
	assert.NoError(t, err)

	type args struct {
		header *types.Header
	}
	tests := []struct {
		name     string
		verifier verifier
		args     args
		expErr   error
	}{
		{
			name:     "missing digest",
			verifier: verifier{},
			args:     args{types.NewEmptyHeader()},
			expErr:   errors.New("block header is missing digest items"),
		},
		{
			name:     "first digest invalid",
			verifier: verifier{},
			args:     args{header0},
			expErr:   errors.New("first digest item is not pre-digest"),
		},
		{
			name:     "last digest invalid",
			verifier: verifier{},
			args:     args{header1},
			expErr:   errors.New("last digest item is not seal"),
		},
		{
			name:     "invalid preruntime digest data",
			verifier: verifier{},
			args:     args{header2},
			expErr:   errors.New("failed to verify pre-runtime digest: EOF, field: 0"),
		},
		{
			name:     "invalid seal length",
			verifier: *babeVerifier,
			args:     args{header3},
			expErr:   errors.New("invalid signature length"),
		},
		{
			name:     "invalid seal signature - primary",
			verifier: *babeVerifier2,
			args:     args{header4},
			expErr:   ErrBadSignature,
		},
		{
			name:     "invalid seal signature - secondary plain",
			verifier: *babeVerifier3,
			args:     args{header5},
			expErr:   ErrBadSignature,
		},
		{
			name:     "invalid seal signature - secondary vrf",
			verifier: *babeVerifier4,
			args:     args{header6},
			expErr:   ErrBadSignature,
		},
		{
			name:     "valid digest items, getAuthorityIndex error",
			verifier: *babeVerifier5,
			args:     args{header7},
		},
		{
			name:     "get header err",
			verifier: *babeVerifier6,
			args:     args{header7},
		},
		{
			name:     "equivocate - primary",
			verifier: *babeVerifier7,
			args:     args{header7},
			expErr:   ErrProducerEquivocated,
		},
		{
			name:     "equivocate - secondary plain",
			verifier: *babeVerifier8,
			args:     args{header8},
			expErr:   ErrProducerEquivocated,
		},
		{
			name:     "equivocate - secondary vrf",
			verifier: *babeVerifier9,
			args:     args{header9},
			expErr:   ErrProducerEquivocated,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &tt.verifier
			err := b.verifyAuthorshipRight(tt.args.header)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErr.Error())
			} else {
				assert.NoError(t, err)
			}

		})
	}
}
