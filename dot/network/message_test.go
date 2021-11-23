// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/variadic"

	"github.com/stretchr/testify/require"
)

func TestEncodeBlockRequestMessage(t *testing.T) {
	expected, err := common.HexToBytes("0x08808080082220fd19d9ebac759c993fd2e05a1cff9e757d8741c2704c8682c15b5503496b6aa1280130011220dcd1346701ca8396496e52aa2785b1748deb6db09551b72159dcb3e08991025b")
	require.Nil(t, err)

	genesisHash, err := common.HexToBytes("0xdcd1346701ca8396496e52aa2785b1748deb6db09551b72159dcb3e08991025b")
	require.Nil(t, err)

	endBlock, err := common.HexToHash("0xfd19d9ebac759c993fd2e05a1cff9e757d8741c2704c8682c15b5503496b6aa1")
	require.NoError(t, err)

	one := uint32(1)

	bm := &BlockRequestMessage{
		RequestedData: 1,
		StartingBlock: *variadic.NewUint64OrHashFromBytes(append([]byte{0}, genesisHash...)),
		EndBlockHash:  &endBlock,
		Direction:     1,
		Max:           &one,
	}

	encMsg, err := bm.Encode()
	require.NoError(t, err)

	require.Equal(t, expected, encMsg) // Pass!

	res := new(BlockRequestMessage)
	err = res.Decode(encMsg)
	require.NoError(t, err)
	require.Equal(t, bm, res)
}

func TestEncodeBlockRequestMessage_BlockHash(t *testing.T) {
	genesisHash, err := common.HexToBytes("0xdcd1346701ca8396496e52aa2785b1748deb6db09551b72159dcb3e08991025b")
	require.Nil(t, err)

	endBlock, err := common.HexToHash("0xfd19d9ebac759c993fd2e05a1cff9e757d8741c2704c8682c15b5503496b6aa1")
	require.NoError(t, err)

	one := uint32(1)
	bm := &BlockRequestMessage{
		RequestedData: 1,
		StartingBlock: *variadic.NewUint64OrHashFromBytes(append([]byte{0}, genesisHash...)),
		EndBlockHash:  &endBlock,
		Direction:     1,
		Max:           &one,
	}

	encMsg, err := bm.Encode()
	require.NoError(t, err)

	res := new(BlockRequestMessage)
	err = res.Decode(encMsg)
	require.NoError(t, err)
	require.Equal(t, bm, res)
}

func TestEncodeBlockRequestMessage_BlockNumber(t *testing.T) {
	endBlock, err := common.HexToHash("0xfd19d9ebac759c993fd2e05a1cff9e757d8741c2704c8682c15b5503496b6aa1")
	require.NoError(t, err)

	one := uint32(1)
	bm := &BlockRequestMessage{
		RequestedData: 1,
		StartingBlock: *variadic.NewUint64OrHashFromBytes([]byte{1, 1}),
		EndBlockHash:  &endBlock,
		Direction:     1,
		Max:           &one,
	}

	encMsg, err := bm.Encode()
	require.NoError(t, err)

	res := new(BlockRequestMessage)
	err = res.Decode(encMsg)
	require.NoError(t, err)
	require.Equal(t, bm, res)
}

func TestBlockRequestString(t *testing.T) {
	genesisHash, err := common.HexToBytes("0xdcd1346701ca8396496e52aa2785b1748deb6db09551b72159dcb3e08991025b")
	require.Nil(t, err)

	bm := &BlockRequestMessage{
		RequestedData: 1,
		StartingBlock: *variadic.NewUint64OrHashFromBytes(append([]byte{0}, genesisHash...)),
		EndBlockHash:  nil,
		Direction:     1,
		Max:           nil,
	}

	_ = bm.String()
}

