// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package babe

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_getSecondarySlotAuthor(t *testing.T) {
	type args struct {
		slot       uint64
		numAuths   int
		randomness Randomness
	}
	tests := []struct {
		name    string
		args    args
		exp     uint32
		wantErr bool
		expErr  error
	}{
		{
			name: "happy path",
			args: args{
				slot:       21,
				numAuths:   21,
				randomness: Randomness{},
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
