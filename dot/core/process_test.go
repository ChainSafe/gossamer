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

package core

import (
	"math/big"
	"reflect"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/core/types"
	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/optional"
	"github.com/ChainSafe/gossamer/lib/common/variadic"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/transaction"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/tests"
	"github.com/stretchr/testify/require"
)

// BlockRequestMsgType 1

func TestProcessBlockRequest(t *testing.T) {
	msgSend := make(chan network.Message, 10)

	cfg := &Config{
		MsgSend: msgSend,
	}

	s := newTestService(t, cfg)

	addTestBlocksToState(t, 1, s.blockState)

	endHash := s.blockState.BestBlockHash()

	request := &network.BlockRequestMessage{
		ID:            1,
		RequestedData: 3,
		StartingBlock: variadic.NewUint64OrHash([]byte{1, 1, 0, 0, 0, 0, 0, 0, 0}),
		EndBlockHash:  optional.NewHash(true, endHash),
		Direction:     1,
		Max:           optional.NewUint32(false, 0),
	}

	err := s.ProcessBlockRequestMessage(request)
	if err != nil {
		t.Fatal(err)
	}

	select {
	case resp := <-msgSend:
		msgType := resp.GetType()
		if !reflect.DeepEqual(msgType, network.BlockResponseMsgType) {
			t.Error(
				"received unexpected message type",
				"\nexpected:", network.BlockResponseMsgType,
				"\nreceived:", msgType,
			)
		}
	case <-time.After(TestMessageTimeout):
		t.Error("timeout waiting for message")
	}
}

// BlockResponseMsgType 2

func TestProcessBlockResponseMessage(t *testing.T) {
	tt := trie.NewEmptyTrie(nil)
	rt := runtime.NewTestRuntimeWithTrie(t, tests.POLKADOT_RUNTIME, tt)

	kp, err := sr25519.GenerateKeypair()
	require.Nil(t, err)

	pubkey := kp.Public().Encode()
	err = tt.Put(tests.AuthorityDataKey, append([]byte{4}, pubkey...))
	require.Nil(t, err)

	ks := keystore.NewKeystore()
	ks.Insert(kp)

	cfg := &Config{
		Runtime:     rt,
		Keystore:    ks,
		IsAuthority: false,
	}

	s := newTestService(t, cfg)

	hash := common.NewHash([]byte{0})
	body := optional.CoreBody{0xa, 0xb, 0xc, 0xd}

	parentHash := TestHeader.Hash()
	stateRoot, err := common.HexToHash("0x2747ab7c0dc38b7f2afba82bd5e2d6acef8c31e09800f660b75ec84a7005099f")
	require.Nil(t, err)

	extrinsicsRoot, err := common.HexToHash("0x03170a2e7597b7b7e3d84c05391d139a62b157e78786d8c082f29dcf4c111314")
	require.Nil(t, err)

	header := &types.Header{
		ParentHash:     parentHash,
		Number:         big.NewInt(1),
		StateRoot:      stateRoot,
		ExtrinsicsRoot: extrinsicsRoot,
		Digest:         [][]byte{},
	}

	bds := []*types.BlockData{{
		Hash:          header.Hash(),
		Header:        header.AsOptional(),
		Body:          types.NewBody([]byte{}).AsOptional(),
		Receipt:       optional.NewBytes(false, nil),
		MessageQueue:  optional.NewBytes(false, nil),
		Justification: optional.NewBytes(false, nil),
	}, {
		Hash:          hash,
		Header:        optional.NewHeader(false, nil),
		Body:          optional.NewBody(true, body),
		Receipt:       optional.NewBytes(true, []byte("asdf")),
		MessageQueue:  optional.NewBytes(true, []byte("ghjkl")),
		Justification: optional.NewBytes(true, []byte("qwerty")),
	}}

	blockResponse := &network.BlockResponseMessage{
		BlockData: bds,
	}

	err = s.ProcessBlockResponseMessage(blockResponse)
	require.Nil(t, err)

	res, err := s.blockState.GetHeader(header.Hash())
	require.Nil(t, err)

	if !reflect.DeepEqual(res, header) {
		t.Fatalf("Fail: got %v expected %v", res, header)
	}
}