func TestEncodeBlockRequestMessage_NoOptionals(t *testing.T) {
	genesisHash, err := common.HexToBytes("0xdcd1346701ca8396496e52aa2785b1748deb6db09551b72159dcb3e08991025b")
	require.Nil(t, err)

	bm := &BlockRequestMessage{
		RequestedData: 1,
		StartingBlock: *variadic.NewUint64OrHashFromBytes(append([]byte{0}, genesisHash...)),
		EndBlockHash:  nil,
		Direction:     1,
		Max:           nil,
	}

	encMsg, err := bm.Encode()
	require.NoError(t, err)

	res := new(BlockRequestMessage)
	err = res.Decode(encMsg)
	require.NoError(t, err)
	require.Equal(t, bm, res)
}

func TestEncodeBlockResponseMessage_Empty(t *testing.T) {
	bd := types.NewEmptyBlockData()
	bd.Header = types.NewEmptyHeader()
	bd.Header.Hash()

	bm := &BlockResponseMessage{
		BlockData: []*types.BlockData{bd},
	}

	enc, err := bm.Encode()
	require.NoError(t, err)

	empty := types.NewEmptyBlockData()
	empty.Header = types.NewEmptyHeader()

	act := &BlockResponseMessage{
		BlockData: []*types.BlockData{empty},
	}
	err = act.Decode(enc)
	require.NoError(t, err)

	for _, b := range act.BlockData {
		if b.Header != nil {
			_ = b.Header.Hash()
		}
	}

	require.Equal(t, bm, act)
}

func TestEncodeBlockResponseMessage_WithBody(t *testing.T) {
	hash := common.NewHash([]byte{0})
	testHash := common.NewHash([]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0xa, 0xb, 0xc, 0xd, 0xe, 0xf, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0xa, 0xb, 0xc, 0xd, 0xe, 0xf})
	header, err := types.NewHeader(testHash, testHash, testHash, big.NewInt(1), types.NewDigest())
	require.NoError(t, err)

	exts := [][]byte{{1, 3, 5, 7}, {9, 1, 2}, {3, 4, 5}}
	body := types.NewBody(types.BytesArrayToExtrinsics(exts))

	bd := &types.BlockData{
		Hash:          hash,
		Header:        header,
		Body:          body,
		Receipt:       nil,
		MessageQueue:  nil,
		Justification: nil,
	}

	bm := &BlockResponseMessage{
		BlockData: []*types.BlockData{bd},
	}

	enc, err := bm.Encode()
	require.NoError(t, err)

	empty := types.NewEmptyBlockData()
	empty.Header = types.NewEmptyHeader()

	act := &BlockResponseMessage{
		BlockData: []*types.BlockData{empty},
	}
	err = act.Decode(enc)
	require.NoError(t, err)

	for _, bd := range act.BlockData {
		if bd.Header != nil {
			_ = bd.Header.Hash()
		}
	}

	require.Equal(t, bm, act)

}

func TestEncodeBlockResponseMessage_WithAll(t *testing.T) {
	exp := common.MustHexToBytes("0x0aa2010a2000000000000000000000000000000000000000000000000000000000000000001262000102030405060708090a0b0c0d0e0f000102030405060708090a0b0c0d0e0f04000102030405060708090a0b0c0d0e0f000102030405060708090a0b0c0d0e0f000102030405060708090a0b0c0d0e0f000102030405060708090a0b0c0d0e0f001a0510010305071a040c0901021a040c0304052201012a0102320103")
	hash := common.NewHash([]byte{0})
	testHash := common.NewHash([]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0xa, 0xb, 0xc, 0xd, 0xe, 0xf, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0xa, 0xb, 0xc, 0xd, 0xe, 0xf})

	header, err := types.NewHeader(testHash, testHash, testHash, big.NewInt(1), types.NewDigest())
	require.NoError(t, err)

	exts := [][]byte{{1, 3, 5, 7}, {9, 1, 2}, {3, 4, 5}}
	body := types.NewBody(types.BytesArrayToExtrinsics(exts))

	bd := &types.BlockData{
		Hash:          hash,
		Header:        header,
		Body:          body,
		Receipt:       &[]byte{1},
		MessageQueue:  &[]byte{2},
		Justification: &[]byte{3},
	}

	bm := &BlockResponseMessage{
		BlockData: []*types.BlockData{bd},
	}

	enc, err := bm.Encode()
	require.NoError(t, err)
	require.Equal(t, exp, enc)

	empty := types.NewEmptyBlockData()
	empty.Header = types.NewEmptyHeader()

	act := &BlockResponseMessage{
		BlockData: []*types.BlockData{empty},
	}
	err = act.Decode(enc)
	require.NoError(t, err)

	for _, bd := range act.BlockData {
		if bd.Header != nil {
			_ = bd.Header.Hash()
		}
	}

	require.Equal(t, bm, act)
}

