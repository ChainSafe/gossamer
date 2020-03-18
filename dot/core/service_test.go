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
	"io/ioutil"
	"math/big"
	"reflect"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/core/types"
	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/optional"
	"github.com/ChainSafe/gossamer/lib/common/variadic"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/transaction"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/tests"
)

var TestMessageTimeout = 5 * time.Second

var genesisHeader = &types.Header{
	Number:    big.NewInt(0),
	StateRoot: trie.EmptyHash,
}

func newTestService(t *testing.T, cfg *Config) *Service {
	if cfg == nil {
		rt := runtime.NewTestRuntime(t, tests.POLKADOT_RUNTIME)

		cfg = &Config{
			Runtime:         rt,
			IsBabeAuthority: false,
		}
	}

	if cfg.Keystore == nil {
		cfg.Keystore = keystore.NewKeystore()
	}

	if cfg.MsgRec == nil {
		cfg.MsgRec = make(chan network.Message)
	}

	if cfg.MsgSend == nil {
		cfg.MsgSend = make(chan network.Message)
	}

	if cfg.NewBlocks == nil {
		cfg.NewBlocks = make(chan types.Block)
	}

	if cfg.SyncChan == nil {
		cfg.SyncChan = make(chan *big.Int)
	}

	stateSrvc := state.NewService("")
	stateSrvc.UseMemDB()

	err := stateSrvc.Initialize(genesisHeader, trie.NewEmptyTrie(nil))
	if err != nil {
		t.Fatal(err)
	}

	err = stateSrvc.Start()
	if err != nil {
		t.Fatal(err)
	}

	if cfg.BlockState == nil {
		cfg.BlockState = stateSrvc.Block
	}

	if cfg.StorageState == nil {
		cfg.StorageState = stateSrvc.Storage
	}

	s, err := NewService(cfg)
	if err != nil {
		t.Fatal(err)
	}

	return s
}

func TestStartService(t *testing.T) {
	s := newTestService(t, nil)
	err := s.Start()
	if err != nil {
		t.Fatal(err)
	}

	s.Stop()
}

