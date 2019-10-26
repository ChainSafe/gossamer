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
	"bytes"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	tx "github.com/ChainSafe/gossamer/common/transaction"
	"github.com/ChainSafe/gossamer/consensus/babe"
	"github.com/ChainSafe/gossamer/p2p"
	"github.com/ChainSafe/gossamer/runtime"
	"github.com/ChainSafe/gossamer/trie"
)

const POLKADOT_RUNTIME_FP string = "../substrate_test_runtime.compact.wasm"
const POLKADOT_RUNTIME_URL string = "https://github.com/noot/substrate/blob/david/latest-blob/target/wasm32-unknown-unknown/release/wbuild/substrate-test-runtime/substrate_test_runtime.compact.wasm?raw=true"

// getRuntimeBlob checks if the polkadot runtime wasm file exists and if not, it fetches it from github
func getRuntimeBlob() (n int64, err error) {
	if Exists(POLKADOT_RUNTIME_FP) {
		return 0, nil
	}

	out, err := os.Create(POLKADOT_RUNTIME_FP)
	if err != nil {
		return 0, err
	}
	defer out.Close()

	resp, err := http.Get(POLKADOT_RUNTIME_URL)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	n, err = io.Copy(out, resp.Body)
	return n, err
}

// Exists reports whether the named file or directory exists.
func Exists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func newRuntime(t *testing.T) *runtime.Runtime {
	_, err := getRuntimeBlob()
	if err != nil {
		t.Fatalf("Fail: could not get polkadot runtime")
	}

	fp, err := filepath.Abs(POLKADOT_RUNTIME_FP)
	if err != nil {
		t.Fatal("could not create filepath")
	}

	tt := &trie.Trie{}

	r, err := runtime.NewRuntime(fp, tt)
	if err != nil {
		t.Fatal(err)
	} else if r == nil {
		t.Fatal("did not create new VM")
	}

	return r
}

func TestNewService_Start(t *testing.T) {
	rt := newRuntime(t)
	b := babe.NewSession([32]byte{}, [64]byte{}, rt)
	msgChan := make(chan []byte)

	mgr := NewService(rt, b, msgChan)

	err := mgr.Start()
	if err != nil {
		t.Fatal(err)
	}
}

func TestValidateTransaction(t *testing.T) {
	rt := newRuntime(t)
	mgr := NewService(rt, nil, make(chan []byte))
	// from https://github.com/paritytech/substrate/blob/5420de3face1349a97eb954ae71c5b0b940c31de/core/transaction-pool/src/tests.rs#L95
	// added:
	// let utx = Transfer {
	//  from: AccountKeyring::Alice.into(),
	//  to: AccountKeyring::Bob.into(),
	//  amount: 69,
	//  nonce: 0,
	// }.into_signed_tx();
	// println!("extrinsic: {:?}", &utx.encode());
	// at line 377
	raw := []byte{1, 1, 142, 175, 4, 21, 22, 135, 115, 99, 38, 201, 254, 161, 126, 37, 252, 82, 135, 97, 54, 147, 201, 18, 144, 156, 178, 38, 170, 71, 148, 242, 106, 72, 212, 53, 147, 199, 21, 253, 211, 28, 97, 20, 26, 189, 4, 169, 159, 214, 130, 44, 133, 88, 133, 76, 205, 227, 154, 86, 132, 231, 165, 109, 162, 125, 27, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 144, 113, 116, 62, 181, 98, 184, 56, 117, 228, 121, 88, 45, 21, 26, 200, 248, 62, 155, 0, 183, 222, 15, 145, 160, 249, 135, 252, 180, 226, 194, 88, 48, 123, 247, 162, 47, 213, 161, 96, 27, 77, 76, 159, 198, 1, 62, 132, 58, 140, 191, 96, 198, 4, 32, 138, 215, 61, 78, 143, 18, 32, 207, 140}
	validity, err := mgr.validateTransaction(raw)
	if err != nil {
		t.Fatal(err)
	}

	// see: https://github.com/paritytech/substrate/blob/ea2644a235f4b189c8029b9c9eac9d4df64ee91e/core/test-runtime/src/system.rs#L190
	expected := &tx.Validity{
		Priority: 7037,
		Requires: [][]byte{{}},
		// Provides is the concatenation of from and nonce: see https://github.com/paritytech/substrate/blob/ea2644a235f4b189c8029b9c9eac9d4df64ee91e/core/test-runtime/src/system.rs#L173
		Provides:  [][]byte{{1, 142, 175, 4, 21, 22, 135, 115, 99, 38, 201, 254, 161, 126, 37, 252, 82, 135, 97, 54, 147, 201, 18, 144, 156, 178, 38, 170, 71, 148, 242, 106, 0, 0, 0, 0, 0, 0, 0, 0}},
		Longevity: 64,
		Propagate: true,
	}

	if !reflect.DeepEqual(expected, validity) {
		t.Fatalf("Fail: \ngot %v \nexpected %v", validity, expected)
	}
}

