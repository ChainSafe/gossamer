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
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"math/big"
	"time"

	scale "github.com/ChainSafe/gossamer/codec"
	tx "github.com/ChainSafe/gossamer/common/transaction"
	"github.com/ChainSafe/gossamer/core/types"
	"github.com/ChainSafe/gossamer/crypto/sr25519"
	"github.com/ChainSafe/gossamer/runtime"
	log "github.com/ChainSafe/log15"
)

// Session contains the VRF keys for the validator, as well as BABE configuation data
type Session struct {
	keypair        *sr25519.Keypair
	rt             *runtime.Runtime
	config         *BabeConfiguration
	authorityIndex uint64
	authorityData  []AuthorityData
	epochThreshold *big.Int // validator threshold for this epoch
	txQueue        *tx.PriorityQueue
	slotToProof    map[uint64]*VrfOutputAndProof // for slots where we are a producer, store the vrf output (bytes 0-32) + proof (bytes 32-96)
	newBlocks      chan<- types.Block            // send blocks to core service
}

type SessionConfig struct {
	Keypair   *sr25519.Keypair
	Runtime   *runtime.Runtime
	NewBlocks chan<- types.Block
}

// NewSession returns a new Babe session using the provided VRF keys and runtime
func NewSession(cfg *SessionConfig) (*Session, error) {
	if cfg.Keypair == nil {
		return nil, errors.New("cannot start BABE session; no keypair provided")
	}

	babeSession := &Session{
		keypair:     cfg.Keypair,
		rt:          cfg.Runtime,
		txQueue:     new(tx.PriorityQueue),
		slotToProof: make(map[uint64]*VrfOutputAndProof),
		newBlocks:   cfg.NewBlocks,
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
		b.slotToProof[i], err = b.runLottery(i)
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
			Header: &types.BlockHeader{
				Number: big.NewInt(0),
			},
		}
		time.Sleep(time.Millisecond * time.Duration(b.config.SlotDuration))
	}
}

// runLottery runs the lottery for a specific slot number
// returns an encoded VrfOutput and VrfProof if validator is authorized to produce a block for that slot, nil otherwise
// output = return[0:32]; proof = return[32:96]
func (b *Session) runLottery(slot uint64) (*VrfOutputAndProof, error) {
	slotBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(slotBytes, slot)
	vrfInput := append(slotBytes, b.config.Randomness)

	output, proof, err := b.vrfSign(vrfInput)
	if err != nil {
		return nil, err
	}

	outputInt := big.NewInt(0).SetBytes(output[:])
	if b.epochThreshold == nil {
		err = b.setEpochThreshold()
		if err != nil {
			return nil, err
		}
	}

	if outputInt.Cmp(b.epochThreshold) > 0 {
		outbytes := [sr25519.VrfOutputLength]byte{}
		copy(outbytes[:], output)
		proofbytes := [sr25519.VrfProofLength]byte{}
		copy(proofbytes[:], proof)
		return &VrfOutputAndProof{
			output: outbytes,
			proof:  proofbytes,
		}, nil
	}

	return nil, nil
}

