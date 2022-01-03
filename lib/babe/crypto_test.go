// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package babe

import (
	"errors"

	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/assert"

	"testing"
)

func TestCalculateThreshold(t *testing.T) {
	type args struct {
		C1       uint64
		C2       uint64
		numAuths int
	}
	tests := []struct {
		name   string
		args   args
		exp    *scale.Uint128
		expErr error
	}{
		{
			name: "happy path",
			args: args{
				C1:       1,
				C2:       2,
				numAuths: 3,
			},
			exp: &scale.Uint128{Upper: 0x34d00ad6148e1800, Lower: 0x0},
		},
		{
			name: "0 value input",
			args: args{
				C1:       0,
				C2:       0,
				numAuths: 0,
			},
			expErr: ErrThresholdBothZero,
		},
		{
			name: "C1 > C2",
			args: args{
				C1:       5,
				C2:       2,
				numAuths: 0,
			},
			expErr: errors.New("invalid C1/C2: greater than 1"),
		},
		{
			name: "max threshold",
			args: args{
				C1:       2147483647,
				C2:       2147483647,
				numAuths: 3,
			},
			exp: scale.MaxUint128,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := CalculateThreshold(tt.args.C1, tt.args.C2, tt.args.numAuths)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.exp, res)
		})
	}
}

func Test_checkPrimaryThreshold(t *testing.T) {
	keyring, _ := keystore.NewSr25519Keyring()
	aliceKeypair := keyring.Alice().(*sr25519.Keypair)
	type args struct {
		randomness Randomness
		slot       uint64
		epoch      uint64
		output     [sr25519.VRFOutputLength]byte
		threshold  *scale.Uint128
		pub        *sr25519.PublicKey
	}
	tests := []struct {
		name   string
		args   args
		exp    bool
		expErr error
	}{
		{
			name: "happy path true",
			args: args{
				randomness: Randomness{},
				slot:       uint64(0),
				epoch:      uint64(0),
				output:     [32]byte{},
				threshold:  scale.MaxUint128,
				pub:        aliceKeypair.Public().(*sr25519.PublicKey),
			},
			exp: true,
		},
		{
			name: "happy path false",
			args: args{
				randomness: Randomness{},
				slot:       uint64(0),
				epoch:      uint64(0),
				output:     [32]byte{},
				threshold:  &scale.Uint128{},
				pub:        aliceKeypair.Public().(*sr25519.PublicKey),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := checkPrimaryThreshold(tt.args.randomness, tt.args.slot, tt.args.epoch,
				tt.args.output, tt.args.threshold, tt.args.pub)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.exp, res)
		})
	}
}

func Test_claimPrimarySlot(t *testing.T) {
	keyring, _ := keystore.NewSr25519Keyring()
	type args struct {
		randomness Randomness
		slot       uint64
		epoch      uint64
		threshold  *scale.Uint128
		keypair    *sr25519.Keypair
	}
	tests := []struct {
		name   string
		args   args
		exp    *VrfOutputAndProof
		expErr error
	}{
		{
			name: "authority not authorized",
			args: args{
				randomness: Randomness{},
				slot:       uint64(1),
				epoch:      uint64(2),
				threshold:  &scale.Uint128{},
				keypair:    keyring.Alice().(*sr25519.Keypair),
			},
		},
		{
			name: "authority authorized",
			args: args{
				randomness: Randomness{},
				slot:       uint64(1),
				epoch:      uint64(2),
				threshold:  scale.MaxUint128,
				keypair:    keyring.Alice().(*sr25519.Keypair),
			},
			exp: &VrfOutputAndProof{
				output: [32]uint8{0x80, 0xf0, 0x8a, 0x7d, 0xa1, 0x71, 0x77, 0xdc, 0x7, 0x7f, 0x6, 0xd5, 0xc1, 0x5d, 0x90,
					0x4f, 0x64, 0x21, 0xb6, 0x1d, 0x1c, 0xa8, 0x55, 0x3a, 0x97, 0x1a, 0xbb, 0xf3, 0x35, 0x12, 0x25, 0x18},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := claimPrimarySlot(tt.args.randomness, tt.args.slot, tt.args.epoch, tt.args.threshold, tt.args.keypair)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErr.Error())
			} else {
				assert.NoError(t, err)
			}
			if tt.exp != nil && res != nil {
				assert.Equal(t, tt.exp.output, res.output)
			}
		})
	}
}
