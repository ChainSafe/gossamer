// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package babe

import (
	"errors"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCalculateThreshold(t *testing.T) {
	//exp1, err := scale.NewUint128(big.NewInt(0))
	//assert.NoError(t, err)
	type args struct {
		C1       uint64
		C2       uint64
		numAuths int
	}
	tests := []struct {
		name    string
		args    args
		exp    *scale.Uint128
		expErr  error
	}{
		{
			name: "happy path",
			args: args{
				C1: 1,
				C2: 2,
				numAuths: 3,
			},
			exp: &scale.Uint128{Upper:0x34d00ad6148e1800, Lower:0x0},
		},
		{
			name: "0 value input",
			args: args{
				C1: 0,
				C2: 0,
				numAuths: 0,
			},
			expErr: errors.New("invalid input: C1 and C2 cannot be 0"),
		},
		{
			name: "C1 > C2",
			args: args{
				C1: 5,
				C2: 2,
				numAuths: 0,
			},
			expErr: errors.New("invalid C1/C2: greater than 1"),
		},
		{
			name: "max threshold",
			args: args{
				C1: 2147483647,
				C2: 2147483647,
				numAuths: 3,
			},
			exp: scale.MaxUint128,
		},
		// TODO Maybe come test this (might not be worth it/possible)
		//{
		//	name: "threshold to long",
		//	args: args{
		//		C1: (^uint64(0)) - 1090,
		//		C2: ^uint64(0),
		//		numAuths: 1,
		//	},
		//	expErr: errors.New("threshold must be under or equal to 16 bytes"),
		//},
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