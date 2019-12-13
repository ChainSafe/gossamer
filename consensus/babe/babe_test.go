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

package babe

import (
	"fmt"
	"io"
	"math"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/common"
	tx "github.com/ChainSafe/gossamer/common/transaction"
	"github.com/ChainSafe/gossamer/core/blocktree"
	"github.com/ChainSafe/gossamer/core/types"
	db "github.com/ChainSafe/gossamer/polkadb"
	"github.com/ChainSafe/gossamer/runtime"
	"github.com/ChainSafe/gossamer/trie"
)

const POLKADOT_RUNTIME_FP string = "../../substrate_test_runtime.compact.wasm"
const POLKADOT_RUNTIME_URL string = "https://github.com/noot/substrate/blob/add-blob/core/test-runtime/wasm/wasm32-unknown-unknown/release/wbuild/substrate-test-runtime/substrate_test_runtime.compact.wasm?raw=true"

var zeroHash, _ = common.HexToHash("0x00")

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

	r, err := runtime.NewRuntimeFromFile(fp, tt, nil)
	if err != nil {
		t.Fatal(err)
	} else if r == nil {
		t.Fatal("did not create new VM")
	}

	return r
}

func TestCalculateThreshold(t *testing.T) {
	// C = 1
	var C1 uint64 = 1
	var C2 uint64 = 1
	var authorityIndex uint64 = 0
	authorityWeights := []uint64{1, 1, 1}

	expected := new(big.Int).Lsh(big.NewInt(1), 128)

	threshold, err := calculateThreshold(C1, C2, authorityIndex, authorityWeights)
	if err != nil {
		t.Fatal(err)
	}

	if threshold.Cmp(expected) != 0 {
		t.Fatalf("Fail: got %d expected %d", threshold, expected)
	}

	// C = 1/2
	C2 = 2

	theta := float64(1) / float64(3)
	c := float64(C1) / float64(C2)
	pp := 1 - c
	pp_exp := math.Pow(pp, theta)
	p := 1 - pp_exp
	p_rat := new(big.Rat).SetFloat64(p)
	q := new(big.Int).Lsh(big.NewInt(1), 128)
	expected = q.Mul(q, p_rat.Num()).Div(q, p_rat.Denom())

	threshold, err = calculateThreshold(C1, C2, authorityIndex, authorityWeights)
	if err != nil {
		t.Fatal(err)
	}

	if threshold.Cmp(expected) != 0 {
		t.Fatalf("Fail: got %d expected %d", threshold, expected)
	}
}

func TestCalculateThreshold_AuthorityWeights(t *testing.T) {
	var C1 uint64 = 5
	var C2 uint64 = 17
	var authorityIndex uint64 = 3
	authorityWeights := []uint64{3, 1, 4, 6, 10}

	theta := float64(6) / float64(24)
	c := float64(C1) / float64(C2)
	pp := 1 - c
	pp_exp := math.Pow(pp, theta)
	p := 1 - pp_exp
	p_rat := new(big.Rat).SetFloat64(p)
	q := new(big.Int).Lsh(big.NewInt(1), 128)
	expected := q.Mul(q, p_rat.Num()).Div(q, p_rat.Denom())

	threshold, err := calculateThreshold(C1, C2, authorityIndex, authorityWeights)
	if err != nil {
		t.Fatal(err)
	}

	if threshold.Cmp(expected) != 0 {
		t.Fatalf("Fail: got %d expected %d", threshold, expected)
	}
}

func TestRunLottery(t *testing.T) {
	rt := newRuntime(t)

	cfg := &SessionConfig{
		Runtime: rt,
	}

	babesession, err := NewSession(cfg)
	if err != nil {
		t.Fatal(err)
	}

	babesession.authorityIndex = 0
	babesession.authorityData = []AuthorityData{
		{nil, 1}, {nil, 1}, {nil, 1},
	}
	conf := &BabeConfiguration{
		SlotDuration:       1000,
		EpochLength:        6,
		C1:                 3,
		C2:                 10,
		GenesisAuthorities: []AuthorityDataRaw{},
		Randomness:         0,
		SecondarySlots:     false,
	}
	babesession.config = conf

	_, err = babesession.runLottery(0)
	if err != nil {
		t.Fatal(err)
	}
}

