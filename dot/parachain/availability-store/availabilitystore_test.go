package availability_store

import (
	"reflect"
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/require"
)

func TestAvailabilityStore_LoadAvailableData(t *testing.T) {
	basePath := t.TempDir()
	type args struct {
		candidate common.Hash
	}
	tests := map[string]struct {
		args    args
		want    AvailableData
		wantErr bool
	}{
		"base": {
			args:    args{candidate: common.Hash{0x01}},
			want:    AvailableData{},
			wantErr: false,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			as, err := NewAvailabilityStore(Config{basepath: basePath})
			require.NoError(t, err)

			got, err := as.LoadAvailableData(tt.args.candidate)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadAvailableData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("LoadAvailableData() got = %v, want %v", got, tt.want)
			}
		})
	}
}
