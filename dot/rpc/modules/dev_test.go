package modules

import (
	"errors"
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

	var res string
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
	}{
		{
			name: "EpochLength OK",
			fields: fields{
				devModule.networkAPI,
				devModule.blockProducerAPI,
			},
			args: args{
				r:   nil,
				req: &EmptyRequest{},
				res: &res,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &DevModule{
				networkAPI:       tt.fields.networkAPI,
				blockProducerAPI: tt.fields.blockProducerAPI,
			}
			if err := m.EpochLength(tt.args.r, tt.args.req, tt.args.res); (err != nil) != tt.wantErr {
				t.Errorf("EpochLength() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDevModule_SlotDuration(t *testing.T) {
	mockBlockProducerAPI := new(apimocks.BlockProducerAPI)
	mockBlockProducerAPI.On("SlotDuration").Return(uint64(23))

	var res string
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
	}{
		{
			name: "SlotDuration OK",
			fields: fields{
				nil,
				mockBlockProducerAPI,
			},
			args: args{
				r:   nil,
				req: &EmptyRequest{},
				res: &res,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &DevModule{
				networkAPI:       tt.fields.networkAPI,
				blockProducerAPI: tt.fields.blockProducerAPI,
			}
			if err := m.SlotDuration(tt.args.r, tt.args.req, tt.args.res); (err != nil) != tt.wantErr {
				t.Errorf("SlotDuration() error = %v, wantErr %v", err, tt.wantErr)
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

	var res string
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
	}{
		{
			name: "Not a BlockProducer",
			fields: fields{
				nil,
				nil,
			},
			args: args{
				r:   nil,
				req: &[]string{"babe", "stop"},
				res: &res,
			},
			wantErr: true,
		},
		{
			name: "Babe Stop Error",
			fields: fields{
				mockNetworkAPI,
				mockErrorBlockProducerAPI,
			},
			args: args{
				r:   nil,
				req: &[]string{"babe", "stop"},
				res: &res,
			},
			wantErr: true,
		},
		{
			name: "Babe Stop OK",
			fields: fields{
				mockNetworkAPI,
				mockBlockProducerAPI,
			},
			args: args{
				r:   nil,
				req: &[]string{"babe", "stop"},
				res: &res,
			},
			wantErr: false,
		},
		{
			name: "Babe Start Error",
			fields: fields{
				mockNetworkAPI,
				mockErrorBlockProducerAPI,
			},
			args: args{
				r:   nil,
				req: &[]string{"babe", "start"},
				res: &res,
			},
			wantErr: true,
		},
		{
			name: "Babe Start OK",
			fields: fields{
				mockNetworkAPI,
				mockBlockProducerAPI,
			},
			args: args{
				r:   nil,
				req: &[]string{"babe", "start"},
				res: &res,
			},
			wantErr: false,
		},
		{
			name: "Network Stop Error",
			fields: fields{
				mockErrorNetworkAPI,
				mockBlockProducerAPI,
			},
			args: args{
				r:   nil,
				req: &[]string{"network", "stop"},
				res: &res,
			},
			wantErr: true,
		},
		{
			name: "Network Stop OK",
			fields: fields{
				mockNetworkAPI,
				mockBlockProducerAPI,
			},
			args: args{
				r:   nil,
				req: &[]string{"network", "stop"},
				res: &res,
			},
			wantErr: false,
		},
		{
			name: "Network Start Error",
			fields: fields{
				mockErrorNetworkAPI,
				mockBlockProducerAPI,
			},
			args: args{
				r:   nil,
				req: &[]string{"network", "start"},
				res: &res,
			},
			wantErr: true,
		},
		{
			name: "Network Start OK",
			fields: fields{
				mockNetworkAPI,
				mockBlockProducerAPI,
			},
			args: args{
				r:   nil,
				req: &[]string{"network", "start"},
				res: &res,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &DevModule{
				networkAPI:       tt.fields.networkAPI,
				blockProducerAPI: tt.fields.blockProducerAPI,
			}
			if err := m.Control(tt.args.r, tt.args.req, tt.args.res); (err != nil) != tt.wantErr {
				t.Errorf("Control() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
