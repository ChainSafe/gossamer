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
		transactionStateBuilder func(ctrl *gomock.Controller, tickerCancel chan<- struct{}) TransactionState
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
		"initial empty transaction queue": {
			args: args{
				slot: Slot{
					start:    time.Now(),
					duration: 30 * time.Second,
				},
			},
			fields: fields{
				transactionStateBuilder: func(ctrl *gomock.Controller, tickerCancel chan<- struct{}) TransactionState {
					mockTransactionState := NewMockTransactionState(ctrl)

					call := mockTransactionState.EXPECT().Pop().DoAndReturn(func() (*transaction.ValidTransaction,
						error) {
						tickerCancel <- struct{}{}
						t.Log("GOT TO HERE")
						return nil, nil // nolint: nilnil
					})

					watcherOne := make(chan struct{})
					close(watcherOne)
					call2 := mockTransactionState.EXPECT().NextPushWatcher().
						Return(watcherOne).After(call)
					call3 := mockTransactionState.EXPECT().Pop().Return(nil).After(call2)
					mockTransactionState.EXPECT().NextPushWatcher().
						Return(watcherOne).After(call3)
					return mockTransactionState
				},
			},
		},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			tickerCancel := make(chan struct{}, 1)
			b := &BlockBuilder{
				keypair:               tt.fields.keypair,
				transactionState:      tt.fields.transactionStateBuilder(ctrl, tickerCancel),
				blockState:            tt.fields.blockState,
				currentAuthorityIndex: tt.fields.currentAuthorityIndex,
				preRuntimeDigest:      tt.fields.preRuntimeDigest,
				testSkipTimer:         tickerCancel,
			}
			got := b.buildBlockExtrinsics(tt.args.slot, tt.args.rt)
			assert.Equal(t, tt.want, got)
		})
	}
}
