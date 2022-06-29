// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only
package babe

import (
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/transaction"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestBlockBuilder_buildBlockExtrinsics(t *testing.T) {
	type fields struct {
		keypair                 *sr25519.Keypair
		transactionStateBuilder func(ctrl *gomock.Controller) TransactionState
		blockState              BlockState
		currentAuthorityIndex   uint32
		preRuntimeDigest        *types.PreRuntimeDigest
	}
	type args struct {
		slot Slot
		rt   runtime.Instance
	}
	tests := map[string]struct {
		fields fields
		args   args
		want   []*transaction.ValidTransaction
	}{
		"empty transaction queue": {
			args: args{
				slot: Slot{
					start:    time.Now(),
					duration: time.Minute,
				},
			},
			fields: fields{
				transactionStateBuilder: func(ctrl *gomock.Controller) TransactionState {
					mockTransacionState := NewMockTransactionState(ctrl)
					mockTransacionState.EXPECT().Pop().Return(nil).AnyTimes()
					return mockTransacionState
				},
			},
		},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			b := &BlockBuilder{
				keypair:               tt.fields.keypair,
				transactionState:      tt.fields.transactionStateBuilder(ctrl),
				blockState:            tt.fields.blockState,
				currentAuthorityIndex: tt.fields.currentAuthorityIndex,
				preRuntimeDigest:      tt.fields.preRuntimeDigest,
			}
			got := b.buildBlockExtrinsics(tt.args.slot, tt.args.rt)
			assert.Equal(t, tt.want, got)
		})
	}
}