func TestCalculateThreshold_Failing(t *testing.T) {
	var C1 uint64 = 5
	var C2 uint64 = 4
	var authorityIndex uint64 = 3
	authorityWeights := []uint64{3, 1, 4, 6, 10}

	_, err := calculateThreshold(C1, C2, authorityIndex, authorityWeights)
	if err == nil {
		t.Fatal("Fail: did not err for c>1")
	}
}

func TestConfigurationFromRuntime(t *testing.T) {
	rt := newRuntime(t)
	cfg := &SessionConfig{
		Runtime: rt,
	}

	babesession, err := NewSession(cfg)
	if err != nil {
		t.Fatal(err)
	}

	err = babesession.configurationFromRuntime()
	if err != nil {
		t.Fatal(err)
	}

	// see: https://github.com/paritytech/substrate/blob/7b1d822446982013fa5b7ad5caff35ca84f8b7d0/core/test-runtime/src/lib.rs#L621
	expected := &BabeConfiguration{
		SlotDuration:       1000,
		EpochLength:        6,
		C1:                 3,
		C2:                 10,
		GenesisAuthorities: []AuthorityDataRaw{},
		Randomness:         0,
		SecondarySlots:     false,
	}

	if babesession.config == expected {
		t.Errorf("Fail: got %v expected %v\n", babesession.config, expected)
	}
}

func TestMedian_OddLength(t *testing.T) {
	us := []uint64{3, 2, 1, 4, 5}
	res, err := median(us)
	if err != nil {
		t.Fatal(err)
	}

	var expected uint64 = 3

	if res != expected {
		t.Errorf("Fail: got %v expected %v\n", res, expected)
	}

}

func TestMedian_EvenLength(t *testing.T) {
	us := []uint64{1, 4, 2, 4, 5, 6}
	res, err := median(us)
	if err != nil {
		t.Fatal(err)
	}

	var expected uint64 = 4

	if res != expected {
		t.Errorf("Fail: got %v expected %v\n", res, expected)
	}

}

func TestSlotOffset_Failing(t *testing.T) {
	var st uint64 = 1000001
	var se uint64 = 1000000

	_, err := slotOffset(st, se)
	if err == nil {
		t.Fatal("Fail: did not err for c>1")
	}

}

func TestSlotOffset(t *testing.T) {
	var st uint64 = 1000000
	var se uint64 = 1000001

	res, err := slotOffset(st, se)
	if err != nil {
		t.Fatal(err)
	}

	var expected uint64 = 1

	if res != expected {
		t.Errorf("Fail: got %v expected %v\n", res, expected)
	}

}

func createFlatBlockTree(t *testing.T, depth int) *blocktree.BlockTree {

	genesisBlock := types.Block{
		Header: types.BlockHeaderWithHash{
			ParentHash: zeroHash,
			Number:     big.NewInt(0),
			Hash:       common.Hash{0x00},
		},
		Body: types.BlockBody{},
	}

	genesisBlock.SetBlockArrivalTime(uint64(1000))

	d := &db.BlockDB{
		Db: db.NewMemDatabase(),
	}

	bt := blocktree.NewBlockTreeFromGenesis(genesisBlock, d)
	previousHash := genesisBlock.Header.Hash
	previousAT := genesisBlock.GetBlockArrivalTime()

	for i := 1; i <= depth; i++ {
		hex := fmt.Sprintf("%06x", i)

		hash, err := common.HexToHash("0x" + hex)

		if err != nil {
			t.Error(err)
		}

		block := types.Block{
			Header: types.BlockHeaderWithHash{
				ParentHash: previousHash,
				Hash:       hash,
				Number:     big.NewInt(int64(i)),
			},
			Body: types.BlockBody{},
		}

		block.SetBlockArrivalTime(previousAT + uint64(1000))

		bt.AddBlock(block)
		previousHash = hash
		previousAT = block.GetBlockArrivalTime()
	}

	return bt

}

func TestSlotTime(t *testing.T) {
	rt := newRuntime(t)
	bt := createFlatBlockTree(t, 100)
	cfg := &SessionConfig{
		Runtime: rt,
	}

	babesession, err := NewSession(cfg)
	if err != nil {
		t.Fatal(err)
	}

	err = babesession.configurationFromRuntime()
	if err != nil {
		t.Fatal(err)
	}

	res, err := babesession.slotTime(103, bt, 20)
	if err != nil {
		t.Fatal(err)
	}

	var expected uint64 = 104000

	if res != expected {
		t.Errorf("Fail: got %v expected %v\n", res, expected)
	}

}

