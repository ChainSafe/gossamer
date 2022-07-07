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
		slot                   Slot
		runtimeInstanceBuilder func(ctrl *gomock.Controller) (instance runtime.Instance, forceReturn <-chan struct{})
	}
	tests := map[string]struct {
		fields fields
		args   args
		want   []*transaction.ValidTransaction
	}{
		"return no transaction whilst waiting for queue": {
			args: args{
				slot: Slot{
					// we don't set the start time or duration
					// so the timeout is effectively 0 such that
					// the timer triggers on the second iteration
					// of the for loop
				},
				runtimeInstanceBuilder: func(ctrl *gomock.Controller) (instance runtime.Instance, forceReturn <-chan struct{}) {
					return nil, nil
				},
			},
			fields: fields{
				transactionStateBuilder: func(ctrl *gomock.Controller) TransactionState {
					transactionState := NewMockTransactionState(ctrl)

					transactionState.EXPECT().Pop().Return(nil)
					transactionState.EXPECT().NextPushWatcher().Return(nil)

					return transactionState
				},
			},
		},
		"return one transaction": {
			args: args{
				slot: Slot{
					// we don't set the start time or duration
					// so the timeout is effectively 0 such that
					// the timer triggers on the second iteration
					// of the for loop
				},
				runtimeInstanceBuilder: func(ctrl *gomock.Controller) (instance runtime.Instance, forceReturn <-chan struct{}) {
					mockInstance := NewMockInstance(ctrl)
					mockInstance.EXPECT().ApplyExtrinsic(types.Extrinsic{1}).
						Return([]byte{0, 0}, nil)
					return mockInstance, nil
				},
			},
			fields: fields{
				transactionStateBuilder: func(ctrl *gomock.Controller) TransactionState {
					transactionState := NewMockTransactionState(ctrl)
					transaction := &transaction.ValidTransaction{
						Extrinsic: types.Extrinsic{1},
					}
					transactionState.EXPECT().Pop().Return(transaction)
					return transactionState
				},
			},
			want: []*transaction.ValidTransaction{
				{Extrinsic: types.Extrinsic{1}},
			},
		},
		"return one transaction after waiting for queue": {
			args: args{
				slot: Slot{
					// we don't set the start time or duration
					// so the timeout is effectively 0 such that
					// the timer triggers on the second iteration
					// of the for loop
				},
				runtimeInstanceBuilder: func(ctrl *gomock.Controller) (instance runtime.Instance, forceReturn <-chan struct{}) {
					mockInstance := NewMockInstance(ctrl)
					mockInstance.EXPECT().ApplyExtrinsic(types.Extrinsic{1}).
						Return([]byte{0, 0}, nil)
					return mockInstance, nil
				},
			},
			fields: fields{
				transactionStateBuilder: func(ctrl *gomock.Controller) TransactionState {
					transactionState := NewMockTransactionState(ctrl)

					firstPop := transactionState.EXPECT().Pop().Return(nil)

					pushWatcher := make(chan struct{})
					close(pushWatcher)
					transactionState.EXPECT().NextPushWatcher().Return(pushWatcher)

					transaction := &transaction.ValidTransaction{
						Extrinsic: types.Extrinsic{1},
					}
					transactionState.EXPECT().Pop().Return(transaction).After(firstPop)

					return transactionState
				},
			},
			want: []*transaction.ValidTransaction{
				{Extrinsic: types.Extrinsic{1}},
			},
		},
		"return two transactions": {
			args: args{
				slot: Slot{
					start:    time.Now(),
					duration: time.Hour,
				},
				runtimeInstanceBuilder: func(ctrl *gomock.Controller) (instance runtime.Instance, forceReturn <-chan struct{}) {
					forceReturnCh := make(chan struct{}, 1)
					mockInstance := NewMockInstance(ctrl)

					firstCall := mockInstance.EXPECT().ApplyExtrinsic(types.Extrinsic{1}).
						Return([]byte{0, 0}, nil)

					mockInstance.EXPECT().ApplyExtrinsic(types.Extrinsic{2}).
						DoAndReturn(func(data types.Extrinsic) ([]byte, error) {
							forceReturnCh <- struct{}{} // triggers the timer on the select case
							return []byte{0, 0}, nil
						}).After(firstCall)

					return mockInstance, forceReturnCh
				},
			},
			fields: fields{
				transactionStateBuilder: func(ctrl *gomock.Controller) TransactionState {
					transactionState := NewMockTransactionState(ctrl)

					transaction1 := &transaction.ValidTransaction{
						Extrinsic: types.Extrinsic{1},
					}
					call := transactionState.EXPECT().Pop().Return(transaction1)

					transaction2 := &transaction.ValidTransaction{
						Extrinsic: types.Extrinsic{2},
					}
					transactionState.EXPECT().Pop().Return(transaction2).After(call)

					return transactionState
				},
			},
			want: []*transaction.ValidTransaction{
				{Extrinsic: types.Extrinsic{1}},
				{Extrinsic: types.Extrinsic{2}},
			},
		},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			runtimeInstance, forceReturnCh := tt.args.runtimeInstanceBuilder(ctrl)

			b := &BlockBuilder{
				keypair:               tt.fields.keypair,
				transactionState:      tt.fields.transactionStateBuilder(ctrl),
				blockState:            tt.fields.blockState,
				currentAuthorityIndex: tt.fields.currentAuthorityIndex,
				preRuntimeDigest:      tt.fields.preRuntimeDigest,
				testForceReturn:       forceReturnCh,
			}
			got := b.buildBlockExtrinsics(tt.args.slot, runtimeInstance)
			assert.Equal(t, tt.want, got)
		})
	}
}
