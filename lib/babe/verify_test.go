// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package babe

import (
	"errors"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/golang/mock/gomock"
	"github.com/ChainSafe/gossamer/lib/babe/mocks"
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
		name    string
		args    args
		exp    uint32
		expErr error
	}{
		{
			name: "No Digest",
			args: args{types.NewEmptyHeader()},
			expErr: errors.New("no digest provided"),
		},
		{
			name: "First Digest Invalid Type",
			args: args{headerNoPre},
			expErr: errors.New("first digest item is not pre-runtime digest"),
		},
		{
			name: "Invalid Preruntime Digest Type",
			args: args{headerInvalidPre},
			expErr: errors.New("cannot decode babe header from pre-digest: EOF, field: 0"),
		},
		{
			name: "BabePrimaryPreDigest Type",
			args: args{headerPrimary},
			exp: 21,
		},
		{
			name: "BabeSecondaryVRFPreDigest Type",
			args: args{headerSecondary},
			exp: 21,
		},
		{
			name: "BabeSecondaryPlainPreDigest Type",
			args: args{headerSecondaryPlain},
			exp: 21,
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
		name    string
		verifier  verifier
		args    args
		exp    bool
		expErr error
	}{
		{
			name: "Over threshold",
			verifier: *v,
			args: args{
				authorityIndex: 0,
				slot: uint64(1),
				vrfOutput: [32]byte{},
				vrfProof: [64]byte{},
			},
			expErr: ErrVRFOutputOverThreshold,
		},
		{
			name: "VRF not verified",
			verifier: *v1,
			args: args{
				authorityIndex: 0,
				slot: uint64(1),
				vrfOutput: [32]byte{},
				vrfProof: [64]byte{},
			},
		},
		{
			name: "VRF verified",
			verifier: *v1,
			args: args{
				authorityIndex: 0,
				slot: uint64(1),
				vrfOutput: output,
				vrfProof: proof,
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
		SlotNumber: uint64(1),
		VRFOutput: output,
		VRFProof: proof,
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
		SlotNumber: uint64(1),
		VrfOutput: output,
		VrfProof: proof,
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
		name    string
		verifier  verifier
		args    args
		exp    scale.VaryingDataTypeValue
		expErr error
	}{
		{
			name: "Invalid PreRuntimeDigest",
			verifier: verifier{},
			args: args{&types.PreRuntimeDigest{Data: []byte{0}}},
			expErr: errors.New("unable to find VaryingDataTypeValue with index: 0"),
		},
		{
			name: "Invalid BlockProducer Index",
			verifier: verifier{},
			args: args{prd},
			expErr: ErrInvalidBlockProducerIndex,
		},
		{
			name: "BabePrimaryPreDigest Case OK",
			verifier: *v,
			args: args{prd1},
			exp: types.BabePrimaryPreDigest{
				AuthorityIndex: 0,
				SlotNumber: uint64(1),
				VRFOutput: output,
				VRFProof: proof,
			},
		},
		{
			name: "BabePrimaryPreDigest Case output over threshold",
			verifier: *v1,
			args: args{prd1},
			expErr: errors.New("vrf output over threshold"),
		},
		{
			name: "BabePrimaryPreDigest Case Invalid",
			verifier: *v2,
			args: args{prd1},
			expErr: ErrBadSlotClaim,
		},
		{
			name: "BabeSecondaryPlainPreDigest SecondarySlot false",
			verifier: *vSec,
			args: args{prd},
			expErr: ErrBadSlotClaim,
		},
		{
			name: "BabeSecondaryPlainPreDigest invalid claim",
			verifier: *vSec2,
			args: args{prd},
			expErr: errors.New("invalid secondary slot claim"),
		},
		{
			name: "BabeSecondaryVRFPreDigest SecondarySlot false",
			verifier: *vVRFSec,
			args: args{babePRD},
			expErr: ErrBadSlotClaim,
		},
		{
			name: "BabeSecondaryVRFPreDigest invalid claim",
			verifier: *vVRFSec2,
			args: args{babePRD},
			expErr: errors.New("invalid secondary slot claim"),
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