func (b *Session) vrfSign(input []byte) (out []byte, proof []byte, err error) {
	return b.keypair.VrfSign(input)
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

// construct a block for this slot with the given parent
func (b *Session) buildBlock(parent *types.BlockHeader, slot Slot) (*types.Block, error) {
	log.Debug("build-block", "parent", parent, "slot", slot)

	// create pre-digest
	preDigest, err := b.buildBlockPreDigest(slot)
	if err != nil {
		return nil, err
	}

	// initialize block
	encodedHeader, err := scale.Encode(parent)
	if err != nil {
		return nil, err
	}
	err = b.initializeBlock(encodedHeader)
	if err != nil {
		return nil, err
	}

	// add block inherents
	err = b.buildBlockInherents(slot)
	if err != nil {
		return nil, err
	}

	// add block extrinsics
	included, err := b.buildBlockExtrinsics(slot)
	if err != nil {
		return nil, err
	}

	// finalize block
	log.Trace("build_block finalize block")
	block, err := b.finalizeBlock()
	if err != nil {
		b.addToQueue(included)
		return nil, err
	}

	block.Header.Number.Add(parent.Number, big.NewInt(1))

	// add BABE header to digest
	block.Header.Digest = append(block.Header.Digest, preDigest.Encode())

	// create seal and add to digest
	seal, err := b.buildBlockSeal(block.Header)
	if err != nil {
		return nil, err
	}

	block.Header.Digest = append(block.Header.Digest, seal.Encode())

	return block, nil
}

// buildBlockSeal creates the seal for the block header.
// the seal consists of the ConsensusEngineId and a signature of the encoded block header.
func (b *Session) buildBlockSeal(header *types.BlockHeader) (*types.SealDigest, error) {
	encHeader, err := header.Encode()
	if err != nil {
		return nil, err
	}

	sig, err := b.keypair.Sign(encHeader)
	if err != nil {
		return nil, err
	}

	return &types.SealDigest{
		ConsensusEngineID: types.BabeEngineID,
		Data:              sig,
	}, nil
}

// buildBlockPreDigest creates the pre-digest for the slot.
// the pre-digest consists of the ConsensusEngineId and the encoded BABE header for the slot.
func (b *Session) buildBlockPreDigest(slot Slot) (*types.PreRuntimeDigest, error) {
	babeHeader, err := b.buildBlockBabeHeader(slot)
	if err != nil {
		return nil, err
	}
	encBabeHeader := babeHeader.Encode()

	return &types.PreRuntimeDigest{
		ConsensusEngineID: types.BabeEngineID,
		Data:              encBabeHeader,
	}, nil
}

// buildBlockBabeHeader creates the BABE header for the slot.
// the BABE header includes the proof of authorship right for this slot.
func (b *Session) buildBlockBabeHeader(slot Slot) (*BabeHeader, error) {
	if b.slotToProof[slot.number] == nil {
		return nil, errors.New("not authorized to produce block")
	}
	outAndProof := b.slotToProof[slot.number]
	return &BabeHeader{
		VrfOutput:          outAndProof.output,
		VrfProof:           outAndProof.proof,
		BlockProducerIndex: b.authorityIndex,
		SlotNumber:         slot.number,
	}, nil
}

// buildBlockExtrinsics applies extrinsics to the block. it returns an array of included extrinsics.
// for each extrinsic in queue, add it to the block, until the slot ends or the block is full.
// if any extrinsic fails, it returns an empty array and an error.
func (b *Session) buildBlockExtrinsics(slot Slot) ([]*tx.ValidTransaction, error) {
	extrinsic := b.nextReadyExtrinsic()
	included := []*tx.ValidTransaction{}

	// TODO: check when block is full
	for !hasSlotEnded(slot) && extrinsic != nil {
		log.Trace("build_block", "applying extrinsic", extrinsic)
		ret, err := b.applyExtrinsic(*extrinsic)
		if err != nil {
			return nil, err
		}

		// if ret == 0x0001, there is a dispatch error; if ret == 0x01, there is an apply error
		if ret[0] == 1 || bytes.Equal(ret[:2], []byte{0, 1}) {
			// TODO: specific error code checking
			log.Error("build_block apply extrinsic", "error", ret, "extrinsic", extrinsic)

			// remove invalid extrinsic from queue
			b.txQueue.Pop()

			// re-add previously popped extrinsics back to queue
			b.addToQueue(included)

			return nil, errors.New("could not apply extrinsic")
		} else {
			log.Trace("build_block applied extrinsic", "extrinsic", extrinsic)
		}

		// keep track of included transactions; re-add them to queue later if block building fails
		t := b.txQueue.Pop()
		included = append(included, t)
		extrinsic = b.nextReadyExtrinsic()
	}

	return included, nil
}

// buildBlockInherents applies the inherents for a block
func (b *Session) buildBlockInherents(slot Slot) error {
	// Setup inherents: add timstap0 and babeslot
	idata := NewInherentsData()
	err := idata.SetInt64Inherent(Timstap0, uint64(time.Now().Unix()))
	if err != nil {
		return err
	}

	err = idata.SetInt64Inherent(Babeslot, slot.number)
	if err != nil {
		return err
	}

	ienc, err := idata.Encode()
	if err != nil {
		return err
	}

	// Call BlockBuilder_inherent_extrinsics
	_, err = b.inherentExtrinsics(ienc)
	if err != nil {
		return err
	}

	return nil
}

func (b *Session) addToQueue(txs []*tx.ValidTransaction) {
	for _, t := range txs {
		b.txQueue.Insert(t)
	}
}

// nextReadyExtrinsic peeks from the transaction queue. it does not remove any transactions from the queue
func (b *Session) nextReadyExtrinsic() *types.Extrinsic {
	transaction := b.txQueue.Peek()
	if transaction == nil {
		return nil
	}
	return transaction.Extrinsic
}

func hasSlotEnded(slot Slot) bool {
	return slot.start+slot.duration < uint64(time.Now().Unix())
}
