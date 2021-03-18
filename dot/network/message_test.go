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

package network

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/optional"
	"github.com/ChainSafe/gossamer/lib/common/variadic"

	"github.com/stretchr/testify/require"
)

func TestEncodeBlockRequestMessage_BlockHash(t *testing.T) {
	genesisHash, err := common.HexToBytes("0xdcd1346701ca8396496e52aa2785b1748deb6db09551b72159dcb3e08991025b")
	require.Nil(t, err)

	endBlock, err := common.HexToHash("0xfd19d9ebac759c993fd2e05a1cff9e757d8741c2704c8682c15b5503496b6aa1")
	require.NoError(t, err)

	bm := &BlockRequestMessage{
		RequestedData: 1,
		StartingBlock: variadic.NewUint64OrHashFromBytes(append([]byte{0}, genesisHash...)),
		EndBlockHash:  optional.NewHash(true, endBlock),
		Direction:     1,
		Max:           optional.NewUint32(true, 1),
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

	bm := &BlockRequestMessage{
		RequestedData: 1,
		StartingBlock: variadic.NewUint64OrHashFromBytes([]byte{1, 1}),
		EndBlockHash:  optional.NewHash(true, endBlock),
		Direction:     1,
		Max:           optional.NewUint32(true, 1),
	}

	encMsg, err := bm.Encode()
	require.NoError(t, err)

	res := new(BlockRequestMessage)
	err = res.Decode(encMsg)
	require.NoError(t, err)
	require.Equal(t, bm, res)
}

func TestEncodeBlockRequestMessage_NoOptionals(t *testing.T) {
	genesisHash, err := common.HexToBytes("0xdcd1346701ca8396496e52aa2785b1748deb6db09551b72159dcb3e08991025b")
	require.Nil(t, err)

	bm := &BlockRequestMessage{
		RequestedData: 1,
		StartingBlock: variadic.NewUint64OrHashFromBytes(append([]byte{0}, genesisHash...)),
		EndBlockHash:  optional.NewHash(false, common.Hash{}),
		Direction:     1,
		Max:           optional.NewUint32(false, 0),
	}

	encMsg, err := bm.Encode()
	require.NoError(t, err)

	res := new(BlockRequestMessage)
	err = res.Decode(encMsg)
	require.NoError(t, err)
	require.Equal(t, bm, res)
}

func TestEncodeBlockResponseMessage_WithHeader(t *testing.T) {
	hash := common.NewHash([]byte{0})
	testHash := common.NewHash([]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0xa, 0xb, 0xc, 0xd, 0xe, 0xf, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0xa, 0xb, 0xc, 0xd, 0xe, 0xf})

	header := &optional.CoreHeader{
		ParentHash:     testHash,
		Number:         big.NewInt(1),
		StateRoot:      testHash,
		ExtrinsicsRoot: testHash,
		Digest:         &types.Digest{},
	}

	bd := &types.BlockData{
		Hash:          hash,
		Header:        optional.NewHeader(true, header),
		Body:          optional.NewBody(false, nil),
		Receipt:       optional.NewBytes(false, nil),
		MessageQueue:  optional.NewBytes(false, nil),
		Justification: optional.NewBytes(false, nil),
	}

	bm := &BlockResponseMessage{
		BlockData: []*types.BlockData{bd},
	}

	encMsg, err := bm.Encode()
	require.NoError(t, err)

	res := new(BlockResponseMessage)
	err = res.Decode(encMsg)
	require.NoError(t, err)
	require.Equal(t, bm, res)
}

func TestEncodeBlockResponseMessage_WithBody(t *testing.T) {
	hash := common.NewHash([]byte{0})
	testHash := common.NewHash([]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0xa, 0xb, 0xc, 0xd, 0xe, 0xf, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0xa, 0xb, 0xc, 0xd, 0xe, 0xf})

	header := &optional.CoreHeader{
		ParentHash:     testHash,
		Number:         big.NewInt(1),
		StateRoot:      testHash,
		ExtrinsicsRoot: testHash,
		Digest:         &types.Digest{},
	}

	exts := [][]byte{{1, 3, 5, 7}, {9, 1, 2}, {3, 4, 5}}
	body, err := types.NewBodyFromBytes(exts)
	require.NoError(t, err)

	bd := &types.BlockData{
		Hash:          hash,
		Header:        optional.NewHeader(true, header),
		Body:          body.AsOptional(),
		Receipt:       optional.NewBytes(false, nil),
		MessageQueue:  optional.NewBytes(false, nil),
		Justification: optional.NewBytes(false, nil),
	}

	bm := &BlockResponseMessage{
		BlockData: []*types.BlockData{bd},
	}

	encMsg, err := bm.Encode()
	require.NoError(t, err)

	res := new(BlockResponseMessage)
	err = res.Decode(encMsg)
	require.NoError(t, err)
	require.Equal(t, bm, res)
}

