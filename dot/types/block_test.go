// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package types

import (
	"bytes"
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"

	"github.com/stretchr/testify/require"
)

func TestEmptyBlock(t *testing.T) {
	block := NewEmptyBlock()
	isEmpty := block.Empty()
	require.True(t, isEmpty)

	block = NewBlock(*NewEmptyHeader(), Body{})
	isEmpty = block.Empty()
	require.True(t, isEmpty)

	parentHash, err := common.HexToHash("0x4545454545454545454545454545454545454545454545454545454545454545")
	require.NoError(t, err)

	stateRoot, err := common.HexToHash("0x2747ab7c0dc38b7f2afba82bd5e2d6acef8c31e09800f660b75ec84a7005099f")
	require.NoError(t, err)

	extrinsicsRoot, err := common.HexToHash("0x03170a2e7597b7b7e3d84c05391d139a62b157e78786d8c082f29dcf4c111314")
	require.NoError(t, err)

	header := NewHeader(parentHash, stateRoot, extrinsicsRoot, 1, NewDigest())

	block = NewBlock(*header, Body{})
	isEmpty = block.Empty()
	require.False(t, isEmpty)

	block = NewBlock(*NewEmptyHeader(), *NewBody([]Extrinsic{[]byte{4, 1}}))
	isEmpty = block.Empty()
	require.False(t, isEmpty)
}

func TestEncodeAndDecodeBlock(t *testing.T) {
	// SCALE encoding of the block, NewBlock(*header, *NewBody([]Extrinsic{[]byte{4, 1}}))
	expected, err := common.HexToBytes("0x4545454545454545454545454545454545454545454545454545454545454545042747ab7c0dc38b7f2afba82bd5e2d6acef8c31e09800f660b75ec84a7005099f03170a2e7597b7b7e3d84c05391d139a62b157e78786d8c082f29dcf4c1113140004080401") //nolint:lll
	require.NoError(t, err)

	parentHash, err := common.HexToHash("0x4545454545454545454545454545454545454545454545454545454545454545")
	require.NoError(t, err)

	stateRoot, err := common.HexToHash("0x2747ab7c0dc38b7f2afba82bd5e2d6acef8c31e09800f660b75ec84a7005099f")
	require.NoError(t, err)

	extrinsicsRoot, err := common.HexToHash("0x03170a2e7597b7b7e3d84c05391d139a62b157e78786d8c082f29dcf4c111314")
	require.NoError(t, err)

	header := NewHeader(parentHash, stateRoot, extrinsicsRoot, 1, nil)

	block := NewBlock(*header, *NewBody([]Extrinsic{[]byte{4, 1}}))

	enc, err := scale.Marshal(block)
	require.NoError(t, err)

	require.Equal(t, expected, enc)

	dec := NewBlock(*NewEmptyHeader(), Body{})
	err = scale.Unmarshal(enc, &dec)
	require.NoError(t, err)
	dec.Header.Hash()
	require.Equal(t, block, dec)
}

func TestDeepCopyBlock(t *testing.T) {
	data := []byte{
		69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69,
		69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 4, 39, 71, 171,
		124, 13, 195, 139, 127, 42, 251, 168, 43, 213, 226, 214, 172, 239, 140,
		49, 224, 152, 0, 246, 96, 183, 94, 200, 74, 112, 5, 9, 159, 3, 23, 10,
		46, 117, 151, 183, 183, 227, 216, 76, 5, 57, 29, 19, 154, 98, 177, 87,
		231, 135, 134, 216, 192, 130, 242, 157, 207, 76, 17, 19, 20, 0, 0}
	block := NewBlock(*NewEmptyHeader(), Body{})

	err := scale.Unmarshal(data, &block)
	if err != nil {
		t.Fatal(err)
	}

	bc, err := block.DeepCopy()
	require.NoError(t, err)
	bc.Header.ParentHash = common.Hash{}
	require.NotEqual(t, block.Header.ParentHash, bc.Header.ParentHash)
}

func TestMustEncodeBlock(t *testing.T) {
	h1 := NewHeader(common.Hash{}, common.Hash{}, common.Hash{}, 0, NewDigest())

	b1 := NewBlock(*h1, *NewBody([]Extrinsic{[]byte{4, 1}}))
	enc, err := b1.Encode()
	require.NoError(t, err)

	h2 := NewHeader(common.Hash{0x1, 0x2}, common.Hash{}, common.Hash{}, 0, NewDigest())

	b2 := NewBlock(*h2, *NewBody([]Extrinsic{[]byte{0xa, 0xb}}))
	enc2, err := b2.Encode()
	require.NoError(t, err)

	tests := []struct {
		name string
		take *Block
		want []byte
	}{
		{
			name: "correct",
			take: &b1,
			want: enc,
		},
		{
			name: "correct2",
			take: &b2,
			want: enc2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.take.MustEncode(); !bytes.Equal(got, tt.want) {
				t.Errorf("MustEncode() = %v, want %v", got, tt.want)
			}
		})
	}
}
