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

	"github.com/ChainSafe/gossamer/tests"

	"github.com/ChainSafe/gossamer/common/transaction"
	"github.com/ChainSafe/gossamer/core/types"
	"github.com/ChainSafe/gossamer/keystore"
	"github.com/ChainSafe/gossamer/p2p"
	"github.com/ChainSafe/gossamer/runtime"
)

var TestMessageTimeout = 2 * time.Second

func TestStartService(t *testing.T) {
	rt := runtime.NewTestRuntime(t, tests.POLKADOT_RUNTIME)

	cfg := &Config{
		Runtime:  rt,
		Keystore: keystore.NewKeystore(),
		MsgRec:   make(chan p2p.Message),
		MsgSend:  make(chan p2p.Message),
	}

	s, err := NewService(cfg)
	if err != nil {
		t.Fatal(err)
	}

	err = s.Start()
	if err != nil {
		t.Fatal(err)
	}

	s.Stop()
}

func TestValidateBlock(t *testing.T) {
	rt := runtime.NewTestRuntime(t, tests.POLKADOT_RUNTIME)

	cfg := &Config{
		Runtime:  rt,
		Keystore: keystore.NewKeystore(),
	}

	s, err := NewService(cfg)
	if err != nil {
		t.Fatal(err)
	}

	// https://github.com/paritytech/substrate/blob/426c26b8bddfcdbaf8d29f45b128e0864b57de1c/core/test-runtime/src/system.rs#L371
	data := []byte{69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 4, 179, 38, 109, 225, 55, 210, 10, 93, 15, 243, 166, 64, 30, 181, 113, 39, 82, 95, 217, 178, 105, 55, 1, 240, 191, 90, 138, 133, 63, 163, 235, 224, 3, 23, 10, 46, 117, 151, 183, 183, 227, 216, 76, 5, 57, 29, 19, 154, 98, 177, 87, 231, 135, 134, 216, 192, 130, 242, 157, 207, 76, 17, 19, 20, 0, 0}

	// `core_execute_block` will throw error, no expected result
	err = s.validateBlock(data)
	if err != nil {
		t.Fatal(err)
	}
}