func TestValidateBlock(t *testing.T) {
	s := newTestService(t, nil)

	// https://github.com/paritytech/substrate/blob/426c26b8bddfcdbaf8d29f45b128e0864b57de1c/core/test-runtime/src/system.rs#L371
	//data := []byte{69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 4, 179, 38, 109, 225, 55, 210, 10, 93, 15, 243, 166, 64, 30, 181, 113, 39, 82, 95, 217, 178, 105, 55, 1, 240, 191, 90, 138, 133, 63, 163, 235, 224, 3, 23, 10, 46, 117, 151, 183, 183, 227, 216, 76, 5, 57, 29, 19, 154, 98, 177, 87, 231, 135, 134, 216, 192, 130, 242, 157, 207, 76, 17, 19, 20, 0, 0}

	// data from build
	//data := []byte{248, 3, 150, 246, 83, 26, 208, 228, 51, 109, 203, 57, 167, 35, 100, 17, 171, 125, 223, 88, 96, 25, 64, 79, 255, 74, 100, 32, 231, 29, 226, 226, 1, 220, 221, 137, 146, 125, 138, 52, 142, 0, 37, 126, 30, 204, 134, 23, 244, 94, 219, 81, 24, 239, 255, 62, 162, 249, 150, 27, 42, 217, 183, 105, 10, 4, 49, 206, 94, 116, 215, 20, 21, 32, 171, 193, 27, 138, 104, 248, 132, 203, 29, 1, 181, 71, 106, 99, 118, 166, 89, 217, 58, 25, 156, 72, 132, 224, 216, 142, 4, 142, 218, 23, 170, 239, 196, 39, 200, 50, 234, 18, 8, 80, 141, 103, 163, 233, 101, 39, 190, 9, 149, 219, 116, 43, 92, 217, 26, 97, 8, 213, 1, 1, 66, 65, 66, 69, 232, 38, 172, 103, 76, 110, 14, 12, 228, 44, 199, 174, 255, 244, 27, 181, 98, 187, 123, 27, 70, 246, 102, 187, 206, 62, 182, 249, 106, 189, 236, 86, 148, 216, 9, 191, 143, 137, 199, 195, 62, 83, 215, 156, 207, 198, 246, 56, 252, 55, 171, 73, 194, 225, 190, 99, 4, 183, 128, 219, 74, 232, 60, 4, 57, 197, 40, 75, 30, 234, 128, 126, 3, 101, 165, 102, 246, 237, 70, 146, 28, 29, 211, 133, 207, 199, 151, 10, 150, 255, 67, 250, 177, 108, 131, 2, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 21, 1, 4, 66, 65, 66, 69, 38, 55, 18, 147, 126, 202, 180, 98, 163, 108, 119, 56, 198, 150, 116, 214, 54, 213, 170, 160, 7, 81, 100, 177, 68, 93, 145, 149, 34, 229, 161, 90, 197, 60, 148, 193, 137, 50, 143, 45, 65, 214, 215, 213, 198, 42, 66, 214, 139, 54, 95, 190, 22, 158, 218, 180, 80, 250, 231, 110, 4, 44, 135, 134, 1, 104, 4, 96, 3, 16, 110, 111, 111, 116, 1, 64, 103, 111, 115, 115, 97, 109, 101, 114, 95, 105, 115, 95, 99, 111, 111, 108, 0, 0, 0}
	// data 2 from build
	data := []byte{61, 148, 17, 62, 58, 76, 63, 19, 47, 217, 166, 46, 7, 123, 86, 254, 147, 232, 251, 27, 101, 26, 255, 184, 193, 110, 189, 247, 33, 130, 199, 60, 1, 220, 221, 137, 146, 125, 138, 52, 142, 0, 37, 126, 30, 204, 134, 23, 244, 94, 219, 81, 24, 239, 255, 62, 162, 249, 150, 27, 42, 217, 183, 105, 10, 4, 49, 206, 94, 116, 215, 20, 21, 32, 171, 193, 27, 138, 104, 248, 132, 203, 29, 1, 181, 71, 106, 99, 118, 166, 89, 217, 58, 25, 156, 72, 132, 224, 216, 142, 4, 142, 218, 23, 170, 239, 196, 39, 200, 50, 234, 18, 8, 80, 141, 103, 163, 233, 101, 39, 190, 9, 149, 219, 116, 43, 92, 217, 26, 97, 8, 213, 1, 1, 66, 65, 66, 69, 252, 234, 109, 121, 200, 157, 51, 47, 150, 87, 244, 157, 47, 110, 87, 247, 167, 92, 42, 102, 207, 33, 235, 57, 253, 187, 9, 108, 160, 194, 179, 39, 214, 99, 27, 208, 12, 189, 166, 132, 78, 12, 222, 92, 152, 158, 247, 65, 174, 80, 167, 76, 62, 123, 147, 35, 185, 57, 237, 142, 176, 183, 237, 1, 236, 7, 234, 108, 52, 72, 130, 246, 64, 118, 112, 89, 71, 64, 198, 136, 176, 191, 254, 173, 63, 243, 97, 14, 34, 28, 65, 223, 129, 59, 10, 14, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 21, 1, 4, 66, 65, 66, 69, 198, 205, 114, 252, 50, 10, 9, 72, 240, 15, 16, 54, 116, 242, 107, 201, 202, 175, 151, 151, 212, 239, 9, 46, 197, 176, 144, 248, 163, 99, 29, 123, 129, 14, 217, 77, 90, 18, 104, 50, 74, 108, 87, 120, 91, 104, 203, 174, 68, 7, 43, 32, 175, 163, 207, 141, 135, 215, 191, 17, 190, 56, 238, 134, 1, 104, 4, 96, 3, 16, 110, 111, 111, 116, 1, 64, 103, 111, 115, 115, 97, 109, 101, 114, 95, 105, 115, 95, 99, 111, 111, 108, 0, 0, 0}

	// `core_execute_block` will throw error, no expected result
	err := s.executeBlock(data)
	if err != nil {
		t.Fatal(err)
	}
}