func TestProcessTransaction(t *testing.T) {
	rt := newRuntime(t)
	b := babe.NewSession([32]byte{}, [64]byte{}, rt)
	mgr := NewService(rt, b, make(chan []byte))
	ext := []byte{1, 212, 53, 147, 199, 21, 253, 211, 28, 97, 20, 26, 189, 4, 169, 159, 214, 130, 44, 133, 88, 133, 76, 205, 227, 154, 86, 132, 231, 165, 109, 162, 125, 142, 175, 4, 21, 22, 135, 115, 99, 38, 201, 254, 161, 126, 37, 252, 82, 135, 97, 54, 147, 201, 18, 144, 156, 178, 38, 170, 71, 148, 242, 106, 72, 69, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 216, 5, 113, 87, 87, 40, 221, 120, 247, 252, 137, 201, 74, 231, 222, 101, 85, 108, 102, 39, 31, 190, 210, 14, 215, 124, 19, 160, 180, 203, 54, 110, 167, 163, 149, 45, 12, 108, 80, 221, 65, 238, 57, 237, 199, 16, 10, 33, 185, 8, 244, 184, 243, 139, 5, 87, 252, 245, 24, 225, 37, 154, 163, 142}
	err := mgr.ProcessTransaction(ext)
	if err != nil {
		t.Fatal(err)
	}
	// check if in babe tx queue
	tx := b.PeekFromTxQueue()
	if !bytes.Equal([]byte(*tx.Extrinsic), ext) {
		t.Fatalf("Fail: got %x expected %x", tx.Extrinsic, ext)
	}
}

func TestValidateBlock(t *testing.T) {
	rt := newRuntime(t)
	mgr := NewService(rt, nil, make(chan []byte))
	// from https://github.com/paritytech/substrate/blob/426c26b8bddfcdbaf8d29f45b128e0864b57de1c/core/test-runtime/src/system.rs#L371
	data := []byte{69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 4, 54, 80, 204, 15, 224, 203, 27, 130, 91, 15, 110, 82, 145, 92, 248, 242, 236, 17, 227, 227, 115, 89, 34, 29, 205, 53, 175, 178, 113, 20, 206, 110, 78, 159, 233, 247, 36, 126, 2, 254, 254, 226, 231, 207, 11, 15, 45, 199, 227, 220, 9, 119, 130, 151, 13, 58, 200, 34, 76, 141, 108, 196, 86, 171, 0, 4, 1, 212, 53, 147, 199, 21, 253, 211, 28, 97, 20, 26, 189, 4, 169, 159, 214, 130, 44, 133, 88, 133, 76, 205, 227, 154, 86, 132, 231, 165, 109, 162, 125, 142, 175, 4, 21, 22, 135, 115, 99, 38, 201, 254, 161, 126, 37, 252, 82, 135, 97, 54, 147, 201, 18, 144, 156, 178, 38, 170, 71, 148, 242, 106, 72, 69, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 196, 220, 94, 253, 195, 179, 200, 124, 115, 14, 141, 25, 124, 112, 110, 223, 205, 245, 176, 209, 140, 184, 144, 213, 195, 212, 83, 128, 141, 163, 50, 24, 106, 198, 184, 95, 205, 194, 170, 208, 39, 196, 133, 54, 151, 249, 113, 30, 139, 72, 92, 175, 136, 146, 50, 5, 30, 136, 170, 127, 113, 84, 9, 138}
	err := mgr.validateBlock(data)
	if err != nil {
		t.Fatal(err)
	}
}