func TestEncodeBlockResponseMessage_WithAll(t *testing.T) {
	hash := common.NewHash([]byte{0})
	testHash := common.NewHash([]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0xa, 0xb, 0xc, 0xd, 0xe, 0xf, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0xa, 0xb, 0xc, 0xd, 0xe, 0xf})

	header := &optional.CoreHeader{
		ParentHash:     testHash,
		Number:         big.NewInt(1),
		StateRoot:      testHash,
		ExtrinsicsRoot: testHash,
		Digest:         &types.Digest{},
	}

	exts := [][]byte{{16, 1, 3, 5, 7}, {12, 9, 1, 2}, {12, 3, 4, 5}}
	body, err := types.NewBodyFromEncodedBytes(exts)
	require.NoError(t, err)

	bd := &types.BlockData{
		Hash:          hash,
		Header:        optional.NewHeader(true, header),
		Body:          body.AsOptional(),
		Receipt:       optional.NewBytes(true, []byte{77}),
		MessageQueue:  optional.NewBytes(true, []byte{88, 99}),
		Justification: optional.NewBytes(true, []byte{11, 22, 33}),
	}

	bm := &BlockResponseMessage{
		BlockData: []*types.BlockData{bd},
	}

	encMsg, err := bm.Encode()
	require.NoError(t, err)

	res := new(BlockResponseMessage)
	err = res.Decode(encMsg)
	require.NoError(t, err)
	require.Equal(t, bm, res)
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
		Digest:         types.Digest{},
	}
	encMsg, err := bhm.Encode()
	require.Nil(t, err)

	require.Equal(t, expected, encMsg)

}

func TestDecode_BlockAnnounceMessage(t *testing.T) {
	announceMessage, err := common.HexToBytes("0x454545454545454545454545454545454545454545454545454545454545454504b3266de137d20a5d0ff3a6401eb57127525fd9b2693701f0bf5a8a853fa3ebe003170a2e7597b7b7e3d84c05391d139a62b157e78786d8c082f29dcf4c1113140000")
	require.Nil(t, err)

	bhm := new(BlockAnnounceMessage)
	err = bhm.Decode(announceMessage)
	require.Nil(t, err)

	parentHash, err := common.HexToHash("0x4545454545454545454545454545454545454545454545454545454545454545")
	require.Nil(t, err)

	stateRoot, err := common.HexToHash("0xb3266de137d20a5d0ff3a6401eb57127525fd9b2693701f0bf5a8a853fa3ebe0")
	require.Nil(t, err)

	extrinsicsRoot, err := common.HexToHash("0x03170a2e7597b7b7e3d84c05391d139a62b157e78786d8c082f29dcf4c111314")
	require.Nil(t, err)

	expected := &BlockAnnounceMessage{
		ParentHash:     parentHash,
		Number:         big.NewInt(1),
		StateRoot:      stateRoot,
		ExtrinsicsRoot: extrinsicsRoot,
		Digest:         types.Digest{},
	}

	require.Equal(t, expected, bhm)
}

func TestEncodeTransactionMessageSingleExtrinsic(t *testing.T) {
	// expected:
	// 0x04 - Message Type
	// 0x14 - byte array (of all extrinsics encoded) - len 5
	// 0x10 - btye array (first extrinsic) - len 4
	// 0x01020304 - value of array extrinsic array
	expected, err := common.HexToBytes("0x141001020304")
	require.Nil(t, err)

	extrinsic := types.Extrinsic{0x01, 0x02, 0x03, 0x04}

	transactionMessage := TransactionMessage{Extrinsics: []types.Extrinsic{extrinsic}}

	encMsg, err := transactionMessage.Encode()
	require.Nil(t, err)

	require.Equal(t, expected, encMsg)
}

func TestEncodeTransactionMessageTwoExtrinsics(t *testing.T) {
	// expected:
	// 0x04 - Message Type
	// 0x24 - byte array (of all extrinsics encoded) - len 9
	// 0x0C - btye array (first extrinsic) len 3
	// 0x010203 - value of array first extrinsic array
	// 0x10 - byte array (second extrinsic) len 4
	// 0x04050607 - value of second extrinsic array
	expected, err := common.HexToBytes("0x240C0102031004050607")
	require.Nil(t, err)

	extrinsic1 := types.Extrinsic{0x01, 0x02, 0x03}
	extrinsic2 := types.Extrinsic{0x04, 0x05, 0x06, 0x07}

	transactionMessage := TransactionMessage{Extrinsics: []types.Extrinsic{extrinsic1, extrinsic2}}

	encMsg, err := transactionMessage.Encode()
	require.Nil(t, err)

	require.Equal(t, expected, encMsg)
}

func TestDecodeTransactionMessageOneExtrinsic(t *testing.T) {
	originalMessage, err := common.HexToBytes("0x141001020304") // (without message type byte prepended)
	require.Nil(t, err)

	decodedMessage := new(TransactionMessage)
	err = decodedMessage.Decode(originalMessage)
	require.Nil(t, err)

	extrinsic := types.Extrinsic{0x01, 0x02, 0x03, 0x04}
	expected := TransactionMessage{[]types.Extrinsic{extrinsic}}

	require.Equal(t, expected, *decodedMessage)

}

func TestDecodeTransactionMessageTwoExtrinsics(t *testing.T) {
	originalMessage, err := common.HexToBytes("0x240C0102031004050607") // (without message type byte prepended)
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
	ConsensusEngineID := types.BabeEngineID

	testID := hex.EncodeToString(types.BabeEngineID.ToBytes())
	testData := "03100405"

	msg := "0x" + testID + testData // 0x4241424503100405

	encMsg, err := common.HexToBytes(msg)
	require.Nil(t, err)

	m := new(ConsensusMessage)
	err = m.Decode(encMsg)
	require.Nil(t, err)

	out, err := hex.DecodeString(testData)
	require.Nil(t, err)

	expected := &ConsensusMessage{
		ConsensusEngineID: ConsensusEngineID,
		Data:              out,
	}

	require.Equal(t, expected, m)

	encodedMessage, err := expected.Encode()
	require.Nil(t, err)
	require.Equal(t, encMsg, encodedMessage)
}
