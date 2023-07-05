// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"encoding/hex"
	"regexp"
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/variadic"

	"github.com/stretchr/testify/require"
)

func TestEncodeBlockRequestMessage(t *testing.T) {
	t.Parallel()

	expected := common.MustHexToBytes("0x0880808008280130011220dcd1346701ca8396496e52" +
		"aa2785b1748deb6db09551b72159dcb3e08991025b")
	genesisHash := common.MustHexToBytes("0xdcd1346701ca8396496e52aa2785b1748deb6db09551b72159dcb3e08991025b")

	var one uint32 = 1
	bm := &BlockRequestMessage{
		RequestedData: 1,
		StartingBlock: *variadic.NewUint32OrHashFromBytes(append([]byte{0}, genesisHash...)),
		Direction:     1,
		Max:           &one,
	}

	encMsg, err := bm.Encode()
	require.NoError(t, err)

	require.Equal(t, expected, encMsg)

	res := new(BlockRequestMessage)
	err = res.Decode(encMsg)
	require.NoError(t, err)
	require.Equal(t, bm, res)
}

func TestEncodeBlockRequestMessage_BlockHash(t *testing.T) {
	t.Parallel()

	genesisHash := common.MustHexToBytes("0xdcd1346701ca8396496e52aa2785b1748deb6db09551b72159dcb3e08991025b")

	var one uint32 = 1
	bm := &BlockRequestMessage{
		RequestedData: 1,
		StartingBlock: *variadic.NewUint32OrHashFromBytes(append([]byte{0}, genesisHash...)),
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
	t.Parallel()

	var one uint32 = 1
	bm := &BlockRequestMessage{
		RequestedData: 1,
		StartingBlock: *variadic.NewUint32OrHashFromBytes([]byte{1, 1}),
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
	t.Parallel()

	genesisHash := common.MustHexToBytes("0xdcd1346701ca8396496e52aa2785b1748deb6db09551b72159dcb3e08991025b")

	bm := &BlockRequestMessage{
		RequestedData: 1,
		StartingBlock: *variadic.NewUint32OrHashFromBytes(append([]byte{0}, genesisHash...)),
		Direction:     1,
		Max:           nil,
	}

	var blockRequestStringRegex = regexp.MustCompile(
		`^\ABlockRequestMessage RequestedData=[0-9]* StartingBlock={[\[0-9(\s?)]+\]} Direction=[0-9]* Max=[0-9]*\z$`) //nolint:lll

	match := blockRequestStringRegex.MatchString(bm.String())
	require.True(t, match)
}

func TestEncodeBlockRequestMessage_NoOptionals(t *testing.T) {
	t.Parallel()

	genesisHash := common.MustHexToBytes("0xdcd1346701ca8396496e52aa2785b1748deb6db09551b72159dcb3e08991025b")

	bm := &BlockRequestMessage{
		RequestedData: 1,
		StartingBlock: *variadic.NewUint32OrHashFromBytes(append([]byte{0}, genesisHash...)),
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
	t.Parallel()

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
	t.Parallel()

	hash := common.NewHash([]byte{0})
	testHash := common.NewHash([]byte{
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9,
		0xa, 0xb, 0xc, 0xd, 0xe, 0xf,
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9,
		0xa, 0xb, 0xc, 0xd, 0xe, 0xf})

	header := types.NewHeader(testHash, testHash, testHash, 1, types.NewDigest())

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
	t.Parallel()

	exp := common.MustHexToBytes("0x0aa2010a2000000000000000000000000000000000000000000000000000000000000000001262000102030405060708090a0b0c0d0e0f000102030405060708090a0b0c0d0e0f04000102030405060708090a0b0c0d0e0f000102030405060708090a0b0c0d0e0f000102030405060708090a0b0c0d0e0f000102030405060708090a0b0c0d0e0f001a0510010305071a040c0901021a040c0304052201012a0102320103") //nolint:lll
	hash := common.NewHash([]byte{0})
	testHash := common.NewHash([]byte{
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9,
		0xa, 0xb, 0xc, 0xd, 0xe, 0xf,
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9,
		0xa, 0xb, 0xc, 0xd, 0xe, 0xf})

	header := types.NewHeader(testHash, testHash, testHash, 1, types.NewDigest())

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
	/* this value is a concatenation of:
	 *  ParentHash: Hash: 0x4545454545454545454545454545454545454545454545454545454545454545
	 *	Number: *big.Int // block number: 1
	 *	StateRoot:  Hash: 0xb3266de137d20a5d0ff3a6401eb57127525fd9b2693701f0bf5a8a853fa3ebe0
	 *	ExtrinsicsRoot: Hash: 0x03170a2e7597b7b7e3d84c05391d139a62b157e78786d8c082f29dcf4c111314
	 *	Digest: []byte
	 *
	 * mtparenthash bnstateroot extrinsicsroot di
	 */

	t.Parallel()

	expected := common.MustHexToBytes("0x454545454545454545454545454545454545454545454545454545454545454504b3266de137d20a5d0ff3a6401eb57127525fd9b2693701f0bf5a8a853fa3ebe003170a2e7597b7b7e3d84c05391d139a62b157e78786d8c082f29dcf4c1113140000") //nolint:lll

	parentHash := common.MustHexToHash("0x4545454545454545454545454545454545454545454545454545454545454545")

	stateRoot := common.MustHexToHash("0xb3266de137d20a5d0ff3a6401eb57127525fd9b2693701f0bf5a8a853fa3ebe0")

	extrinsicsRoot := common.MustHexToHash("0x03170a2e7597b7b7e3d84c05391d139a62b157e78786d8c082f29dcf4c111314")

	bhm := &BlockAnnounceMessage{
		ParentHash:     parentHash,
		Number:         1,
		StateRoot:      stateRoot,
		ExtrinsicsRoot: extrinsicsRoot,
		Digest:         types.NewDigest(),
	}
	encMsg, err := bhm.Encode()
	require.NoError(t, err)

	require.Equal(t, expected, encMsg)
}

func TestDecode_BlockAnnounceMessage(t *testing.T) {
	t.Parallel()

	announceMessage := common.MustHexToBytes("0x454545454545454545454545454545454545454545454545454545454545454504b3266de137d20a5d0ff3a6401eb57127525fd9b2693701f0bf5a8a853fa3ebe003170a2e7597b7b7e3d84c05391d139a62b157e78786d8c082f29dcf4c1113140000") //nolint:lll

	bhm := BlockAnnounceMessage{
		Number: 0,
		Digest: types.NewDigest(),
	}

	err := bhm.Decode(announceMessage)
	require.NoError(t, err)

	parentHash := common.MustHexToHash("0x4545454545454545454545454545454545454545454545454545454545454545")

	stateRoot := common.MustHexToHash("0xb3266de137d20a5d0ff3a6401eb57127525fd9b2693701f0bf5a8a853fa3ebe0")

	extrinsicsRoot := common.MustHexToHash("0x03170a2e7597b7b7e3d84c05391d139a62b157e78786d8c082f29dcf4c111314")

	expected := BlockAnnounceMessage{
		ParentHash:     parentHash,
		Number:         1,
		StateRoot:      stateRoot,
		ExtrinsicsRoot: extrinsicsRoot,
		Digest:         types.NewDigest(),
	}

	require.Equal(t, expected, bhm)
}

func TestEncodeTransactionMessageSingleExtrinsic(t *testing.T) {
	/* expected:
	 * 0x04 - Scale encoded count of Extrinsic array(count = 1)
	 * 0x10 - Scale encoded length of the first Extrinsic(len = 4)
	 * 0x01020304 - value of array extrinsic array
	 */
	t.Parallel()
	expected := common.MustHexToBytes("0x041001020304")
	extrinsic := types.Extrinsic{0x01, 0x02, 0x03, 0x04}

	transactionMessage := TransactionMessage{Extrinsics: []types.Extrinsic{extrinsic}}

	encMsg, err := transactionMessage.Encode()
	require.NoError(t, err)

	require.Equal(t, expected, encMsg)
}

func TestEncodeTransactionMessageTwoExtrinsics(t *testing.T) {
	/* expected:
	 * 0x08 - Scale encoded count of Extrinsic array(count = 2)
	 * 0x0c - Scale encoded length of the first Extrinsic(len = 3)
	 * 0x010203 - Data of first Extrinsic
	 * 0x10 - Scale encoded length of the second Extrinsic(len = 4)
	 * 0x04050607 - Data of second Extrinsic
	 */

	t.Parallel()

	expected := common.MustHexToBytes("0x080c0102031004050607")

	extrinsic1 := types.Extrinsic{0x01, 0x02, 0x03}
	extrinsic2 := types.Extrinsic{0x04, 0x05, 0x06, 0x07}

	transactionMessage := TransactionMessage{Extrinsics: []types.Extrinsic{extrinsic1, extrinsic2}}

	encMsg, err := transactionMessage.Encode()
	require.NoError(t, err)

	require.Equal(t, expected, encMsg)
}

func TestDecodeTransactionMessageOneExtrinsic(t *testing.T) {
	t.Parallel()

	// (without message type byte prepended)
	originalMessage := common.MustHexToBytes("0x041001020304")

	decodedMessage := new(TransactionMessage)
	err := decodedMessage.Decode(originalMessage)
	require.NoError(t, err)

	extrinsic := types.Extrinsic{0x01, 0x02, 0x03, 0x04}
	expected := TransactionMessage{[]types.Extrinsic{extrinsic}}

	require.Equal(t, expected, *decodedMessage)

}

func TestDecodeTransactionMessageTwoExtrinsics(t *testing.T) {
	t.Parallel()

	// (without message type byte prepended)
	originalMessage, err := common.HexToBytes("0x080c0102031004050607")
	require.NoError(t, err)

	decodedMessage := new(TransactionMessage)
	err = decodedMessage.Decode(originalMessage)
	require.NoError(t, err)

	extrinsic1 := types.Extrinsic{0x01, 0x02, 0x03}
	extrinsic2 := types.Extrinsic{0x04, 0x05, 0x06, 0x07}
	expected := TransactionMessage{[]types.Extrinsic{extrinsic1, extrinsic2}}

	require.Equal(t, expected, *decodedMessage)
}

func TestDecodeConsensusMessage(t *testing.T) {
	t.Parallel()

	const testData = "0x03100405"

	encMsg := common.MustHexToBytes(testData)

	m := new(ConsensusMessage)

	err := m.Decode(encMsg)
	require.NoError(t, err)

	out, err := hex.DecodeString(testData[2:])
	require.NoError(t, err)

	expected := &ConsensusMessage{
		Data: out,
	}

	require.Equal(t, expected, m)

	encodedMessage, err := expected.Encode()
	require.NoError(t, err)
	require.Equal(t, encMsg, encodedMessage)
}

func TestAscendingBlockRequest(t *testing.T) {
	one := uint32(1)
	three := uint32(3)
	maxResponseSize := uint32(MaxBlocksInResponse)
	cases := map[string]struct {
		startNumber, targetNumber   uint
		expectedBlockRequestMessage []*BlockRequestMessage
	}{
		"start_greater_than_target": {
			startNumber:                 10,
			targetNumber:                0,
			expectedBlockRequestMessage: []*BlockRequestMessage{},
		},

		"no_difference_between_start_and_target": {
			startNumber:  10,
			targetNumber: 10,
			expectedBlockRequestMessage: []*BlockRequestMessage{
				{
					RequestedData: BootstrapRequestData,
					StartingBlock: *variadic.MustNewUint32OrHash(uint32(10)),
					Direction:     Ascending,
					Max:           &one,
				},
			},
		},

		"requesting_128_blocks": {
			startNumber:  0,
			targetNumber: 128,
			expectedBlockRequestMessage: []*BlockRequestMessage{
				{
					RequestedData: BootstrapRequestData,
					StartingBlock: *variadic.MustNewUint32OrHash(uint32(0)),
					Direction:     Ascending,
					Max:           &maxResponseSize,
				},
			},
		},

		"requesting_4_chunks_of_128_blocks": {
			startNumber:  0,
			targetNumber: 512, // 128 * 4
			expectedBlockRequestMessage: []*BlockRequestMessage{
				{
					RequestedData: BootstrapRequestData,
					StartingBlock: *variadic.MustNewUint32OrHash(uint32(0)),
					Direction:     Ascending,
					Max:           &maxResponseSize,
				},
				{
					RequestedData: BootstrapRequestData,
					StartingBlock: *variadic.MustNewUint32OrHash(uint32(128)),
					Direction:     Ascending,
					Max:           &maxResponseSize,
				},
				{
					RequestedData: BootstrapRequestData,
					StartingBlock: *variadic.MustNewUint32OrHash(uint32(256)),
					Direction:     Ascending,
					Max:           &maxResponseSize,
				},
				{
					RequestedData: BootstrapRequestData,
					StartingBlock: *variadic.MustNewUint32OrHash(uint32(384)),
					Direction:     Ascending,
					Max:           &maxResponseSize,
				},
			},
		},

		"requesting_4_chunks_of_128_plus_3_blocks": {
			startNumber:  0,
			targetNumber: (128 * 4) + 3,
			expectedBlockRequestMessage: []*BlockRequestMessage{
				{
					RequestedData: BootstrapRequestData,
					StartingBlock: *variadic.MustNewUint32OrHash(uint32(0)),
					Direction:     Ascending,
					Max:           &maxResponseSize,
				},
				{
					RequestedData: BootstrapRequestData,
					StartingBlock: *variadic.MustNewUint32OrHash(uint32(128)),
					Direction:     Ascending,
					Max:           &maxResponseSize,
				},
				{
					RequestedData: BootstrapRequestData,
					StartingBlock: *variadic.MustNewUint32OrHash(uint32(256)),
					Direction:     Ascending,
					Max:           &maxResponseSize,
				},
				{
					RequestedData: BootstrapRequestData,
					StartingBlock: *variadic.MustNewUint32OrHash(uint32(384)),
					Direction:     Ascending,
					Max:           &maxResponseSize,
				},
				{
					RequestedData: BootstrapRequestData,
					StartingBlock: *variadic.MustNewUint32OrHash(uint32(512)),
					Direction:     Ascending,
					Max:           &three,
				},
			},
		},
	}

	for tname, tt := range cases {
		tt := tt

		t.Run(tname, func(t *testing.T) {
			requests := NewAscedingBlockRequests(tt.startNumber, tt.targetNumber, BootstrapRequestData)
			require.Equal(t, tt.expectedBlockRequestMessage, requests)
		})
	}
}