func TestEncodeBlockAnnounceMessage(t *testing.T) {
	// this value is a concatenation of:
	//  ParentHash: Hash: 0x4545454545454545454545454545454545454545454545454545454545454545
	//	Number: *big.Int // block number: 1
	//	StateRoot:  Hash: 0xb3266de137d20a5d0ff3a6401eb57127525fd9b2693701f0bf5a8a853fa3ebe0
	//	ExtrinsicsRoot: Hash: 0x03170a2e7597b7b7e3d84c05391d139a62b157e78786d8c082f29dcf4c111314
	//	Digest: []byte

	//                                    mtparenthash                                                      bnstateroot                                                       extrinsicsroot                                                di
	expected, err := common.HexToBytes("0x454545454545454545454545454545454545454545454545454545454545454504b3266de137d20a5d0ff3a6401eb57127525fd9b2693701f0bf5a8a853fa3ebe003170a2e7597b7b7e3d84c05391d139a62b157e78786d8c082f29dcf4c1113140000")
	require.Nil(t, err)

	parentHash, err := common.HexToHash("0x4545454545454545454545454545454545454545454545454545454545454545")
	require.Nil(t, err)

	stateRoot, err := common.HexToHash("0xb3266de137d20a5d0ff3a6401eb57127525fd9b2693701f0bf5a8a853fa3ebe0")
	require.Nil(t, err)

	extrinsicsRoot, err := common.HexToHash("0x03170a2e7597b7b7e3d84c05391d139a62b157e78786d8c082f29dcf4c111314")
	require.Nil(t, err)

	bhm := &BlockAnnounceMessage{
		ParentHash:     parentHash,
		Number:         big.NewInt(1),
		StateRoot:      stateRoot,
		ExtrinsicsRoot: extrinsicsRoot,
		Digest:         types.NewDigest(),
	}
	encMsg, err := bhm.Encode()
	require.Nil(t, err)

	require.Equal(t, expected, encMsg)
}

func TestDecode_BlockAnnounceMessage(t *testing.T) {
	announceMessage, err := common.HexToBytes("0x454545454545454545454545454545454545454545454545454545454545454504b3266de137d20a5d0ff3a6401eb57127525fd9b2693701f0bf5a8a853fa3ebe003170a2e7597b7b7e3d84c05391d139a62b157e78786d8c082f29dcf4c1113140000")
	require.Nil(t, err)

	bhm := BlockAnnounceMessage{
		Number: big.NewInt(0),
		Digest: types.NewDigest(),
	}
	err = bhm.Decode(announceMessage)
	require.Nil(t, err)

	parentHash, err := common.HexToHash("0x4545454545454545454545454545454545454545454545454545454545454545")
	require.Nil(t, err)

	stateRoot, err := common.HexToHash("0xb3266de137d20a5d0ff3a6401eb57127525fd9b2693701f0bf5a8a853fa3ebe0")
	require.Nil(t, err)

	extrinsicsRoot, err := common.HexToHash("0x03170a2e7597b7b7e3d84c05391d139a62b157e78786d8c082f29dcf4c111314")
	require.Nil(t, err)

	expected := BlockAnnounceMessage{
		ParentHash:     parentHash,
		Number:         big.NewInt(1),
		StateRoot:      stateRoot,
		ExtrinsicsRoot: extrinsicsRoot,
		Digest:         types.NewDigest(),
	}

	require.Equal(t, expected, bhm)
}