func TestValidateTransaction(t *testing.T) {
	s := newTestService(t, nil)

	// https://github.com/paritytech/substrate/blob/5420de3face1349a97eb954ae71c5b0b940c31de/core/transaction-pool/src/tests.rs#L95
	tx := []byte{1, 212, 53, 147, 199, 21, 253, 211, 28, 97, 20, 26, 189, 4, 169, 159, 214, 130, 44, 133, 88, 133, 76, 205, 227, 154, 86, 132, 231, 165, 109, 162, 125, 142, 175, 4, 21, 22, 135, 115, 99, 38, 201, 254, 161, 126, 37, 252, 82, 135, 97, 54, 147, 201, 18, 144, 156, 178, 38, 170, 71, 148, 242, 106, 72, 69, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 216, 5, 113, 87, 87, 40, 221, 120, 247, 252, 137, 201, 74, 231, 222, 101, 85, 108, 102, 39, 31, 190, 210, 14, 215, 124, 19, 160, 180, 203, 54, 110, 167, 163, 149, 45, 12, 108, 80, 221, 65, 238, 57, 237, 199, 16, 10, 33, 185, 8, 244, 184, 243, 139, 5, 87, 252, 245, 24, 225, 37, 154, 163, 142}

	validity, err := s.ValidateTransaction(tx)
	if err != nil {
		t.Fatal(err)
	}

	// https://github.com/paritytech/substrate/blob/ea2644a235f4b189c8029b9c9eac9d4df64ee91e/core/test-runtime/src/system.rs#L190
	expected := &transaction.Validity{
		Priority: 69,
		Requires: [][]byte{},
		// https://github.com/paritytech/substrate/blob/ea2644a235f4b189c8029b9c9eac9d4df64ee91e/core/test-runtime/src/system.rs#L173
		Provides:  [][]byte{{146, 157, 61, 99, 63, 98, 30, 242, 128, 49, 150, 90, 140, 165, 187, 249}},
		Longevity: 64,
		Propagate: true,
	}

	if !reflect.DeepEqual(expected, validity) {
		t.Error(
			"received unexpected validity",
			"\nexpected:", expected,
			"\nreceived:", validity,
		)
	}
}

func TestAnnounceBlock(t *testing.T) {
	msgSend := make(chan network.Message)
	newBlocks := make(chan types.Block)

	cfg := &Config{
		NewBlocks: newBlocks,
		MsgSend:   msgSend,
	}

	s := newTestService(t, cfg)
	err := s.Start()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Stop()

	parent := &types.Header{
		Number:    big.NewInt(0),
		StateRoot: trie.EmptyHash,
	}

	// simulate block sent from BABE session
	newBlocks <- types.Block{
		Header: &types.Header{
			ParentHash: parent.Hash(),
			Number:     big.NewInt(1),
		},
		Body: &types.Body{},
	}

	select {
	case msg := <-msgSend:
		msgType := msg.GetType()
		if !reflect.DeepEqual(msgType, network.BlockAnnounceMsgType) {
			t.Error(
				"received unexpected message type",
				"\nexpected:", network.BlockAnnounceMsgType,
				"\nreceived:", msgType,
			)
		}
	case <-time.After(TestMessageTimeout):
		t.Error("timeout waiting for message")
	}
}

func TestProcessBlockResponseMessage(t *testing.T) {
	tt := trie.NewEmptyTrie(nil)
	rt := runtime.NewTestRuntimeWithTrie(t, tests.POLKADOT_RUNTIME, tt)

	kp, err := sr25519.GenerateKeypair()
	if err != nil {
		t.Fatal(err)
	}

	pubkey := kp.Public().Encode()
	err = tt.Put(tests.AuthorityDataKey, append([]byte{4}, pubkey...))
	if err != nil {
		t.Fatal(err)
	}

	ks := keystore.NewKeystore()
	ks.Insert(kp)

	cfg := &Config{
		Runtime:         rt,
		Keystore:        ks,
		IsBabeAuthority: false,
	}

	s := newTestService(t, cfg)

	hash := common.NewHash([]byte{0})
	body := optional.CoreBody{0xa, 0xb, 0xc, 0xd}

	parentHash := genesisHeader.Hash()
	stateRoot, err := common.HexToHash("0x2747ab7c0dc38b7f2afba82bd5e2d6acef8c31e09800f660b75ec84a7005099f")
	if err != nil {
		t.Fatal(err)
	}

	extrinsicsRoot, err := common.HexToHash("0x03170a2e7597b7b7e3d84c05391d139a62b157e78786d8c082f29dcf4c111314")
	if err != nil {
		t.Fatal(err)
	}

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
	if err != nil {
		t.Fatal(err)
	}

	res, err := s.blockState.GetHeader(header.Hash())
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(res, header) {
		t.Fatalf("Fail: got %v expected %v", res, header)
	}
}

