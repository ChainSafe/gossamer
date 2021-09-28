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
	"math/big"
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
)

var (
	digestItem = scale.MustNewVaryingDataType(ChangesTrieRootDigest{}, PreRuntimeDigest{}, ConsensusDigest{}, SealDigest{})
	digest     = scale.NewVaryingDataTypeSlice(digestItem)
	testDigest = digest
)
var _ = testDigest.Add(
	PreRuntimeDigest{
		ConsensusEngineID: BabeEngineID,
		Data:              []byte{1, 2, 3},
	},
	SealDigest{
		ConsensusEngineID: BabeEngineID,
		Data:              []byte{4, 5, 6, 7},
	},
)

func TestNumber(t *testing.T) {
	testHash := common.NewHash([]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0xa, 0xb, 0xc, 0xd, 0xe, 0xf, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0xa, 0xb, 0xc, 0xd, 0xe, 0xf})

	headerVdt, err := NewHeader(testHash, testHash, testHash, big.NewInt(5), testDigest)
	require.NoError(t, err)

	bd := BlockData{
		Hash:          common.NewHash([]byte{0}),
		Header:        headerVdt,
		Body:          nil,
		Receipt:       nil,
		MessageQueue:  nil,
		Justification: nil,
	}

	num := bd.Number()
	require.Equal(t, big.NewInt(5), num)
}

func TestBlockDataEncodeAndDecodeEmpty(t *testing.T) {
	expected, err := common.HexToBytes("0x00000000000000000000000000000000000000000000000000000000000000000000000000")
	require.NoError(t, err)

	bd := BlockData{
		Hash:          common.NewHash([]byte{0}),
		Header:        nil,
		Body:          nil,
		Receipt:       nil,
		MessageQueue:  nil,
		Justification: nil,
	}

	enc, err := scale.Marshal(bd)
	require.NoError(t, err)

	require.Equal(t, expected, enc)

	var block BlockData
	if bd.Header != nil {
		block.Header = NewEmptyHeader()
	}
	err = scale.Unmarshal(enc, &block)
	require.NoError(t, err)
	if block.Header != nil {
		_ = block.Header.Hash()
	}
	require.Equal(t, bd, block)
}

func TestBlockDataEncodeAndDecodeHeader(t *testing.T) {
	expected, err := common.HexToBytes("0x000000000000000000000000000000000000000000000000000000000000000001000102030405060708090a0b0c0d0e0f000102030405060708090a0b0c0d0e0f04000102030405060708090a0b0c0d0e0f000102030405060708090a0b0c0d0e0f000102030405060708090a0b0c0d0e0f000102030405060708090a0b0c0d0e0f0806424142450c0102030542414245100405060700000000")
	require.NoError(t, err)

	testHash := common.NewHash([]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0xa, 0xb, 0xc, 0xd, 0xe, 0xf, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0xa, 0xb, 0xc, 0xd, 0xe, 0xf})

	headerVdt, err := NewHeader(testHash, testHash, testHash, big.NewInt(1), testDigest)
	require.NoError(t, err)

	bd := BlockData{
		Hash:          common.NewHash([]byte{0}),
		Header:        headerVdt,
		Body:          nil,
		Receipt:       nil,
		MessageQueue:  nil,
		Justification: nil,
	}

	enc, err := scale.Marshal(bd)
	require.NoError(t, err)

	require.Equal(t, expected, enc)

	var block BlockData
	if bd.Header != nil {
		block.Header = NewEmptyHeader()
	}
	err = scale.Unmarshal(enc, &block)
	require.NoError(t, err)
	if block.Header != nil {
		_ = block.Header.Hash()
	}
	require.Equal(t, bd, block)
}

func TestBlockDataEncodeAndDecodeBody(t *testing.T) {
	expected, err := common.HexToBytes("0x0000000000000000000000000000000000000000000000000000000000000000000104100a0b0c0d000000")
	require.NoError(t, err)

	bd := BlockData{
		Hash:          common.NewHash([]byte{0}),
		Header:        nil,
		Body:          NewBody([]Extrinsic{[]byte{0xa, 0xb, 0xc, 0xd}}),
		Receipt:       nil,
		MessageQueue:  nil,
		Justification: nil,
	}

	enc, err := scale.Marshal(bd)
	require.NoError(t, err)

	require.Equal(t, expected, enc)

	var block BlockData
	if bd.Header != nil {
		block.Header = NewEmptyHeader()
	}
	err = scale.Unmarshal(enc, &block)
	require.NoError(t, err)
	if block.Header != nil {
		_ = block.Header.Hash()
	}
	require.Equal(t, bd, block)
}

func TestBlockDataEncodeAndDecodeAll(t *testing.T) {
	expected, err := common.HexToBytes("0x7d0000000000000000000000000000000000000000000000000000000000000001000102030405060708090a0b0c0d0e0f000102030405060708090a0b0c0d0e0f04000102030405060708090a0b0c0d0e0f000102030405060708090a0b0c0d0e0f000102030405060708090a0b0c0d0e0f000102030405060708090a0b0c0d0e0f0806424142450c010203054241424510040506070104100a0b0c0d010401010402010403")
	require.NoError(t, err)

	hash := common.NewHash([]byte{125})
	testHash := common.NewHash([]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0xa, 0xb, 0xc, 0xd, 0xe, 0xf, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0xa, 0xb, 0xc, 0xd, 0xe, 0xf})

	headerVdt, err := NewHeader(testHash, testHash, testHash, big.NewInt(1), testDigest)
	require.NoError(t, err)

	bd := BlockData{
		Hash:          hash,
		Header:        headerVdt,
		Body:          NewBody([]Extrinsic{[]byte{0xa, 0xb, 0xc, 0xd}}),
		Receipt:       &[]byte{1},
		MessageQueue:  &[]byte{2},
		Justification: &[]byte{3},
	}

	enc, err := scale.Marshal(bd)
	require.NoError(t, err)

	require.Equal(t, expected, enc)

	var block BlockData
	if bd.Header != nil {
		block.Header = NewEmptyHeader()
	}
	err = scale.Unmarshal(enc, &block)
	require.NoError(t, err)
	if block.Header != nil {
		_ = block.Header.Hash()
	}
	require.Equal(t, bd, block)
}
