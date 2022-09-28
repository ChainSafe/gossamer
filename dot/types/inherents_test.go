package types

import (
	"testing"

	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
)

func TestInherentsDataMarshal(t *testing.T) {
	tests := []struct {
		name             string
		getInherentsData func(t *testing.T) *InherentsData
		want             []byte
	}{
		{
			/*
				let mut data = InherentData::new();
				let timestamp: u64 = 99;
				data.put_data(*b"babeslot", &timestamp).unwrap();
				data.put_data(*b"timstap0", &timestamp).unwrap();
			*/
			getInherentsData: func(t *testing.T) *InherentsData {
				id := NewInherentsData()
				err := id.SetInherent(Babeslot, uint64(99))
				require.NoError(t, err)

				err = id.SetInherent(Timstap0, uint64(99))
				require.NoError(t, err)
				return id
			},
			want: []byte{8, 98, 97, 98, 101, 115, 108, 111, 116, 32, 99, 0, 0, 0, 0, 0, 0, 0, 116, 105, 109, 115, 116, 97, 112, 48, 32, 99, 0, 0, 0, 0, 0, 0, 0},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idata := tt.getInherentsData(t)
			got, err := scale.Marshal(*idata)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
