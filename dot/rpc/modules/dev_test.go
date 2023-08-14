// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package modules

import (
	"errors"
	"net/http"
	"testing"

	"github.com/ChainSafe/gossamer/dot/rpc/modules/mocks"
	"github.com/golang/mock/gomock"

	"github.com/stretchr/testify/assert"
)

func Test_uint64ToHex(t *testing.T) {
	type args struct {
		input uint64
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "uint64ToHex_one",
			args: args{
				input: uint64(1),
			},
			want: "0x0100000000000000",
		},
		{
			name: "uint64ToHex_zero",
			args: args{
				input: uint64(0),
			},
			want: "0x0000000000000000",
		},
		{
			name: "uint64ToHex_max",
			args: args{
				input: uint64(18446744073709551615),
			},
			want: "0xffffffffffffffff",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := uint64ToHex(tt.args.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDevModule_EpochLength(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockBlockProducerAPI := mocks.NewMockBlockProducerAPI(ctrl)
	mockBlockProducerAPI.EXPECT().EpochLength().Return(uint64(23))
	devModule := NewDevModule(mockBlockProducerAPI, nil)

	type fields struct {
		networkAPI       NetworkAPI
		blockProducerAPI BlockProducerAPI
	}
	type args struct {
		r   *http.Request
		req *EmptyRequest
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		expErr error
		exp    string
	}{
		{
			name: "EpochLength_OK",
			fields: fields{
				devModule.networkAPI,
				devModule.blockProducerAPI,
			},
			args: args{
				req: &EmptyRequest{},
			},
			exp: "0x1700000000000000",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &DevModule{
				networkAPI:       tt.fields.networkAPI,
				blockProducerAPI: tt.fields.blockProducerAPI,
			}
			res := ""
			err := m.EpochLength(tt.args.r, tt.args.req, &res)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.exp, res)
		})
	}
}

func TestDevModule_SlotDuration(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockBlockProducerAPI := mocks.NewMockBlockProducerAPI(ctrl)
	mockBlockProducerAPI.EXPECT().SlotDuration().Return(uint64(23))

	type fields struct {
		networkAPI       NetworkAPI
		blockProducerAPI BlockProducerAPI
	}
	type args struct {
		r   *http.Request
		req *EmptyRequest
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		expErr error
		exp    string
	}{
		{
			name: "SlotDuration_OK",
			fields: fields{
				nil,
				mockBlockProducerAPI,
			},
			args: args{
				req: &EmptyRequest{},
			},
			exp: "0x1700000000000000",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &DevModule{
				networkAPI:       tt.fields.networkAPI,
				blockProducerAPI: tt.fields.blockProducerAPI,
			}
			res := ""
			err := m.SlotDuration(tt.args.r, tt.args.req, &res)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.exp, res)
		})
	}
}

func TestDevModule_Control(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockBlockProducerAPI := mocks.NewMockBlockProducerAPI(ctrl)
	mockNetworkAPI := mocks.NewMockNetworkAPI(ctrl)
	mockErrorNetworkAPI := mocks.NewMockNetworkAPI(ctrl)

	mockBlockProducerAPI.EXPECT().Pause()
	mockBlockProducerAPI.EXPECT().Resume()

	mockErrorNetworkAPI.EXPECT().Stop().Return(errors.New("network stop error"))
	mockNetworkAPI.EXPECT().Stop().Return(nil)

	mockErrorNetworkAPI.EXPECT().Start().Return(errors.New("network start error"))
	mockNetworkAPI.EXPECT().Start().Return(nil)

	type fields struct {
		networkAPI       NetworkAPI
		blockProducerAPI BlockProducerAPI
	}
	type args struct {
		r   *http.Request
		req *[]string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		expErr error
		exp    string
	}{
		{
			name: "Not_a_BlockProducer",
			fields: fields{
				nil,
				nil,
			},
			args: args{
				req: &[]string{"babe", "stop"},
			},
			expErr: errors.New("not a block producer"),
		},
		{
			name: "Babe_Stop_OK",
			fields: fields{
				mockNetworkAPI,
				mockBlockProducerAPI,
			},
			args: args{
				req: &[]string{"babe", "stop"},
			},
			exp: "babe service stopped",
		},
		{
			name: "Babe_Start_OK",
			fields: fields{
				mockNetworkAPI,
				mockBlockProducerAPI,
			},
			args: args{
				req: &[]string{"babe", "start"},
			},
			exp: "babe service started",
		},
		{
			name: "Network_Stop_Error",
			fields: fields{
				mockErrorNetworkAPI,
				mockBlockProducerAPI,
			},
			args: args{
				req: &[]string{"network", "stop"},
			},
			exp:    "network service stopped",
			expErr: errors.New("network stop error"),
		},
		{
			name: "Network_Stop_OK",
			fields: fields{
				mockNetworkAPI,
				mockBlockProducerAPI,
			},
			args: args{
				req: &[]string{"network", "stop"},
			},
			exp: "network service stopped",
		},
		{
			name: "Network_Start_Error",
			fields: fields{
				mockErrorNetworkAPI,
				mockBlockProducerAPI,
			},
			args: args{
				req: &[]string{"network", "start"},
			},
			exp:    "network service started",
			expErr: errors.New("network start error"),
		},
		{
			name: "Network_Start_OK",
			fields: fields{
				mockNetworkAPI,
				mockBlockProducerAPI,
			},
			args: args{
				req: &[]string{"network", "start"},
			},
			exp: "network service started",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &DevModule{
				networkAPI:       tt.fields.networkAPI,
				blockProducerAPI: tt.fields.blockProducerAPI,
			}
			var res string
			err := m.Control(tt.args.r, tt.args.req, &res)
			if tt.expErr != nil {
				assert.EqualError(t, err, tt.expErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.exp, res)
		})
	}
}
