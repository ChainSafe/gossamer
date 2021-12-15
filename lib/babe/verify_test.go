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

func newTestVerifier(t *testing.T, kp *sr25519.Keypair, blockState BlockState, threshold *scale.Uint128, secSlots bool) (*verifier, error){
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
- fix naming
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


	// Primary Test Header
	babeTestDigest := types.NewBabeDigest()
	err = babeTestDigest.Set(types.BabePrimaryPreDigest{AuthorityIndex: 0})
	assert.NoError(t, err)

	encTestDigest, err := scale.Marshal(babeTestDigest)
	require.NoError(t, err)

	testDigestPrimary := types.NewDigest()
	err = testDigestPrimary.Add(types.PreRuntimeDigest{
		ConsensusEngineID: types.BabeEngineID,
		Data:              encTestDigest,
	})
	assert.NoError(t, err)
	testHeaderPrimary := types.NewEmptyHeader()
	testHeaderPrimary.Digest = testDigestPrimary

	// Secondary Plain test header
	testParentDigest := types.BabeSecondaryPlainPreDigest{
		AuthorityIndex: 1,
		SlotNumber:     uint64(1),
	}
	testParentPrd, err := testParentDigest.ToPreRuntimeDigest()
	assert.NoError(t, err)

	testParentHeader := types.NewEmptyHeader()
	err = testParentHeader.Digest.Add(*testParentPrd)
	assert.NoError(t, err)

	encParentHeader, err := scale.Marshal(*testParentHeader)
	assert.NoError(t, err)

	testParentHash, err := common.Blake2bHash(encParentHeader)
	assert.NoError(t, err)

	testSecondaryDigest := types.BabeSecondaryPlainPreDigest{
		AuthorityIndex: 1,
		SlotNumber:     uint64(1),
	}

	testSecondaryPrd, err := testSecondaryDigest.ToPreRuntimeDigest()
	assert.NoError(t, err)
	testSecPlainHeader := types.NewEmptyHeader()
	err = testSecPlainHeader.Digest.Add(*testSecondaryPrd)
	assert.NoError(t, err)
	testSecPlainHeader.ParentHash = testParentHash

	// Secondary Vrf Test header
	testParentVrfHeader := types.NewEmptyHeader()
	testParentVrfDigest := types.BabeSecondaryVRFPreDigest{
		AuthorityIndex: 1,
		SlotNumber:     uint64(1),
		VrfOutput:      output,
		VrfProof:       proof,
	}

	testParentBabeDigest := types.NewBabeDigest()
	err = testParentBabeDigest.Set(testParentVrfDigest)
	assert.NoError(t, err)

	encParentVrfDigest, err := scale.Marshal(testParentBabeDigest)
	require.NoError(t, err)

	testBabePrd := types.NewBABEPreRuntimeDigest(encParentVrfDigest)
	err = testParentVrfHeader.Digest.Add(*testBabePrd)
	assert.NoError(t, err)

	encParentVrfHeader, err := scale.Marshal(*testParentVrfHeader)
	assert.NoError(t, err)

	testVrfParentHash, err := common.Blake2bHash(encParentVrfHeader)
	assert.NoError(t, err)

	testSecondaryVrfDigest := types.BabeSecondaryVRFPreDigest{
		AuthorityIndex: 1,
		SlotNumber:     uint64(1),
		VrfOutput:      output,
		VrfProof:       proof,
	}

	testBabeVrfDigest := types.NewBabeDigest()
	err = testBabeVrfDigest.Set(testSecondaryVrfDigest)
	assert.NoError(t, err)

	testSecVrfHeader := types.NewEmptyHeader()
	encVrfHeader, err := scale.Marshal(testBabeVrfDigest)
	require.NoError(t, err)

	testBabeVrfPrd := types.NewBABEPreRuntimeDigest(encVrfHeader)
	err = testSecVrfHeader.Digest.Add(*testBabeVrfPrd)
	assert.NoError(t, err)
	testSecVrfHeader.ParentHash = testVrfParentHash

	h := common.MustHexToHash("0x01")
	h1 := []common.Hash{h}
	mockBlockState.
		EXPECT().
		GetAllBlocksAtDepth(gomock.Any()).
		Return(h1)
	mockBlockState.
		EXPECT().
		GetHeader(h).
		Return(types.NewEmptyHeader(), nil)

	mockBlockStateErr.
		EXPECT().
		GetAllBlocksAtDepth(gomock.Any()).
		Return(h1)
	mockBlockStateErr.
		EXPECT().
		GetHeader(h).
		Return(nil, errors.New("get header error"))

	mockBlockStateEquiv1.
		EXPECT().
		GetAllBlocksAtDepth(gomock.Any()).
		Return(h1)
	mockBlockStateEquiv1.
		EXPECT().
		GetHeader(h).
		Return(testHeaderPrimary, nil)

	mockBlockStateEquiv2.
		EXPECT().
		GetAllBlocksAtDepth(gomock.Any()).
		Return(h1)
	mockBlockStateEquiv2.
		EXPECT().
		GetHeader(h).
		Return(testSecPlainHeader, nil)
	mockBlockStateEquiv3.
		EXPECT().
		GetAllBlocksAtDepth(gomock.Any()).
		Return(h1)
	mockBlockStateEquiv3.
		EXPECT().
		GetHeader(h).
		Return(testSecVrfHeader, nil)


	// First element not preruntime digest
	header0 := types.NewEmptyHeader()
	err = header0.Digest.Add(types.SealDigest{
		ConsensusEngineID: types.BabeEngineID,
		Data:              []byte{1},
	})
	assert.NoError(t, err)
	err = header0.Digest.Add(types.SealDigest{
		ConsensusEngineID: types.BabeEngineID,
		Data:              []byte{1},
	})
	assert.NoError(t, err)

	// Case 1: Last element not seal
	header1 := types.NewEmptyHeader()
	err = header1.Digest.Add(types.PreRuntimeDigest{
		ConsensusEngineID: types.BabeEngineID,
		Data:              []byte{1},
	})
	assert.NoError(t, err)
	err = header1.Digest.Add(types.PreRuntimeDigest{
		ConsensusEngineID: types.BabeEngineID,
		Data:              []byte{1},
	})
	assert.NoError(t, err)

	// Case 2: Fail to verify preruntime digest
	header2 := types.NewEmptyHeader()
	err = header2.Digest.Add(types.PreRuntimeDigest{
		ConsensusEngineID: types.BabeEngineID,
		Data:              []byte{1},
	})
	assert.NoError(t, err)
	err = header2.Digest.Add(types.SealDigest{
		ConsensusEngineID: types.BabeEngineID,
		Data:              []byte{1},
	})
	assert.NoError(t, err)

	// Case 3: Invalid Seal Length
	header3 := types.NewEmptyHeader()
	babePrimaryDigest := types.BabePrimaryPreDigest{
		AuthorityIndex: 0,
		SlotNumber:     uint64(1),
		VRFOutput:      output,
		VRFProof:       proof,
	}
	babePrd, err := babePrimaryDigest.ToPreRuntimeDigest()
	assert.NoError(t, err)
	err = header3.Digest.Add(*babePrd)
	assert.NoError(t, err)
	err = header3.Digest.Add(types.SealDigest{
		ConsensusEngineID: types.BabeEngineID,
		Data:              []byte{1},
	})
	assert.NoError(t, err)

	babeVerifier, err := newTestVerifier(t, kp, mockBlockState, scale.MaxUint128, false)
	assert.NoError(t, err)

	// Case 4: Invalid signature - BabePrimaryPreDigest
	header4 := types.NewEmptyHeader()
	babePrimaryDigest2 := types.BabePrimaryPreDigest{
		AuthorityIndex: 0,
		SlotNumber:     uint64(1),
		VRFOutput:      output,
		VRFProof:       proof,
	}
	babePrd2, err := babePrimaryDigest2.ToPreRuntimeDigest()
	assert.NoError(t, err)
	err = header4.Digest.Add(*babePrd2)
	assert.NoError(t, err)

	sig, err := kp.Sign([]byte{1})
	assert.NoError(t, err)
	err = header4.Digest.Add(types.SealDigest{
		ConsensusEngineID: types.BabeEngineID,
		Data:              sig,
	})
	assert.NoError(t, err)

	babeVerifier2, err := newTestVerifier(t, kp, mockBlockState, scale.MaxUint128, false)
	assert.NoError(t, err)

	// Case 5: Invalid signature - BabeSecondaryPlainPreDigest
	header5 := types.NewEmptyHeader()
	babeSecondaryPlainDigest := types.BabeSecondaryPlainPreDigest{
		AuthorityIndex: 1,
		SlotNumber:     uint64(1),
	}

	babeSecPlainPrd, err := babeSecondaryPlainDigest.ToPreRuntimeDigest()
	assert.NoError(t, err)
	err = header5.Digest.Add(*babeSecPlainPrd)
	assert.NoError(t, err)

	sig2, err := kp.Sign([]byte{1})
	assert.NoError(t, err)
	err = header5.Digest.Add(types.SealDigest{
		ConsensusEngineID: types.BabeEngineID,
		Data:              sig2,
	})
	assert.NoError(t, err)

	babeVerifier3, err := newTestVerifier(t, kp, mockBlockState, scale.MaxUint128, true)
	assert.NoError(t, err)

	// Case 6: Invalid signature - BabeSecondaryVrfPreDigest
	header6 := types.NewEmptyHeader()
	babeSecondaryVrfPreDigest := types.BabeSecondaryVRFPreDigest{
		AuthorityIndex: 1,
		SlotNumber:     uint64(1),
		VrfOutput:      output,
		VrfProof:       proof,
	}

	babeSecondaryVrfDigest := types.NewBabeDigest()
	err = babeSecondaryVrfDigest.Set(babeSecondaryVrfPreDigest)
	assert.NoError(t, err)

	encSecVrfDigest, err := scale.Marshal(babeSecondaryVrfDigest)
	require.NoError(t, err)

	babeSecVrfPrd := types.NewBABEPreRuntimeDigest(encSecVrfDigest)
	err = header6.Digest.Add(*babeSecVrfPrd)
	assert.NoError(t, err)

	sig3, err := kp.Sign([]byte{1})
	assert.NoError(t, err)
	err = header6.Digest.Add(types.SealDigest{
		ConsensusEngineID: types.BabeEngineID,
		Data:              sig3,
	})
	assert.NoError(t, err)

	babeVerifier4, err := newTestVerifier(t, kp, mockBlockState, scale.MaxUint128, true)
	assert.NoError(t, err)

	// Case 7: GetAuthorityIndex Err
	babeParentPrimaryPreDigest := types.BabePrimaryPreDigest{
		AuthorityIndex: 1,
		SlotNumber:     uint64(1),
		VRFOutput:      output,
		VRFProof:       proof,
	}
	babeParentPrd, err := babeParentPrimaryPreDigest.ToPreRuntimeDigest()
	assert.NoError(t, err)

	babeParentHeader := types.NewEmptyHeader()
	err = babeParentHeader.Digest.Add(*babeParentPrd)
	assert.NoError(t, err)

	encParentHeader2, err := scale.Marshal(*babeParentHeader)
	assert.NoError(t, err)

	parentHash, err := common.Blake2bHash(encParentHeader2)
	assert.NoError(t, err)

	header7 := types.NewEmptyHeader()
	header7.ParentHash = parentHash
	babePrimaryPreDigest := types.BabePrimaryPreDigest{
		AuthorityIndex: 0,
		SlotNumber:     uint64(1),
		VRFOutput:      output,
		VRFProof:       proof,
	}
	babePrd3, err := babePrimaryPreDigest.ToPreRuntimeDigest()
	assert.NoError(t, err)
	err = header7.Digest.Add(*babePrd3)
	assert.NoError(t, err)

	encHeader, err := scale.Marshal(*header7)
	assert.NoError(t, err)

	hash, err := common.Blake2bHash(encHeader)
	assert.NoError(t, err)

	sig4, err := kp.Sign(hash[:])
	assert.NoError(t, err)

	seal := types.SealDigest{
		ConsensusEngineID: types.BabeEngineID,
		Data:              sig4,
	}
	err = header7.Digest.Add(seal)
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
	header8 := types.NewEmptyHeader()
	babeSecPlainDigest := types.BabeSecondaryPlainPreDigest{
		AuthorityIndex: 1,
		SlotNumber:     uint64(1),
	}

	babeSecPlainPrd2, err := babeSecPlainDigest.ToPreRuntimeDigest()
	assert.NoError(t, err)
	err = header8.Digest.Add(*babeSecPlainPrd2)
	assert.NoError(t, err)

	encHeader2, err := scale.Marshal(*header8)
	assert.NoError(t, err)

	hash2, err := common.Blake2bHash(encHeader2)
	assert.NoError(t, err)

	sig5, err := kp.Sign(hash2[:])
	assert.NoError(t, err)

	seal2 := types.SealDigest{
		ConsensusEngineID: types.BabeEngineID,
		Data:              sig5,
	}
	err = header8.Digest.Add(seal2)
	assert.NoError(t, err)

	babeVerifier8, err := newTestVerifier(t, kp, mockBlockStateEquiv2, scale.MaxUint128, true)
	assert.NoError(t, err)

	// Case 11: equivocation case for secondary VRF
	header9 := types.NewEmptyHeader()
	babeSecVrfDigest := types.BabeSecondaryVRFPreDigest{
		AuthorityIndex: 1,
		SlotNumber:     uint64(1),
		VrfOutput:      output,
		VrfProof:       proof,
	}

	babeDigest := types.NewBabeDigest()
	err = babeDigest.Set(babeSecVrfDigest)
	assert.NoError(t, err)

	encVrfDigest, err := scale.Marshal(babeDigest)
	require.NoError(t, err)

	babeSecVrfPrd2 := types.NewBABEPreRuntimeDigest(encVrfDigest)
	err = header9.Digest.Add(*babeSecVrfPrd2)
	assert.NoError(t, err)

	encHeader3, err := scale.Marshal(*header9)
	assert.NoError(t, err)

	hash3, err := common.Blake2bHash(encHeader3)
	assert.NoError(t, err)

	sig6, err := kp.Sign(hash3[:])
	assert.NoError(t, err)
	err = header9.Digest.Add(types.SealDigest{
		ConsensusEngineID: types.BabeEngineID,
		Data:              sig6,
	})
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
