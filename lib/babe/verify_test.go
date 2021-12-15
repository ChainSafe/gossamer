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

/*
TODO for this test:
- equivocate cases
- think about why we dont handle errors
- fix naming
- Can I clean this test up? Helper funcs?
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

	//BabePrimaryPreDigest case
	output, proof, err := kp.VrfSign(makeTranscript(Randomness{}, uint64(1), 1))
	assert.NoError(t, err)

	// Setup test headers

	//Primary
	babeDigest := types.NewBabeDigest()
	err = babeDigest.Set(types.BabePrimaryPreDigest{AuthorityIndex: 0})
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

	// Secondary plain
	parDigestTest := types.BabeSecondaryPlainPreDigest{
		AuthorityIndex: 1,
		SlotNumber:     uint64(1),
	}
	prdParTest, err := parDigestTest.ToPreRuntimeDigest()
	assert.NoError(t, err)

	parHeaderTest := types.NewEmptyHeader()
	err = parHeaderTest.Digest.Add(*prdParTest)
	assert.NoError(t, err)

	encParTest, err := scale.Marshal(*parHeaderTest)
	assert.NoError(t, err)

	parentHashTest, err := common.Blake2bHash(encParTest)
	assert.NoError(t, err)

	digestSecPlain := types.BabeSecondaryPlainPreDigest{
		AuthorityIndex: 1,
		SlotNumber:     uint64(1),
	}

	secPrd, err := digestSecPlain.ToPreRuntimeDigest()
	assert.NoError(t, err)
	headerSecPlain := types.NewEmptyHeader()
	err = headerSecPlain.Digest.Add(*secPrd)
	assert.NoError(t, err)
	headerSecPlain.ParentHash = parentHashTest

	// Secondary Vrf
	headerb := types.NewEmptyHeader()
	secVRFDigestb := types.BabeSecondaryVRFPreDigest{
		AuthorityIndex: 1,
		SlotNumber:     uint64(1),
		VrfOutput:      output,
		VrfProof:       proof,
	}

	digestSecondaryVRFb := types.NewBabeDigest()
	err = digestSecondaryVRFb.Set(secVRFDigestb)
	assert.NoError(t, err)

	bdEnc2b, err := scale.Marshal(digestSecondaryVRFb)
	require.NoError(t, err)

	prdb := types.NewBABEPreRuntimeDigest(bdEnc2b)
	err = headerb.Digest.Add(*prdb)
	assert.NoError(t, err)

	encParTest2, err := scale.Marshal(*headerb)
	assert.NoError(t, err)

	parentHashTest2, err := common.Blake2bHash(encParTest2)
	assert.NoError(t, err)

	secVRFDigestc := types.BabeSecondaryVRFPreDigest{
		AuthorityIndex: 1,
		SlotNumber:     uint64(1),
		VrfOutput:      output,
		VrfProof:       proof,
	}

	digestSecondaryVRFc := types.NewBabeDigest()
	err = digestSecondaryVRFc.Set(secVRFDigestc)
	assert.NoError(t, err)

	headerSecVrf := types.NewEmptyHeader()
	bdEncb, err := scale.Marshal(digestSecondaryVRFc)
	require.NoError(t, err)

	prdc := types.NewBABEPreRuntimeDigest(bdEncb)
	err = headerSecVrf.Digest.Add(*prdc)
	assert.NoError(t, err)
	headerSecPlain.ParentHash = parentHashTest2

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
		Return(headerPrimary, nil)

	mockBlockStateEquiv2.
		EXPECT().
		GetAllBlocksAtDepth(gomock.Any()).
		Return(h1)
	mockBlockStateEquiv2.
		EXPECT().
		GetHeader(h).
		Return(headerSecPlain, nil)
	mockBlockStateEquiv3.
		EXPECT().
		GetAllBlocksAtDepth(gomock.Any()).
		Return(h1)
	mockBlockStateEquiv3.
		EXPECT().
		GetHeader(h).
		Return(headerSecVrf, nil)


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

	// Last element not seal
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

	//Fail to verify preruntime digest
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

	// Invalid Seal Length
	header3 := types.NewEmptyHeader()
	secDigest0 := types.BabePrimaryPreDigest{
		AuthorityIndex: 0,
		SlotNumber:     uint64(1),
		VRFOutput:      output,
		VRFProof:       proof,
	}
	prd, err := secDigest0.ToPreRuntimeDigest()
	assert.NoError(t, err)
	err = header3.Digest.Add(*prd)
	assert.NoError(t, err)
	err = header3.Digest.Add(types.SealDigest{
		ConsensusEngineID: types.BabeEngineID,
		Data:              []byte{1},
	})
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

	// Invalid signature - BabePrimaryPreDigest
	header4 := types.NewEmptyHeader()
	secDigest1 := types.BabePrimaryPreDigest{
		AuthorityIndex: 0,
		SlotNumber:     uint64(1),
		VRFOutput:      output,
		VRFProof:       proof,
	}
	prd2, err := secDigest1.ToPreRuntimeDigest()
	assert.NoError(t, err)
	err = header4.Digest.Add(*prd2)
	assert.NoError(t, err)

	sig, err := kp.Sign([]byte{1})
	assert.NoError(t, err)
	err = header4.Digest.Add(types.SealDigest{
		ConsensusEngineID: types.BabeEngineID,
		Data:              sig,
	})
	assert.NoError(t, err)

	auth2 := types.NewAuthority(kp.Public(), uint64(1))
	vi2 := &verifierInfo{
		authorities:    []types.Authority{*auth2, *auth2},
		randomness:     Randomness{},
		threshold:      scale.MaxUint128,
		secondarySlots: false,
	}

	v2, err := newVerifier(mockBlockState, 1, vi2)
	assert.NoError(t, err)

	// Invalid signature - BabeSecondaryPlainPreDigest
	header6 := types.NewEmptyHeader()
	priDigest1 := types.BabeSecondaryPlainPreDigest{
		AuthorityIndex: 1,
		SlotNumber:     uint64(1),
	}

	prd6, err := priDigest1.ToPreRuntimeDigest()
	assert.NoError(t, err)
	err = header6.Digest.Add(*prd6)
	assert.NoError(t, err)

	sig6, err := kp.Sign([]byte{1})
	assert.NoError(t, err)
	err = header6.Digest.Add(types.SealDigest{
		ConsensusEngineID: types.BabeEngineID,
		Data:              sig6,
	})
	assert.NoError(t, err)

	auth6 := types.NewAuthority(kp.Public(), uint64(1))
	vi6 := &verifierInfo{
		authorities:    []types.Authority{*auth6, *auth6},
		randomness:     Randomness{},
		threshold:      scale.MaxUint128,
		secondarySlots: true,
	}

	v6, err := newVerifier(mockBlockState, 1, vi6)
	assert.NoError(t, err)

	// Invalid signature - BabeSecondaryVrfPreDigest
	header7 := types.NewEmptyHeader()
	secVRFDigest := types.BabeSecondaryVRFPreDigest{
		AuthorityIndex: 1,
		SlotNumber:     uint64(1),
		VrfOutput:      output,
		VrfProof:       proof,
	}

	digestSecondaryVRF := types.NewBabeDigest()
	err = digestSecondaryVRF.Set(secVRFDigest)
	assert.NoError(t, err)

	bdEnc2, err := scale.Marshal(digestSecondaryVRF)
	require.NoError(t, err)

	prd7 := types.NewBABEPreRuntimeDigest(bdEnc2)
	err = header7.Digest.Add(*prd7)
	assert.NoError(t, err)

	sig7, err := kp.Sign([]byte{1})
	assert.NoError(t, err)
	err = header7.Digest.Add(types.SealDigest{
		ConsensusEngineID: types.BabeEngineID,
		Data:              sig7,
	})
	assert.NoError(t, err)

	auth7 := types.NewAuthority(kp.Public(), uint64(1))
	vi7 := &verifierInfo{
		authorities:    []types.Authority{*auth7, *auth7},
		randomness:     Randomness{},
		threshold:      scale.MaxUint128,
		secondarySlots: true,
	}

	v7, err := newVerifier(mockBlockState, 1, vi7)
	assert.NoError(t, err)

	//GetAuthorityIndex Err
	parDigest := types.BabePrimaryPreDigest{
		AuthorityIndex: 1,
		SlotNumber:     uint64(1),
		VRFOutput:      output,
		VRFProof:       proof,
	}
	prdPar, err := parDigest.ToPreRuntimeDigest()
	assert.NoError(t, err)

	parHeader := types.NewEmptyHeader()
	err = parHeader.Digest.Add(*prdPar)
	assert.NoError(t, err)

	encPar, err := scale.Marshal(*parHeader)
	assert.NoError(t, err)

	parentHash, err := common.Blake2bHash(encPar)
	assert.NoError(t, err)

	header5 := types.NewEmptyHeader()
	header5.ParentHash = parentHash
	secDigest2 := types.BabePrimaryPreDigest{
		AuthorityIndex: 0,
		SlotNumber:     uint64(1),
		VRFOutput:      output,
		VRFProof:       proof,
	}
	prd3, err := secDigest2.ToPreRuntimeDigest()
	assert.NoError(t, err)
	err = header5.Digest.Add(*prd3)
	assert.NoError(t, err)

	encHeader, err := scale.Marshal(*header5)
	assert.NoError(t, err)

	hash, err := common.Blake2bHash(encHeader)
	assert.NoError(t, err)

	sig2, err := kp.Sign(hash[:])
	assert.NoError(t, err)

	seal := types.SealDigest{
		ConsensusEngineID: types.BabeEngineID,
		Data:              sig2,
	}
	err = header5.Digest.Add(seal)
	assert.NoError(t, err)

	auth3 := types.NewAuthority(kp.Public(), uint64(1))
	vi3 := &verifierInfo{
		authorities:    []types.Authority{*auth3, *auth3},
		randomness:     Randomness{},
		threshold:      scale.MaxUint128,
		secondarySlots: false,
	}

	v3, err := newVerifier(mockBlockState, 1, vi3)
	assert.NoError(t, err)

	//// Get header error
	v4, err := newVerifier(mockBlockStateErr, 1, vi3)
	assert.NoError(t, err)

	// Equivocate case
	v5, err := newVerifier(mockBlockStateEquiv1, 1, vi3)
	assert.NoError(t, err)

	// Equivocate case secondary
	header9 := types.NewEmptyHeader()
	priDigest9 := types.BabeSecondaryPlainPreDigest{
		AuthorityIndex: 1,
		SlotNumber:     uint64(1),
	}

	prd9, err := priDigest9.ToPreRuntimeDigest()
	assert.NoError(t, err)
	err = header9.Digest.Add(*prd9)
	assert.NoError(t, err)

	encHeader9, err := scale.Marshal(*header9)
	assert.NoError(t, err)

	hash9, err := common.Blake2bHash(encHeader9)
	assert.NoError(t, err)

	sig9, err := kp.Sign(hash9[:])
	assert.NoError(t, err)

	seal9 := types.SealDigest{
		ConsensusEngineID: types.BabeEngineID,
		Data:              sig9,
	}
	err = header9.Digest.Add(seal9)
	assert.NoError(t, err)

	auth9 := types.NewAuthority(kp.Public(), uint64(1))
	vi9 := &verifierInfo{
		authorities:    []types.Authority{*auth9, *auth9},
		randomness:     Randomness{},
		threshold:      scale.MaxUint128,
		secondarySlots: true,
	}

	v9, err := newVerifier(mockBlockStateEquiv2, 1, vi9)
	assert.NoError(t, err)

	// TODO add equivocation case for secondary VRF
	header10 := types.NewEmptyHeader()
	secVRFDigest1 := types.BabeSecondaryVRFPreDigest{
		AuthorityIndex: 1,
		SlotNumber:     uint64(1),
		VrfOutput:      output,
		VrfProof:       proof,
	}

	digestSecondaryVRF1 := types.NewBabeDigest()
	err = digestSecondaryVRF1.Set(secVRFDigest1)
	assert.NoError(t, err)

	bdEnc2a, err := scale.Marshal(digestSecondaryVRF1)
	require.NoError(t, err)

	prd10 := types.NewBABEPreRuntimeDigest(bdEnc2a)
	err = header10.Digest.Add(*prd10)
	assert.NoError(t, err)

	encHeader10, err := scale.Marshal(*header10)
	assert.NoError(t, err)

	hash10, err := common.Blake2bHash(encHeader10)
	assert.NoError(t, err)

	sig10, err := kp.Sign(hash10[:])
	assert.NoError(t, err)
	err = header10.Digest.Add(types.SealDigest{
		ConsensusEngineID: types.BabeEngineID,
		Data:              sig10,
	})
	assert.NoError(t, err)

	auth10 := types.NewAuthority(kp.Public(), uint64(1))
	vi10 := &verifierInfo{
		authorities:    []types.Authority{*auth10, *auth10},
		randomness:     Randomness{},
		threshold:      scale.MaxUint128,
		secondarySlots: true,
	}

	v10, err := newVerifier(mockBlockStateEquiv3, 1, vi10)
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
			verifier: *v,
			args:     args{header3},
			expErr:   errors.New("invalid signature length"),
		},
		{
			name:     "invalid seal signature - primary",
			verifier: *v2,
			args:     args{header4},
			expErr:   ErrBadSignature,
		},
		{
			name:     "invalid seal signature - secondary plain",
			verifier: *v6,
			args:     args{header6},
			expErr:   ErrBadSignature,
		},
		{
			name:     "invalid seal signature - secondary vrf",
			verifier: *v7,
			args:     args{header7},
			expErr:   ErrBadSignature,
		},
		{
			name:     "valid digest items, getAuthorityIndex error",
			verifier: *v3,
			args:     args{header5},
		},
		{
			name:     "get header err",
			verifier: *v4,
			args:     args{header5},
		},
		{
			name:     "equivocate - primary",
			verifier: *v5,
			args:     args{header5},
			expErr:   ErrProducerEquivocated,
		},
		{
			name:     "equivocate - secondary plain",
			verifier: *v9,
			args:     args{header9},
			expErr:   ErrProducerEquivocated,
		},
		{
			name:     "equivocate - secondary vrf",
			verifier: *v10,
			args:     args{header10},
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
