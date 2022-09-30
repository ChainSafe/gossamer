// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInherentDataMarshal(t *testing.T) {
	tests := []struct {
		name            string
		getInherentData func(t *testing.T) *InherentData
		want            []byte
	}{
		{
			/*
				let mut data = InherentData::new();
				let timestamp: u64 = 99;
				data.put_data(*b"babeslot", &timestamp).unwrap();
				data.put_data(*b"timstap0", &timestamp).unwrap();
			*/
			getInherentData: func(t *testing.T) *InherentData {
				id := NewInherentData()
				err := id.SetInherent(Babeslot, uint64(99))
				require.NoError(t, err)

				err = id.SetInherent(Timstap0, uint64(99))
				require.NoError(t, err)
				return id
			},
			want: []byte{8, 98, 97, 98, 101, 115, 108, 111, 116, 32, 99, 0, 0, 0, 0, 0, 0, 0, 116, 105, 109, 115, 116, 97, 112, 48, 32, 99, 0, 0, 0, 0, 0, 0, 0}, //nolint:lll
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idata := tt.getInherentData(t)
			got, err := idata.Encode()
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
