// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package babe

import (
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"

	"github.com/stretchr/testify/assert"
)

func Test_getSecondarySlotAuthor(t *testing.T) {
	type args struct {
		slot       uint64
		numAuths   int
		randomness Randomness
	}
	tests := []struct {
		name   string
		args   args
		exp    uint32
		expErr error
	}{
		{
			name: "happy path",
			args: args{
				slot:     21,
				numAuths: 21,
			},
			exp: 14,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := getSecondarySlotAuthor(tt.args.slot, tt.args.numAuths, tt.args.randomness)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.exp, res)
		})
	}
}

func Test_verifySecondarySlotPlain(t *testing.T) {
	type args struct {
		authorityIndex uint32
		slot           uint64
		numAuths       int
		randomness     Randomness
	}
	tests := []struct {
		name   string
		args   args
		expErr error
	}{
		{
			name: "happy path",
			args: args{
				authorityIndex: 14,
				slot:           21,
				numAuths:       21,
			},
		},
		{
			name: "err path",
			args: args{
				authorityIndex: 13,
				slot:           21,
				numAuths:       21,
			},
			expErr: ErrBadSecondarySlotClaim,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := verifySecondarySlotPlain(tt.args.authorityIndex, tt.args.slot, tt.args.numAuths, tt.args.randomness)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_verifySecondarySlotVRF(t *testing.T) {
	kp, err := sr25519.GenerateKeypair()
	assert.NoError(t, err)

	transcript := makeTranscript(Randomness{}, 77, 0)
	out, proof, err := kp.VrfSign(transcript)
	assert.NoError(t, err)

	type args struct {
		digest     *types.BabeSecondaryVRFPreDigest
		pk         *sr25519.PublicKey
		epoch      uint64
		numAuths   int
		randomness Randomness
	}
	tests := []struct {
		name   string
		args   args
		exp    bool
		expErr error
	}{
		{
			name: "happy path",
			args: args{
				digest:   types.NewBabeSecondaryVRFPreDigest(0, 77, out, proof),
				pk:       kp.Public().(*sr25519.PublicKey),
				numAuths: 1,
			},
			exp: true,
		},
		{
			name: "err ErrBadSecondarySlotClaim",
			args: args{
				digest:   types.NewBabeSecondaryVRFPreDigest(3, 77, out, proof),
				pk:       kp.Public().(*sr25519.PublicKey),
				epoch:    77,
				numAuths: 1,
			},
			expErr: ErrBadSecondarySlotClaim,
		},
		{
			name: "false path",
			args: args{
				digest:   types.NewBabeSecondaryVRFPreDigest(0, 77, out, proof),
				pk:       kp.Public().(*sr25519.PublicKey),
				epoch:    77,
				numAuths: 1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := verifySecondarySlotVRF(tt.args.digest, tt.args.pk, tt.args.epoch, tt.args.numAuths, tt.args.randomness)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.exp, res)
		})
	}
}
