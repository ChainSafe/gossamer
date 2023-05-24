package erasure_coding

import (
	"reflect"
	"testing"
)

func TestObtainChunks(t *testing.T) {
	type args struct {
		validatorsQty int
		data          []byte
	}
	tests := map[string]struct {
		args    args
		want    [][]byte
		wantErr bool
	}{
		"test1": {
			args: args{
				validatorsQty: 10,
				data:          []byte("this is a test of the erasure coding"),
			},
			want: [][]byte{{116, 104, 105, 115}, {32, 105, 115, 32}, {97, 32, 116, 101}, {115, 116, 32, 111},
				{102, 32, 116, 104}, {101, 32, 101, 114}, {97, 115, 117, 114}, {101, 32, 99, 111}, {100, 105, 110, 103},
				{0, 0, 0, 0}, {133, 189, 154, 178}, {88, 245, 245, 220}, {59, 208, 165, 70}, {127, 213, 208, 179}},
			wantErr: false,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := ObtainChunks(tt.args.validatorsQty, tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ObtainChunks() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ObtainChunks() got = %v, want %v", got, tt.want)
			}
		})
	}
}