func TestEncodeTransactionMessageSingleExtrinsic(t *testing.T) {
	// expected:
	// 0x04 - Scale encoded count of Extrinsic array(count = 1)
	// 0x10 - Scale encoded length of the first Extrinsic(len = 4)
	// 0x01020304 - value of array extrinsic array
	expected, err := common.HexToBytes("0x041001020304")
	require.Nil(t, err)

	extrinsic := types.Extrinsic{0x01, 0x02, 0x03, 0x04}

	transactionMessage := TransactionMessage{Extrinsics: []types.Extrinsic{extrinsic}}

	encMsg, err := transactionMessage.Encode()
	require.Nil(t, err)

	require.Equal(t, expected, encMsg)
}

func TestEncodeTransactionMessageTwoExtrinsics(t *testing.T) {
	// expected:
	// 0x08 - Scale encoded count of Extrinsic array(count = 2)
	// 0x0c - Scale encoded length of the first Extrinsic(len = 3)
	// 0x010203 - Data of first Extrinsic
	// 0x10 - Scale encoded length of the second Extrinsic(len = 4)
	// 0x04050607 - Data of second Extrinsic
	expected, err := common.HexToBytes("0x080c0102031004050607")
	require.Nil(t, err)

	extrinsic1 := types.Extrinsic{0x01, 0x02, 0x03}
	extrinsic2 := types.Extrinsic{0x04, 0x05, 0x06, 0x07}

	transactionMessage := TransactionMessage{Extrinsics: []types.Extrinsic{extrinsic1, extrinsic2}}

	encMsg, err := transactionMessage.Encode()
	require.Nil(t, err)

	require.Equal(t, expected, encMsg)
}

func TestDecodeTransactionMessageOneExtrinsic(t *testing.T) {
	originalMessage, err := common.HexToBytes("0x041001020304") // (without message type byte prepended)
	require.Nil(t, err)

	decodedMessage := new(TransactionMessage)
	err = decodedMessage.Decode(originalMessage)
	require.Nil(t, err)

	extrinsic := types.Extrinsic{0x01, 0x02, 0x03, 0x04}
	expected := TransactionMessage{[]types.Extrinsic{extrinsic}}

	require.Equal(t, expected, *decodedMessage)

}

func TestDecodeTransactionMessageTwoExtrinsics(t *testing.T) {
	originalMessage, err := common.HexToBytes("0x080c0102031004050607") // (without message type byte prepended)
	require.Nil(t, err)

	decodedMessage := new(TransactionMessage)
	err = decodedMessage.Decode(originalMessage)
	require.Nil(t, err)

	extrinsic1 := types.Extrinsic{0x01, 0x02, 0x03}
	extrinsic2 := types.Extrinsic{0x04, 0x05, 0x06, 0x07}
	expected := TransactionMessage{[]types.Extrinsic{extrinsic1, extrinsic2}}

	require.Equal(t, expected, *decodedMessage)
}

func TestDecodeConsensusMessage(t *testing.T) {
	testData := "03100405"
	msg := "0x" + testData

	encMsg, err := common.HexToBytes(msg)
	require.Nil(t, err)

	m := new(ConsensusMessage)
	err = m.Decode(encMsg)
	require.Nil(t, err)

	out, err := hex.DecodeString(testData)
	require.Nil(t, err)

	expected := &ConsensusMessage{
		Data: out,
	}

	require.Equal(t, expected, m)

	encodedMessage, err := expected.Encode()
	require.Nil(t, err)
	require.Equal(t, encMsg, encodedMessage)
}
