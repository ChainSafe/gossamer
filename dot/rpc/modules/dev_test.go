// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package modules

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/ChainSafe/gossamer/dot/rpc/modules/mocks"

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
			name: "uint64ToHex one",
			args: args{
				input: uint64(1),
			},
			want: "0x0100000000000000",
		},
		{
			name: "uint64ToHex zero",
			args: args{
				input: uint64(0),
			},
			want: "0x0000000000000000",
		},
		{
			name: "uint64ToHex max",
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
	mockBlockProducerAPI := mocks.NewBlockProducerAPI(t)
	mockBlockProducerAPI.On("EpochLength").Return(uint64(23))
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
			name: "EpochLength OK",
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
	mockBlockProducerAPI := mocks.NewBlockProducerAPI(t)
	mockBlockProducerAPI.On("SlotDuration").Return(uint64(23))

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
			name: "SlotDuration OK",
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
	mockBlockProducerAPI := mocks.NewBlockProducerAPI(t)
	mockErrorBlockProducerAPI := mocks.NewBlockProducerAPI(t)
	mockNetworkAPI := mocks.NewNetworkAPI(t)
	mockErrorNetworkAPI := mocks.NewNetworkAPI(t)

	mockErrorBlockProducerAPI.On("Pause").Return(fmt.Errorf("babe pause error"))
	mockBlockProducerAPI.On("Pause").Return(nil)

	mockErrorBlockProducerAPI.On("Resume").Return(fmt.Errorf("babe resume error"))
	mockBlockProducerAPI.On("Resume").Return(nil)

	mockErrorNetworkAPI.On("Stop").Return(fmt.Errorf("network stop error"))
	mockNetworkAPI.On("Stop").Return(nil)

	mockErrorNetworkAPI.On("Start").Return(fmt.Errorf("network start error"))
	mockNetworkAPI.On("Start").Return(nil)

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
			name: "Not a BlockProducer",
			fields: fields{
				nil,
				nil,
			},
			args: args{
				req: &[]string{"babe", "stop"},
			},
			expErr: fmt.Errorf("not a block producer"),
		},
		{
			name: "Babe Stop Error",
			fields: fields{
				mockNetworkAPI,
				mockErrorBlockProducerAPI,
			},
			args: args{
				req: &[]string{"babe", "stop"},
			},
			exp:    "babe service stopped",
			expErr: fmt.Errorf("babe pause error"),
		},
		{
			name: "Babe Stop OK",
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
			name: "Babe Start Error",
			fields: fields{
				mockNetworkAPI,
				mockErrorBlockProducerAPI,
			},
			args: args{
				req: &[]string{"babe", "start"},
			},
			exp:    "babe service started",
			expErr: fmt.Errorf("babe resume error"),
		},
		{
			name: "Babe Start OK",
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
			name: "Network Stop Error",
			fields: fields{
				mockErrorNetworkAPI,
				mockBlockProducerAPI,
			},
			args: args{
				req: &[]string{"network", "stop"},
			},
			exp:    "network service stopped",
			expErr: fmt.Errorf("network stop error"),
		},
		{
			name: "Network Stop OK",
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
			name: "Network Start Error",
			fields: fields{
				mockErrorNetworkAPI,
				mockBlockProducerAPI,
			},
			args: args{
				req: &[]string{"network", "start"},
			},
			exp:    "network service started",
			expErr: fmt.Errorf("network start error"),
		},
		{
			name: "Network Start OK",
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
