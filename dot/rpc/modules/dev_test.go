package modules

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"

	apimocks "github.com/ChainSafe/gossamer/dot/rpc/modules/mocks"
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
			if got := uint64ToHex(tt.args.input); got != tt.want {
				t.Errorf("uint64ToHex() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDevModule_EpochLength(t *testing.T) {
	mockBlockProducerAPI := new(apimocks.BlockProducerAPI)
	mockBlockProducerAPI.On("EpochLength").Return(uint64(23))
	devModule := NewDevModule(mockBlockProducerAPI, nil)

	type fields struct {
		networkAPI       NetworkAPI
		blockProducerAPI BlockProducerAPI
	}
	type args struct {
		r   *http.Request
		req *EmptyRequest
		res *string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		err     error
		exp     string
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
		var res string
		tt.args.res = &res
		t.Run(tt.name, func(t *testing.T) {
			m := &DevModule{
				networkAPI:       tt.fields.networkAPI,
				blockProducerAPI: tt.fields.blockProducerAPI,
			}
			var err error
			if err = m.EpochLength(tt.args.r, tt.args.req, tt.args.res); (err != nil) != tt.wantErr {
				t.Errorf("EpochLength() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.exp, *tt.args.res)
			}
		})
	}
}

func TestDevModule_SlotDuration(t *testing.T) {
	mockBlockProducerAPI := new(apimocks.BlockProducerAPI)
	mockBlockProducerAPI.On("SlotDuration").Return(uint64(23))

	type fields struct {
		networkAPI       NetworkAPI
		blockProducerAPI BlockProducerAPI
	}
	type args struct {
		r   *http.Request
		req *EmptyRequest
		res *string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		err     error
		exp     string
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
		var res string
		tt.args.res = &res
		t.Run(tt.name, func(t *testing.T) {
			m := &DevModule{
				networkAPI:       tt.fields.networkAPI,
				blockProducerAPI: tt.fields.blockProducerAPI,
			}
			var err error
			if err = m.SlotDuration(tt.args.r, tt.args.req, tt.args.res); (err != nil) != tt.wantErr {
				t.Errorf("SlotDuration() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.exp, *tt.args.res)
			}
		})
	}
}

func TestDevModule_Control(t *testing.T) {
	mockBlockProducerAPI := new(apimocks.BlockProducerAPI)
	mockErrorBlockProducerAPI := new(apimocks.BlockProducerAPI)
	mockNetworkAPI := new(apimocks.NetworkAPI)
	mockErrorNetworkAPI := new(apimocks.NetworkAPI)

	mockErrorBlockProducerAPI.On("Pause").Return(errors.New("babe pause error"))
	mockBlockProducerAPI.On("Pause").Return(nil)

	mockErrorBlockProducerAPI.On("Resume").Return(errors.New("babe resume error"))
	mockBlockProducerAPI.On("Resume").Return(nil)

	mockErrorNetworkAPI.On("Stop").Return(errors.New("network stop error"))
	mockNetworkAPI.On("Stop").Return(nil)

	mockErrorNetworkAPI.On("Start").Return(errors.New("network start error"))
	mockNetworkAPI.On("Start").Return(nil)

	type fields struct {
		networkAPI       NetworkAPI
		blockProducerAPI BlockProducerAPI
	}
	type args struct {
		r   *http.Request
		req *[]string
		res *string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		err     error
		exp     string
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
			wantErr: true,
			err: errors.New("not a block producer"),
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
			wantErr: true,
			err: errors.New("babe pause error"),
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
			wantErr: true,
			err: errors.New("babe resume error"),
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
			wantErr: true,
			err: errors.New("network stop error"),
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
			wantErr: true,
			err: errors.New("network start error"),
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
		var res string
		tt.args.res = &res
		t.Run(tt.name, func(t *testing.T) {
			m := &DevModule{
				networkAPI:       tt.fields.networkAPI,
				blockProducerAPI: tt.fields.blockProducerAPI,
			}
			var err error
			if err = m.Control(tt.args.r, tt.args.req, tt.args.res); (err != nil) != tt.wantErr {
				t.Errorf("Control() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.exp, *tt.args.res)
			}
		})
	}
}