// BlockAnnounceMsgType 3

func TestProcessBlockAnnounce(t *testing.T) {
	msgSend := make(chan network.Message)
	newBlocks := make(chan types.Block)

	cfg := &Config{
		MsgSend:     msgSend,
		Keystore:    keystore.NewKeystore(),
		NewBlocks:   newBlocks,
		IsAuthority: false,
	}

	s := newTestService(t, cfg)
	err := s.Start()
	require.Nil(t, err)

	expected := &network.BlockAnnounceMessage{
		Number:         big.NewInt(1),
		ParentHash:     TestHeader.Hash(),
		StateRoot:      common.Hash{},
		ExtrinsicsRoot: common.Hash{},
		Digest:         nil,
	}

	// simulate block sent from BABE session
	newBlocks <- types.Block{
		Header: &types.Header{
			Number:     big.NewInt(1),
			ParentHash: TestHeader.Hash(),
		},
		Body: types.NewBody([]byte{}),
	}

	select {
	case msg := <-msgSend:
		msgType := msg.GetType()
		require.Equal(t, network.BlockAnnounceMsgType, msgType)
		require.Equal(t, expected, msg)
	case <-time.After(TestMessageTimeout):
		t.Error("timeout waiting for message")
	}
}

// TransactionMsgType 4

func TestProcessTransactionMessage(t *testing.T) {
	tt := trie.NewEmptyTrie(nil)
	rt := runtime.NewTestRuntimeWithTrie(t, tests.POLKADOT_RUNTIME, tt)

	kp, err := sr25519.GenerateKeypair()
	require.Nil(t, err)

	pubkey := kp.Public().Encode()
	err = tt.Put(tests.AuthorityDataKey, append([]byte{4}, pubkey...))
	require.Nil(t, err)

	ks := keystore.NewKeystore()
	ks.Insert(kp)

	cfg := &Config{
		Runtime:          rt,
		Keystore:         ks,
		TransactionQueue: transaction.NewPriorityQueue(),
		IsAuthority:      true,
	}

	s := newTestService(t, cfg)

	// https://github.com/paritytech/substrate/blob/5420de3face1349a97eb954ae71c5b0b940c31de/core/transaction-pool/src/tests.rs#L95
	ext := []byte{1, 212, 53, 147, 199, 21, 253, 211, 28, 97, 20, 26, 189, 4, 169, 159, 214, 130, 44, 133, 88, 133, 76, 205, 227, 154, 86, 132, 231, 165, 109, 162, 125, 142, 175, 4, 21, 22, 135, 115, 99, 38, 201, 254, 161, 126, 37, 252, 82, 135, 97, 54, 147, 201, 18, 144, 156, 178, 38, 170, 71, 148, 242, 106, 72, 69, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 216, 5, 113, 87, 87, 40, 221, 120, 247, 252, 137, 201, 74, 231, 222, 101, 85, 108, 102, 39, 31, 190, 210, 14, 215, 124, 19, 160, 180, 203, 54, 110, 167, 163, 149, 45, 12, 108, 80, 221, 65, 238, 57, 237, 199, 16, 10, 33, 185, 8, 244, 184, 243, 139, 5, 87, 252, 245, 24, 225, 37, 154, 163, 142}

	msg := &network.TransactionMessage{Extrinsics: []types.Extrinsic{ext}}

	err = s.ProcessTransactionMessage(msg)
	require.Nil(t, err)

	bsTx := s.transactionQueue.Peek()
	bsTxExt := []byte(bsTx.Extrinsic)

	if !reflect.DeepEqual(ext, bsTxExt) {
		t.Error(
			"received unexpected transaction extrinsic",
			"\nexpected:", ext,
			"\nreceived:", bsTxExt,
		)
	}
}
