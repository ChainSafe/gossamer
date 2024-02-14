// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package types

import (
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
)

var (
	testDigest = []DigestItem{
		newDigestItem(PreRuntimeDigest{
			ConsensusEngineID: BabeEngineID,
			Data:              []byte{1, 2, 3},
		}),
		newDigestItem(SealDigest{
			ConsensusEngineID: BabeEngineID,
			Data:              []byte{4, 5, 6, 7},
		}),
	}
)

func TestNumber(t *testing.T) {
	testHash := common.NewHash([]byte{
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9,
		0xa, 0xb, 0xc, 0xd, 0xe, 0xf,
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9,
		0xa, 0xb, 0xc, 0xd, 0xe, 0xf})

	headerVdt := NewHeader(testHash, testHash, testHash, 5, testDigest)

	bd := BlockData{
		Hash:          common.NewHash([]byte{0}),
		Header:        headerVdt,
		Body:          nil,
		Receipt:       nil,
		MessageQueue:  nil,
		Justification: nil,
	}

	num := bd.Number()
	require.Equal(t, uint(5), num)
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
	expected, err := common.HexToBytes("0x000000000000000000000000000000000000000000000000000000000000000001000102030405060708090a0b0c0d0e0f000102030405060708090a0b0c0d0e0f04000102030405060708090a0b0c0d0e0f000102030405060708090a0b0c0d0e0f000102030405060708090a0b0c0d0e0f000102030405060708090a0b0c0d0e0f0806424142450c0102030542414245100405060700000000") //nolint:lll
	require.NoError(t, err)

	testHash := common.NewHash([]byte{
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9,
		0xa, 0xb, 0xc, 0xd, 0xe, 0xf,
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9,
		0xa, 0xb, 0xc, 0xd, 0xe, 0xf})

	headerVdt := NewHeader(testHash, testHash, testHash, 1, testDigest)

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
	expected, err := common.HexToBytes("0x0000000000000000000000000000000000000000000000000000000000000000000104100a0b0c0d000000") //nolint:lll
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
	expected, err := common.HexToBytes("0x7d0000000000000000000000000000000000000000000000000000000000000001000102030405060708090a0b0c0d0e0f000102030405060708090a0b0c0d0e0f04000102030405060708090a0b0c0d0e0f000102030405060708090a0b0c0d0e0f000102030405060708090a0b0c0d0e0f000102030405060708090a0b0c0d0e0f0806424142450c010203054241424510040506070104100a0b0c0d010401010402010403") //nolint:lll
	require.NoError(t, err)

	hash := common.NewHash([]byte{125})
	testHash := common.NewHash([]byte{
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9,
		0xa, 0xb, 0xc, 0xd, 0xe, 0xf,
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9,
		0xa, 0xb, 0xc, 0xd, 0xe, 0xf})

	headerVdt := NewHeader(testHash, testHash, testHash, 1, testDigest)

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
