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
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/ChainSafe/gossamer/codec"
	//"github.com/ChainSafe/gossamer/common"
	tx "github.com/ChainSafe/gossamer/common/transaction"
	"github.com/ChainSafe/gossamer/core/types"
	"github.com/ChainSafe/gossamer/keystore"
	"github.com/ChainSafe/gossamer/runtime"
	"github.com/ChainSafe/gossamer/state"
	log "github.com/ChainSafe/log15"
)

// Session contains the VRF keys for the validator
type Session struct {
	keystore       *keystore.Keystore
	rt             *runtime.Runtime
	state          *state.Service
	config         *BabeConfiguration
	authorityIndex uint64
	authorityData  []AuthorityData
	epochThreshold *big.Int // validator threshold for this epoch
	txQueue        *tx.PriorityQueue
	isProducer     map[uint64]bool     // whether we are a block producer at a slot
	newBlocks      chan<- *types.Block // send blocks to core service
}

type SessionConfig struct {
	Keystore  *keystore.Keystore
	Runtime   *runtime.Runtime
	NewBlocks chan<- *types.Block
}

const MAX_BLOCK_SIZE uint = 4*1024*1024 + 512

// NewSession returns a new Babe session using the provided VRF keys and runtime
func NewSession(cfg *SessionConfig) (*Session, error) {
	babeSession := &Session{
		keystore:   cfg.Keystore,
		rt:         cfg.Runtime,
		txQueue:    new(tx.PriorityQueue),
		isProducer: make(map[uint64]bool),
		newBlocks:  cfg.NewBlocks,
	}

	err := babeSession.configurationFromRuntime()
	if err != nil {
		return nil, err
	}

	return babeSession, nil
}

func (b *Session) Start() error {
	var i uint64 = 0
	var err error
	for ; i < b.config.EpochLength; i++ {
		b.isProducer[i], err = b.runLottery(i)
		if err != nil {
			return fmt.Errorf("BABE: error running slot lottery at slot %d: error %s", i, err)
		}
	}

	go b.invokeBlockAuthoring()

	return nil
}

// PushToTxQueue adds a ValidTransaction to BABE's transaction queue
func (b *Session) PushToTxQueue(vt *tx.ValidTransaction) {
	n := vt != nil
	fmt.Println("vt != nil: ", n)
	fmt.Println("vt: ", vt)
	b.txQueue.Insert(vt)
}

func (b *Session) PeekFromTxQueue() *tx.ValidTransaction {
	return b.txQueue.Peek()
}

func (b *Session) invokeBlockAuthoring() {
	// TODO: we might not actually be starting at slot 0, need to run median algorithm here
	var currentSlot uint64 = 0

	for ; currentSlot < b.config.EpochLength; currentSlot++ {
		//startTime := uint64(time.Now().Unix())

		// if b.isProducer[currentSlot] {
		// 	// TODO: implement build block
		// 	parent := b.state.Block.GetLatestBlock()
		// 	slot := Slot{
		// 		start:    startTime,
		// 		duration: b.config.SlotDuration,
		// 		number:   currentSlot,
		// 	}
		// 	block, err := b.buildBlock(parent, slot)
		// 	if err != nil {
		// 		return
		// 	}
		// 	b.newBlocks <- block
		// }

		time.Sleep(time.Millisecond * time.Duration(b.config.SlotDuration))
	}
}

// runs the slot lottery for a specific slot
// returns true if validator is authorized to produce a block for that slot, false otherwise
func (b *Session) runLottery(slot uint64) (bool, error) {
	slotBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(slotBytes, slot)
	vrfInput := append(slotBytes, b.config.Randomness)
	output, err := b.vrfSign(vrfInput)
	if err != nil {
		return false, err
	}

	output_int := new(big.Int).SetBytes(output)
	if b.epochThreshold == nil {
		err = b.setEpochThreshold()
		if err != nil {
			return false, err
		}
	}

	return output_int.Cmp(b.epochThreshold) > 0, nil
}

// sets the slot lottery threshold for the current epoch
func (b *Session) setEpochThreshold() error {
	var err error
	if b.config == nil {
		return errors.New("cannot set threshold: no babe config")
	}

	b.epochThreshold, err = calculateThreshold(b.config.C1, b.config.C2, b.authorityIndex, b.authorityWeights())
	if err != nil {
		return err
	}

	return nil
}

func (b *Session) authorityWeights() []uint64 {
	weights := make([]uint64, len(b.authorityData))
	for i, auth := range b.authorityData {
		weights[i] = auth.weight
	}
	return weights
}

// calculates the slot lottery threshold for the authority at authorityIndex.
// equation: threshold = 2^128 * (1 - (1-c)^(w_k/sum(w_i)))
// where k is the authority index, and sum(w_i) is the
// sum of all the authority weights
// see: https://github.com/paritytech/substrate/blob/master/core/consensus/babe/src/lib.rs#L1022
func calculateThreshold(C1, C2, authorityIndex uint64, authorityWeights []uint64) (*big.Int, error) {
	c := float64(C1) / float64(C2)
	if c > 1 {
		return nil, errors.New("invalid C1/C2: greater than 1")
	}

	// sum(w_i)
	var sum uint64 = 0
	for _, weight := range authorityWeights {
		sum += weight
	}

	if sum == 0 {
		return nil, errors.New("invalid authority weights: sums to zero")
	}

	// w_k/sum(w_i)
	theta := float64(authorityWeights[authorityIndex]) / float64(sum)

	// (1-c)^(w_k/sum(w_i)))
	pp := 1 - c
	pp_exp := math.Pow(pp, theta)

	// 1 - (1-c)^(w_k/sum(w_i)))
	p := 1 - pp_exp
	p_rat := new(big.Rat).SetFloat64(p)

	// 1 << 128
	q := new(big.Int).Lsh(big.NewInt(1), 128)

	// (1 << 128) * (1 - (1-c)^(w_k/sum(w_i)))
	return q.Mul(q, p_rat.Num()).Div(q, p_rat.Denom()), nil
}