func TestValidateTransaction(t *testing.T) {
	rt := runtime.NewTestRuntime(t, tests.POLKADOT_RUNTIME)

	cfg := &Config{
		Runtime:  rt,
		Keystore: keystore.NewKeystore(),
	}

	s, err := NewService(cfg)
	if err != nil {
		t.Fatal(err)
	}

	// https://github.com/paritytech/substrate/blob/5420de3face1349a97eb954ae71c5b0b940c31de/core/transaction-pool/src/tests.rs#L95
	tx := []byte{1, 212, 53, 147, 199, 21, 253, 211, 28, 97, 20, 26, 189, 4, 169, 159, 214, 130, 44, 133, 88, 133, 76, 205, 227, 154, 86, 132, 231, 165, 109, 162, 125, 142, 175, 4, 21, 22, 135, 115, 99, 38, 201, 254, 161, 126, 37, 252, 82, 135, 97, 54, 147, 201, 18, 144, 156, 178, 38, 170, 71, 148, 242, 106, 72, 69, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 216, 5, 113, 87, 87, 40, 221, 120, 247, 252, 137, 201, 74, 231, 222, 101, 85, 108, 102, 39, 31, 190, 210, 14, 215, 124, 19, 160, 180, 203, 54, 110, 167, 163, 149, 45, 12, 108, 80, 221, 65, 238, 57, 237, 199, 16, 10, 33, 185, 8, 244, 184, 243, 139, 5, 87, 252, 245, 24, 225, 37, 154, 163, 142}

	validity, err := s.validateTransaction(tx)
	if err != nil {
		t.Fatal(err)
	}

	// https://github.com/paritytech/substrate/blob/ea2644a235f4b189c8029b9c9eac9d4df64ee91e/core/test-runtime/src/system.rs#L190
	expected := &transaction.Validity{
		Priority: 69,
		Requires: [][]byte{{}},
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
	rt := runtime.NewTestRuntime(t, tests.POLKADOT_RUNTIME)

	msgSend := make(chan p2p.Message)
	newBlocks := make(chan types.Block)

	cfg := &Config{
		Runtime:   rt,
		MsgSend:   msgSend, // message channel from core service to p2p service
		Keystore:  keystore.NewKeystore(),
		NewBlocks: newBlocks,
	}

	s, err := NewService(cfg)
	if err != nil {
		t.Fatal(err)
	}

	err = s.Start()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Stop()

	// simulate block sent from BABE session
	newBlocks <- types.Block{
		Header: &types.Header{
			Number: big.NewInt(0),
		},
	}

	select {
	case msg := <-msgSend:
		msgType := msg.GetType()
		if !reflect.DeepEqual(msgType, p2p.BlockAnnounceMsgType) {
			t.Error(
				"received unexpected message type",
				"\nexpected:", p2p.BlockAnnounceMsgType,
				"\nreceived:", msgType,
			)
		}
	case <-time.After(TestMessageTimeout):
		t.Error("timeout waiting for message")
	}
}

func TestProcessBlockAnnounceMessage(t *testing.T) {
	rt := runtime.NewTestRuntime(t, tests.POLKADOT_RUNTIME)

	msgRec := make(chan p2p.Message)
	msgSend := make(chan p2p.Message)

	cfg := &Config{
		Runtime:  rt,
		MsgRec:   msgRec,
		MsgSend:  msgSend,
		Keystore: keystore.NewKeystore(),
	}

	s, err := NewService(cfg)
	if err != nil {
		t.Fatal(err)
	}

	err = s.Start()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Stop()

	blockAnnounce := &p2p.BlockAnnounceMessage{
		Number: big.NewInt(1),
	}

	// simulate mssage sent from p2p service
	msgRec <- blockAnnounce

	select {
	case msg := <-msgSend:
		msgType := msg.GetType()
		if msgType != p2p.BlockRequestMsgType {
			t.Error(
				"received unexpected message type",
				"\nexpected:", p2p.BlockRequestMsgType,
				"\nreceived:", msgType,
			)
		}
	case <-time.After(TestMessageTimeout):
		t.Error("timeout waiting for message")
	}
}

func TestProcessBlockResponseMessage(t *testing.T) {
	rt := runtime.NewTestRuntime(t, tests.POLKADOT_RUNTIME)

	cfg := &Config{
		Runtime:  rt,
		Keystore: keystore.NewKeystore(),
	}

	s, err := NewService(cfg)
	if err != nil {
		t.Fatal(err)
	}

	err = s.Start()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Stop()

	// https://github.com/paritytech/substrate/blob/426c26b8bddfcdbaf8d29f45b128e0864b57de1c/core/test-runtime/src/system.rs#L371
	data := []byte{69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 4, 179, 38, 109, 225, 55, 210, 10, 93, 15, 243, 166, 64, 30, 181, 113, 39, 82, 95, 217, 178, 105, 55, 1, 240, 191, 90, 138, 133, 63, 163, 235, 224, 3, 23, 10, 46, 117, 151, 183, 183, 227, 216, 76, 5, 57, 29, 19, 154, 98, 177, 87, 231, 135, 134, 216, 192, 130, 242, 157, 207, 76, 17, 19, 20, 0, 0}

	blockResponse := &p2p.BlockResponseMessage{Data: data}

	err = s.ProcessBlockResponseMessage(blockResponse)
	if err != nil {
		t.Error(err)
	}
}

func TestProcessTransactionMessage(t *testing.T) {
	rt := runtime.NewTestRuntime(t, tests.POLKADOT_RUNTIME)

	cfg := &Config{
		Runtime:  rt,
		Keystore: keystore.NewKeystore(),
	}

	s, err := NewService(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Stop()

	// https://github.com/paritytech/substrate/blob/5420de3face1349a97eb954ae71c5b0b940c31de/core/transaction-pool/src/tests.rs#L95
	ext := []byte{1, 212, 53, 147, 199, 21, 253, 211, 28, 97, 20, 26, 189, 4, 169, 159, 214, 130, 44, 133, 88, 133, 76, 205, 227, 154, 86, 132, 231, 165, 109, 162, 125, 142, 175, 4, 21, 22, 135, 115, 99, 38, 201, 254, 161, 126, 37, 252, 82, 135, 97, 54, 147, 201, 18, 144, 156, 178, 38, 170, 71, 148, 242, 106, 72, 69, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 216, 5, 113, 87, 87, 40, 221, 120, 247, 252, 137, 201, 74, 231, 222, 101, 85, 108, 102, 39, 31, 190, 210, 14, 215, 124, 19, 160, 180, 203, 54, 110, 167, 163, 149, 45, 12, 108, 80, 221, 65, 238, 57, 237, 199, 16, 10, 33, 185, 8, 244, 184, 243, 139, 5, 87, 252, 245, 24, 225, 37, 154, 163, 142}

	msg := &p2p.TransactionMessage{Extrinsics: []types.Extrinsic{ext}}

	err = s.ProcessTransactionMessage(msg)
	if err != nil {
		t.Fatal(err)
	}

	bsTx := s.bs.PeekFromTxQueue()
	bsTxExt := []byte(*bsTx.Extrinsic)

	if !reflect.DeepEqual(ext, bsTxExt) {
		t.Error(
			"received unexpected transaction extrinsic",
			"\nexpected:", ext,
			"\nreceived:", bsTxExt,
		)
	}
}
