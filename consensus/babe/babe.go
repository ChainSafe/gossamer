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
	tx "github.com/ChainSafe/gossamer/common/transaction"
	"github.com/ChainSafe/gossamer/core/types"
	"github.com/ChainSafe/gossamer/keystore"
	"github.com/ChainSafe/gossamer/runtime"
	"github.com/ChainSafe/gossamer/state"
	log "github.com/ChainSafe/log15"
)

// Session contains the VRF keys for the validator
//nolint:structcheck
type Session struct {
	keystore       *keystore.Keystore
	rt             *runtime.Runtime
	state          *state.Service //nolint:unused
	config         *BabeConfiguration
	authorityIndex uint64
	authorityData  []AuthorityData
	epochThreshold *big.Int // validator threshold for this epoch
	txQueue        *tx.PriorityQueue
	isProducer     map[uint64]bool    // whether we are a block producer at a slot
	newBlocks      chan<- types.Block // send blocks to core service
}

type SessionConfig struct {
	Keystore  *keystore.Keystore
	Runtime   *runtime.Runtime
	NewBlocks chan<- types.Block
}

const MaxBlockSize uint = 4*1024*1024 + 512

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

	//TODO: finish implementation of build block
	go b.invokeBlockAuthoring()

	return nil
}

// PushToTxQueue adds a ValidTransaction to BABE's transaction queue
func (b *Session) PushToTxQueue(vt *tx.ValidTransaction) {
	b.txQueue.Insert(vt)
}

func (b *Session) PeekFromTxQueue() *tx.ValidTransaction {
	return b.txQueue.Peek()
}

func (b *Session) invokeBlockAuthoring() {
	// TODO: we might not actually be starting at slot 0, need to run median algorithm here
	var currentSlot uint64 = 0

	for ; currentSlot < b.config.EpochLength; currentSlot++ {
		// TODO: call buildBlock
		b.newBlocks <- types.Block{
			Header: &types.BlockHeaderWithHash{
				Number: big.NewInt(0),
			},
		}
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
func (b *Session) buildBlock(parent *types.BlockHeaderWithHash, slot Slot) (*types.Block, error) {
	log.Debug("build-block", "parent", parent, "slot", slot)

	// Initialize block
	encodedHeader, err := codec.Encode(parent)
	if err != nil {
		return nil, err
	}
	err = b.initializeBlock(encodedHeader)
	if err != nil {
		return nil, err
	}

	// TODO: inherents and extrinsics

	// Finalize block
	rawblock, err := b.finalizeBlock()
	if err != nil {
		return nil, err
	}

	rawblock.Number.Add(parent.Number, big.NewInt(1))

	withHash, err := rawblock.BlockHeaderWithHash()
	if err != nil {
		return nil, err
	}

	block := &types.Block{
		Header: withHash,
	}
	return block, nil
}
