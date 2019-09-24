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
	"path/filepath"
	"testing"

	"github.com/ChainSafe/gossamer/consensus/babe"
	"github.com/ChainSafe/gossamer/p2p"
	"github.com/ChainSafe/gossamer/runtime"
	"github.com/ChainSafe/gossamer/trie"
)

const POLKADOT_RUNTIME_FP string = "../polkadot_runtime.wasm"

func newRuntime(t *testing.T) *runtime.Runtime {
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
	p2pcfg := &p2p.Config{
		BootstrapNodes: []string{},
		Port:           7001,
	}
	p, err := p2p.NewService(p2pcfg)
	if err != nil {
		t.Fatal(err)
	}
	mgr := NewService(rt, b, p.MsgChan())
	e := mgr.Start()
	err = <-e
	if err != nil {
		t.Fatal(err)
	}
}

func TestValidateTransaction(t *testing.T) {
	rt := newRuntime(t)
	mgr := NewService(rt, nil, make(chan p2p.Message))
	// from https://github.com/paritytech/substrate/blob/5420de3face1349a97eb954ae71c5b0b940c31de/core/transaction-pool/src/tests.rs#L95
	ext := []byte{1, 212, 53, 147, 199, 21, 253, 211, 28, 97, 20, 26, 189, 4, 169, 159, 214, 130, 44, 133, 88, 133, 76, 205, 227, 154, 86, 132, 231, 165, 109, 162, 125, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 210, 0, 0, 0, 0, 0, 0, 0, 48, 204, 253, 26, 48, 22, 196, 90, 229, 200, 40, 124, 6, 90, 250, 20, 175, 99, 56, 218, 6, 204, 43, 21, 251, 237, 207, 43, 4, 70, 87, 119, 70, 222, 229, 13, 1, 96, 97, 210, 174, 150, 225, 250, 180, 99, 23, 21, 72, 209, 94, 188, 114, 3, 65, 157, 85, 26, 48, 46, 206, 67, 218, 130}
	validity, err := mgr.validateTransaction(ext)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(validity)
}

func TestProcessTransaction(t *testing.T) {
	rt := newRuntime(t)
	b := babe.NewSession([32]byte{}, [64]byte{}, rt)
	mgr := NewService(rt, b, make(chan p2p.Message))
	ext := []byte{80, 131, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4, 0, 0}
	mgr.ProcessTransaction(ext)
}

func TestValidateBlock(t *testing.T) {
	rt := newRuntime(t)
	mgr := NewService(rt, nil, make(chan p2p.Message))
	// from https://github.com/paritytech/substrate/blob/426c26b8bddfcdbaf8d29f45b128e0864b57de1c/core/test-runtime/src/system.rs#L371
	data := []byte{69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 69, 4, 179, 38, 109, 225, 55, 210, 10, 93, 15, 243, 166, 64, 30, 181, 113, 39, 82, 95, 217, 178, 105, 55, 1, 240, 191, 90, 138, 133, 63, 163, 235, 224, 3, 23, 10, 46, 117, 151, 183, 183, 227, 216, 76, 5, 57, 29, 19, 154, 98, 177, 87, 231, 135, 134, 216, 192, 130, 242, 157, 207, 76, 17, 19, 20, 0, 0}
	ret, err := mgr.validateBlock(data)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("ret:", ret)

}