func TestHandleMsg_Transaction(t *testing.T) {
	rt := newRuntime(t)
	b := babe.NewSession([32]byte{}, [64]byte{}, rt)
	msgChan := make(chan []byte)
	mgr := NewService(rt, b, msgChan)
	err := mgr.Start()
	if err != nil {
		t.Fatal(err)
	}

	// wait for mgr to start
	time.Sleep(time.Second)

	ext := []byte{1, 212, 53, 147, 199, 21, 253, 211, 28, 97, 20, 26, 189, 4, 169, 159, 214, 130, 44, 133, 88, 133, 76, 205, 227, 154, 86, 132, 231, 165, 109, 162, 125, 142, 175, 4, 21, 22, 135, 115, 99, 38, 201, 254, 161, 126, 37, 252, 82, 135, 97, 54, 147, 201, 18, 144, 156, 178, 38, 170, 71, 148, 242, 106, 72, 69, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 216, 5, 113, 87, 87, 40, 221, 120, 247, 252, 137, 201, 74, 231, 222, 101, 85, 108, 102, 39, 31, 190, 210, 14, 215, 124, 19, 160, 180, 203, 54, 110, 167, 163, 149, 45, 12, 108, 80, 221, 65, 238, 57, 237, 199, 16, 10, 33, 185, 8, 244, 184, 243, 139, 5, 87, 252, 245, 24, 225, 37, 154, 163, 142}
	msgChan <- append([]byte{p2p.TransactionMsgType}, ext...)

	// wait for message to be handled
	time.Sleep(time.Second)

	// check if in babe tx queue
	tx := b.PeekFromTxQueue()
	if tx == nil {
		t.Fatalf("Fail: got nil expected %x", ext)
	} else if !bytes.Equal([]byte(*tx.Extrinsic), ext) {
		t.Fatalf("Fail: got %x expected %x", tx.Extrinsic, ext)
	}
}

func TestHandleMsg_BlockResponse(t *testing.T) {
	rt := newRuntime(t)
	b := babe.NewSession([32]byte{}, [64]byte{}, rt)
	msgChan := make(chan []byte)
	mgr := NewService(rt, b, msgChan)
	e := make(chan error)
	go mgr.start(e)
	if err := <-e; err != nil {
		t.Fatal(err)
	}

	// wait for mgr to start
	time.Sleep(time.Second)

	block := []byte{135, 164, 7, 194, 167, 148, 232, 42, 43, 31, 129, 126, 245, 184, 31, 128, 77, 135, 236, 181, 149, 15, 158, 152, 180, 144, 97, 38, 175, 128, 168, 191, 8, 124, 133, 208, 123, 83, 2, 140, 236, 112, 61, 162, 138, 163, 168, 31, 141, 239, 214, 5, 132, 169, 92, 153, 159, 228, 235, 177, 123, 16, 18, 160, 65, 149, 94, 233, 157, 60, 170, 184, 112, 105, 250, 137, 90, 115, 21, 155, 136, 122, 34, 171, 142, 209, 18, 21, 164, 96, 6, 195, 242, 14, 117, 109, 174, 0, 8, 1, 142, 175, 4, 21, 22, 135, 115, 99, 38, 201, 254, 161, 126, 37, 252, 82, 135, 97, 54, 147, 201, 18, 144, 156, 178, 38, 170, 71, 148, 242, 106, 72, 212, 53, 147, 199, 21, 253, 211, 28, 97, 20, 26, 189, 4, 169, 159, 214, 130, 44, 133, 88, 133, 76, 205, 227, 154, 86, 132, 231, 165, 109, 162, 125, 27, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 74, 78, 107, 130, 204, 254, 91, 170, 81, 76, 22, 184, 29, 234, 105, 126, 57, 125, 174, 186, 201, 233, 163, 195, 50, 45, 179, 183, 137, 177, 28, 76, 1, 215, 187, 245, 22, 168, 173, 22, 97, 119, 181, 44, 187, 225, 80, 30, 146, 126, 168, 57, 59, 251, 117, 19, 143, 227, 23, 116, 118, 67, 165, 131, 1, 212, 53, 147, 199, 21, 253, 211, 28, 97, 20, 26, 189, 4, 169, 159, 214, 130, 44, 133, 88, 133, 76, 205, 227, 154, 86, 132, 231, 165, 109, 162, 125, 144, 181, 171, 32, 92, 105, 116, 201, 234, 132, 27, 230, 136, 134, 70, 51, 220, 156, 168, 163, 87, 132, 62, 234, 207, 35, 20, 100, 153, 101, 254, 34, 69, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 208, 121, 69, 0, 18, 118, 199, 230, 94, 51, 85, 134, 166, 255, 9, 226, 48, 182, 254, 199, 47, 61, 88, 218, 19, 39, 5, 157, 207, 19, 166, 90, 206, 130, 39, 18, 193, 77, 244, 244, 74, 124, 155, 110, 64, 10, 210, 142, 156, 22, 145, 58, 86, 89, 241, 3, 161, 133, 100, 60, 115, 190, 109, 137}

	msgChan <- append([]byte{p2p.BlockResponseMsgType}, block...)

	// wait for message to be handled
	time.Sleep(time.Second)
	if err := <-e; err != nil {
		t.Fatal(err)
	}
}
