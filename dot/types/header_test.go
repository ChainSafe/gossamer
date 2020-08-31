// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package types

import (
	"bytes"
	"log"
	"math/big"
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"

	"github.com/stretchr/testify/require"
)

func TestDecodeHeader(t *testing.T) {
	header, err := NewHeader(common.Hash{}, big.NewInt(0), common.Hash{}, common.Hash{}, [][]byte{{}})
	require.NoError(t, err)

	enc, err := header.Encode()
	require.NoError(t, err)

	rw := &bytes.Buffer{}
	rw.Write(enc)
	dec, err := new(Header).Decode(rw)
	require.NoError(t, err)
	dec.Hash()
	require.Equal(t, header, dec)
}

func TestMustEncodeHeader(t *testing.T) {
	//correct
	bh1, err := NewHeader(common.Hash{}, big.NewInt(0), common.Hash{}, common.Hash{}, [][]byte{{}})
	require.NoError(t, err)
	enc, err := bh1.Encode()
	require.NoError(t, err)

	//correct2
	bh2, err := NewHeader(common.Hash{}, big.NewInt(0), common.Hash{}, common.Hash{}, [][]byte{{0, 0}, {1, 2}, {2, 4}, {3, 6}, {4, 8}})
	require.NoError(t, err)
	enc2, err := bh2.Encode()
	require.NoError(t, err)

	//panic
	bh3 := &Header{
		ParentHash: common.Hash{}, 
		Number: nil, 
		StateRoot: common.Hash{}, 
		ExtrinsicsRoot: common.Hash{}, 
		Digest: [][]byte{{0, 0}, {1, 2}, {2, 4}, {3, 6}, {4, 8}},
	}

	tests := []struct {
		name string
		take *Header
		want []byte
	}{
		{
			name: "correct",
			take: bh1,
			want: enc,
		},
		{
			name: "correct2",
			take: bh2,
			want: enc2,
		},
		{
			name: "panic",
			take: bh3,
		},
	}

	defer func() {
		if err := recover(); err != nil {
			log.Println("it's panic!!!:", err)
		}
	}()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.take.MustEncode(); !bytes.Equal(got, tt.want) {
				t.Errorf("MustEncode() = %v, want %v", got, tt.want)
			}
		})
	}
}