func TestStart(t *testing.T) {
	rt := newRuntime(t)
	cfg := &SessionConfig{
		Runtime: rt,
	}

	babesession, err := NewSession(cfg)
	if err != nil {
		t.Fatal(err)
	}

	babesession.authorityIndex = 0
	babesession.authorityData = []AuthorityData{{nil, 1}}
	conf := &BabeConfiguration{
		SlotDuration:       1,
		EpochLength:        6,
		C1:                 1,
		C2:                 10,
		GenesisAuthorities: []AuthorityDataRaw{},
		Randomness:         0,
		SecondarySlots:     false,
	}
	babesession.config = conf

	err = babesession.Start()
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Duration(conf.SlotDuration) * time.Duration(conf.EpochLength) * time.Millisecond)
}

func TestBabeAnnounceMessage(t *testing.T) {
	rt := newRuntime(t)

	newBlocks := make(chan types.Block)

	cfg := &SessionConfig{
		Runtime:   rt,
		NewBlocks: newBlocks,
	}

	babesession, err := NewSession(cfg)
	if err != nil {
		t.Fatal(err)
	}

	babesession.authorityIndex = 0
	babesession.authorityData = []AuthorityData{
		{nil, 1}, {nil, 1}, {nil, 1},
	}

	err = babesession.Start()
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Duration(babesession.config.SlotDuration) * time.Duration(babesession.config.EpochLength) * time.Millisecond)

	for i := 0; i < int(babesession.config.EpochLength); i++ {
		block := <-newBlocks
		blockNumber := big.NewInt(int64(i))
		if !reflect.DeepEqual(block.Header.Number, blockNumber) {
			t.Fatalf("Didn't receive the correct block: %+v\nExpected block: %+v", block.Header.Number, blockNumber)
		}
	}

}

func TestBuildBlock(t *testing.T) {
	rt := newRuntime(t)
	babesession, err := NewSession([32]byte{}, [64]byte{}, rt, nil)
	if err != nil {
		t.Fatal(err)
	}
	err = babesession.configurationFromRuntime()
	if err != nil {
		t.Fatal(err)
	}

	// Push an extrinsic to the r
	e1 := types.Extrinsic([]byte{1, 1, 142, 175, 4, 21, 22, 135, 115, 99, 38, 201, 254, 161, 126, 37, 252, 82, 135, 97, 54, 147, 201, 18, 144, 156, 178, 38, 170, 71, 148, 242, 106, 72, 212, 53, 147, 199, 21, 253, 211, 28, 97, 20, 26, 189, 4, 169, 159, 214, 130, 44, 133, 88, 133, 76, 205, 227, 154, 86, 132, 231, 165, 109, 162, 125, 27, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 144, 113, 116, 62, 181, 98, 184, 56, 117, 228, 121, 88, 45, 21, 26, 200, 248, 62, 155, 0, 183, 222, 15, 145, 160, 249, 135, 252, 180, 226, 194, 88, 48, 123, 247, 162, 47, 213, 161, 96, 27, 77, 76, 159, 198, 1, 62, 132, 58, 140, 191, 96, 198, 4, 32, 138, 215, 61, 78, 143, 18, 32, 207, 140})
	v1 := tx.Validity{Priority: 1}
	tx1 := tx.NewValidTransaction(&e1, &v1)
	babesession.PushToTxQueue(tx1)

	_, err = babesession.validateTransaction(e1)
	if err != nil {
		t.Fatal(err)
	}

	// Create a block to put the transactions
	zeroHash, err := common.HexToHash("0x00")
	if err != nil {
		t.Fatalf("Can't convert hex 0x00 to hash")
	}

	block := types.Block{
		Header: types.BlockHeaderWithHash{
			ParentHash: zeroHash,
			Number:     big.NewInt(0),
		},
		Body: types.BlockBody{},
	}

	// Create slot for block
	slot := Slot{
		start:    uint64(time.Now().Unix()),
		duration: uint64(10000000),
		number:   1,
	}

	resultBlock, err := babesession.buildBlock(block, slot, common.Hash{0x00})
	if err != nil {
		t.Fatal("buildblock test failed: ", err)
	}

	t.Log("Got back block: ", resultBlock)
}