func (b *Session) vrfSign(input []byte) ([]byte, error) {
	// TOOD: return VRF output and proof
	out := make([]byte, 32)
	_, err := rand.Read(out)
	return out, err
}

// Block Build
func (b *Session) buildBlock(parent types.BlockHeaderWithHash, slot Slot) (*types.Block, error) {
	// Assign the parent block's hash
	//parentBlockHeader := chainBest.Header
	// TODO: We're assuming parent already has hash as runtime call doesn't exist
	// parentBlockHash, err := b.blockHashFromIdFromRuntime(parentBlockHeader.Number.Bytes())

	var newBlock types.Block
	// Assign values to headers of the new block
	newBlock.Header.ParentHash = parent.Hash
	newBlock.Header.Number = big.NewInt(0)
	//newBlock.Header.Number = newBlockNum.Add(newBlockNum, parent.Number)

	// Initialize block through runtime
	encodedHeader, err := codec.Encode(&newBlock.Header)
	if err != nil {
		return nil, err
	}
	err = b.initializeBlockFromRuntime(encodedHeader)
	if err != nil {
		return nil, err
	}

	// Calling BlockBuilder_inherent_extrinsics using encoded data
	// extrinsicsArray, err := b.inherentExtrinsicsFromRuntime([]byte{8, 102, 105, 110, 97, 108, 110, 117, 109, 32, 1, 0, 0, 0, 0, 0, 0, 0, 116, 105, 109, 115, 116, 97, 112, 48, 32, 5, 0, 0, 0, 0, 0, 0, 0})
	// if err != nil {
	// 	return nil, err
	// }
	// log.Debug("Returning from BlockBuilder_inherent_extrinsics call", "extrinsics array", extrinsicsArray)

	// Loop through inherents in the queue and apply them to the block through runtime
	// var blockBody types.BlockBody = make(types.BlockBody, 0, MAX_BLOCK_SIZE)
	// for _, extrinsic := range extrinsicsArray {
	// 	err = b.applyExtrinsicFromRuntime(extrinsic)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	log.Debug("Applied extrinsic", extrinsic)
	// }

	// log.Debug("Returning from BlockBuilder_apply_extrinsic calls")

	blockBody := []byte{}

	// Add Extrinsics to the block through runtime until block is full
	var extrinsic types.Extrinsic

	//for !blockIsFull(blockBody) && !endOfSlot(slot) {
	//extrinsic = b.nextReadyExtrinsic()
	extrinsic = []byte{1, 212, 53, 147, 199, 21, 253, 211, 28, 97, 20, 26, 189, 4, 169, 159, 214, 130, 44, 133, 88, 133, 76, 205, 227, 154, 86, 132, 231, 165, 109, 162, 125, 142, 175, 4, 21, 22, 135, 115, 99, 38, 201, 254, 161, 126, 37, 252, 82, 135, 97, 54, 147, 201, 18, 144, 156, 178, 38, 170, 71, 148, 242, 106, 72, 69, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 216, 5, 113, 87, 87, 40, 221, 120, 247, 252, 137, 201, 74, 231, 222, 101, 85, 108, 102, 39, 31, 190, 210, 14, 215, 124, 19, 160, 180, 203, 54, 110, 167, 163, 149, 45, 12, 108, 80, 221, 65, 238, 57, 237, 199, 16, 10, 33, 185, 8, 244, 184, 243, 139, 5, 87, 252, 245, 24, 225, 37, 154, 163, 142}

	//fmt.Println("Applying Extrinsic", extrinsic)
	err = b.applyExtrinsicFromRuntime(extrinsic)
	if err != nil {
		return nil, err
	}

	// Add the extrinsic to the blockbody
	blockBody = append(blockBody, extrinsic...)

	// Drop included extrinsic
	b.txQueue.Pop()
	log.Debug("build_block applied extrinsic", "extrinsic", extrinsic)
	//}

	//return &types.Block{}, nil

	// log.Debug("Added Extrinsics to the block")

	// Finalize block through runtime
	blockHeaderPointer, err := b.finalizeBlockFromRuntime(extrinsic)
	if err != nil {
		return nil, err
	}
	newBlock.Header = *blockHeaderPointer
	newBlock.Body = blockBody
	return &newBlock, nil
}

func blockIsFull(blockBody types.BlockBody) bool {
	return uint(len(blockBody)) == MAX_BLOCK_SIZE
}

func endOfSlot(slot Slot) bool {
	return uint64(time.Now().Unix()) > slot.start+slot.duration
}

func (b *Session) nextReadyExtrinsic() types.Extrinsic {
	transaction := b.txQueue.Peek()
	return *transaction.Extrinsic
}