func TestProcessTransactionMessage(t *testing.T) {
	tt := trie.NewEmptyTrie(nil)
	rt := runtime.NewTestRuntimeWithTrie(t, tests.POLKADOT_RUNTIME, tt)

	kp, err := sr25519.GenerateKeypair()
	if err != nil {
		t.Fatal(err)
	}

	pubkey := kp.Public().Encode()
	err = tt.Put(tests.AuthorityDataKey, append([]byte{4}, pubkey...))
	if err != nil {
		t.Fatal(err)
	}

	ks := keystore.NewKeystore()
	ks.Insert(kp)

	cfg := &Config{
		Runtime:          rt,
		Keystore:         ks,
		TransactionQueue: transaction.NewPriorityQueue(),
		IsBabeAuthority:  true,
	}

	s := newTestService(t, cfg)

	// https://github.com/paritytech/substrate/blob/5420de3face1349a97eb954ae71c5b0b940c31de/core/transaction-pool/src/tests.rs#L95
	ext := []byte{1, 212, 53, 147, 199, 21, 253, 211, 28, 97, 20, 26, 189, 4, 169, 159, 214, 130, 44, 133, 88, 133, 76, 205, 227, 154, 86, 132, 231, 165, 109, 162, 125, 142, 175, 4, 21, 22, 135, 115, 99, 38, 201, 254, 161, 126, 37, 252, 82, 135, 97, 54, 147, 201, 18, 144, 156, 178, 38, 170, 71, 148, 242, 106, 72, 69, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 216, 5, 113, 87, 87, 40, 221, 120, 247, 252, 137, 201, 74, 231, 222, 101, 85, 108, 102, 39, 31, 190, 210, 14, 215, 124, 19, 160, 180, 203, 54, 110, 167, 163, 149, 45, 12, 108, 80, 221, 65, 238, 57, 237, 199, 16, 10, 33, 185, 8, 244, 184, 243, 139, 5, 87, 252, 245, 24, 225, 37, 154, 163, 142}

	msg := &network.TransactionMessage{Extrinsics: []types.Extrinsic{ext}}

	err = s.ProcessTransactionMessage(msg)
	if err != nil {
		t.Fatal(err)
	}

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

func TestService_NotAuthority(t *testing.T) {
	cfg := &Config{
		Keystore:        keystore.NewKeystore(),
		IsBabeAuthority: false,
	}

	s := newTestService(t, cfg)
	if s.bs != nil {
		t.Fatal("Fail: should not have babe session")
	}
}

func TestService_CheckForRuntimeChanges(t *testing.T) {
	tt := trie.NewEmptyTrie(nil)
	rt := runtime.NewTestRuntimeWithTrie(t, tests.POLKADOT_RUNTIME, tt)

	kp, err := sr25519.GenerateKeypair()
	if err != nil {
		t.Fatal(err)
	}

	pubkey := kp.Public().Encode()
	err = tt.Put(tests.AuthorityDataKey, append([]byte{4}, pubkey...))
	if err != nil {
		t.Fatal(err)
	}

	ks := keystore.NewKeystore()
	ks.Insert(kp)

	cfg := &Config{
		Runtime:          rt,
		Keystore:         ks,
		TransactionQueue: transaction.NewPriorityQueue(),
		IsBabeAuthority:  false,
	}

	s := newTestService(t, cfg)

	_, err = tests.GetRuntimeBlob(tests.TESTS_FP, tests.TEST_WASM_URL)
	if err != nil {
		t.Fatal(err)
	}

	testRuntime, err := ioutil.ReadFile(tests.TESTS_FP)
	if err != nil {
		t.Fatal(err)
	}

	err = s.storageState.SetStorage([]byte(":code"), testRuntime)
	if err != nil {
		t.Fatal(err)
	}

	err = s.checkForRuntimeChanges()
	if err != nil {
		t.Fatal(err)
	}
}

func addTestBlocksToState(t *testing.T, depth int, blockState BlockState) {
	previousHash := blockState.BestBlockHash()
	previousNum, err := blockState.BestBlockNumber()
	if err != nil {
		t.Fatal(err)
	}

	for i := 1; i <= depth; i++ {
		block := &types.Block{
			Header: &types.Header{
				ParentHash: previousHash,
				Number:     big.NewInt(int64(i)).Add(previousNum, big.NewInt(int64(i))),
			},
			Body: &types.Body{},
		}

		previousHash = block.Header.Hash()

		err := blockState.AddBlock(block)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestService_ProcessBlockRequest(t *testing.T) {
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
