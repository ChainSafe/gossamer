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
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	ethmetrics "github.com/ethereum/go-ethereum/metrics"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/scale"
	"github.com/ChainSafe/gossamer/lib/transaction"
)

const (
	buildBlockTimer  = "gossamer/proposer/block/constructed"
	buildBlockErrors = "gossamer/proposer/block/constructed/errors"
)

// construct a block for this slot with the given parent
func (b *Service) buildBlock(parent *types.Header, slot Slot) (*types.Block, error) {
	builder, err := NewBlockBuilder(
		b.rt,
		b.keypair,
		b.transactionState,
		b.blockState,
		b.slotToProof,
		b.epochData.authorityIndex,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create block builder: %w", err)
	}

	startBuilt := time.Now()
	block, err := builder.buildBlock(parent, slot)

	// is necessary to enable ethmetrics to be possible register values
	ethmetrics.Enabled = true //nolint

	if err != nil {
		builderErrors := ethmetrics.GetOrRegisterCounter(buildBlockErrors, nil)
		builderErrors.Inc(1)

		return nil, err
	}

	timerMetrics := ethmetrics.GetOrRegisterTimer(buildBlockTimer, nil)
	timerMetrics.Update(time.Since(startBuilt))
	return block, err
}

// nolint
type BlockBuilder struct {
	rt                    runtime.Instance
	keypair               *sr25519.Keypair
	transactionState      TransactionState
	blockState            BlockState
	slotToProof           map[uint64]*VrfOutputAndProof
	currentAuthorityIndex uint32
}

// nolint
func NewBlockBuilder(rt runtime.Instance, kp *sr25519.Keypair, ts TransactionState, bs BlockState, sp map[uint64]*VrfOutputAndProof, authidx uint32) (*BlockBuilder, error) {
	if rt == nil {
		return nil, errors.New("cannot create block builder; runtime instance is nil")
	}
	if ts == nil {
		return nil, errors.New("cannot create block builder; transaction state is nil")
	}
	if bs == nil {
		return nil, errors.New("cannot create block builder; block state is nil")
	}
	if sp == nil {
		return nil, errors.New("cannot create block builder; slot to proff is nil")
	}

	bb := &BlockBuilder{
		rt:                    rt,
		keypair:               kp,
		transactionState:      ts,
		blockState:            bs,
		slotToProof:           sp,
		currentAuthorityIndex: authidx,
	}

	return bb, nil
}

func (b *BlockBuilder) buildBlock(parent *types.Header, slot Slot) (*types.Block, error) {
	logger.Trace("build block", "parent", parent, "slot", slot)

	// create pre-digest
	preDigest, err := b.buildBlockPreDigest(slot)
	if err != nil {
		return nil, err
	}

	logger.Trace("built pre-digest")

	// create new block header
	number := big.NewInt(0).Add(parent.Number, big.NewInt(1))
	header, err := types.NewHeader(parent.Hash(), common.Hash{}, common.Hash{}, number, types.NewDigest(preDigest))
	if err != nil {
		return nil, err
	}

	// initialise block header
	err = b.rt.InitializeBlock(header)
	if err != nil {
		return nil, err
	}

	logger.Trace("initialised block")

	// add block inherents
	inherents, err := b.buildBlockInherents(slot)
	if err != nil {
		return nil, fmt.Errorf("cannot build inherents: %s", err)
	}

	logger.Trace("built block inherents", "encoded inherents", inherents)

	// add block extrinsics
	included := b.buildBlockExtrinsics(slot)

	logger.Trace("built block extrinsics")

	// finalise block
	header, err = b.rt.FinalizeBlock()
	if err != nil {
		b.addToQueue(included)
		return nil, fmt.Errorf("cannot finalise block: %s", err)
	}

	logger.Trace("finalised block")

	// create seal and add to digest
	seal, err := b.buildBlockSeal(header)
	if err != nil {
		return nil, err
	}

	header.Digest = append(header.Digest, seal)

	logger.Trace("built block seal")

	body, err := ExtrinsicsToBody(inherents, included)
	if err != nil {
		return nil, err
	}

	block := &types.Block{
		Header: header,
		Body:   body,
	}

	return block, nil
}

// buildBlockSeal creates the seal for the block header.
// the seal consists of the ConsensusEngineID and a signature of the encoded block header.
func (b *BlockBuilder) buildBlockSeal(header *types.Header) (*types.SealDigest, error) {
	encHeader, err := header.Encode()
	if err != nil {
		return nil, err
	}

	hash, err := common.Blake2bHash(encHeader)
	if err != nil {
		return nil, err
	}

	sig, err := b.keypair.Sign(hash[:])
	if err != nil {
		return nil, err
	}

	return &types.SealDigest{
		ConsensusEngineID: types.BabeEngineID,
		Data:              sig,
	}, nil
}

// buildBlockPreDigest creates the pre-digest for the slot.
// the pre-digest consists of the ConsensusEngineID and the encoded BABE header for the slot.
func (b *BlockBuilder) buildBlockPreDigest(slot Slot) (*types.PreRuntimeDigest, error) {
	babeHeader, err := b.buildBlockBABEPrimaryPreDigest(slot)
	if err != nil {
		return nil, err
	}

	encBABEPrimaryPreDigest := babeHeader.Encode()

	return &types.PreRuntimeDigest{
		ConsensusEngineID: types.BabeEngineID,
		Data:              encBABEPrimaryPreDigest,
	}, nil
}

// buildBlockBABEPrimaryPreDigest creates the BABE header for the slot.
// the BABE header includes the proof of authorship right for this slot.
func (b *BlockBuilder) buildBlockBABEPrimaryPreDigest(slot Slot) (*types.BabePrimaryPreDigest, error) {
	if b.slotToProof[slot.number] == nil {
		return nil, ErrNotAuthorized
	}

	outAndProof := b.slotToProof[slot.number]
	return types.NewBabePrimaryPreDigest(
		b.currentAuthorityIndex,
		slot.number,
		outAndProof.output,
		outAndProof.proof,
	), nil
}

// buildBlockExtrinsics applies extrinsics to the block. it returns an array of included extrinsics.
// for each extrinsic in queue, add it to the block, until the slot ends or the block is full.
// if any extrinsic fails, it returns an empty array and an error.
func (b *BlockBuilder) buildBlockExtrinsics(slot Slot) []*transaction.ValidTransaction {
	var included []*transaction.ValidTransaction

	for !hasSlotEnded(slot) {
		txn := b.transactionState.Pop()
		// Transaction queue is empty.
		if txn == nil {
			return included
		}

		// Move to next extrinsic.
		if txn.Extrinsic == nil {
			continue
		}

		extrinsic := txn.Extrinsic
		logger.Trace("build block", "applying extrinsic", extrinsic)

		ret, err := b.rt.ApplyExtrinsic(extrinsic)
		if err != nil {
			logger.Warn("failed to apply extrinsic", "error", err, "extrinsic", extrinsic)
			continue
		}

		err = determineErr(ret)
		if err != nil {
			logger.Warn("failed to apply extrinsic", "error", err, "extrinsic", extrinsic)

			// Failure of the module call dispatching doesn't invalidate the extrinsic.
			// It is included in the block.
			if _, ok := err.(*DispatchOutcomeError); !ok {
				continue
			}
		}

		logger.Debug("build block applied extrinsic", "extrinsic", extrinsic)
		included = append(included, txn)
	}

	return included
}

func (b *BlockBuilder) buildBlockInherents(slot Slot) ([][]byte, error) {
	// Setup inherents: add timstap0
	idata := types.NewInherentsData()
	err := idata.SetInt64Inherent(types.Timstap0, uint64(time.Now().Unix()))
	if err != nil {
		return nil, err
	}

	// add babeslot
	err = idata.SetInt64Inherent(types.Babeslot, slot.number)
	if err != nil {
		return nil, err
	}

	ienc, err := idata.Encode()
	if err != nil {
		return nil, err
	}

	// Call BlockBuilder_inherent_extrinsics which returns the inherents as extrinsics
	inherentExts, err := b.rt.InherentExtrinsics(ienc)
	if err != nil {
		return nil, err
	}

	// decode inherent extrinsics
	exts, err := scale.Decode(inherentExts, [][]byte{})
	if err != nil {
		return nil, err
	}

	// apply each inherent extrinsic
	for _, ext := range exts.([][]byte) {
		in, err := scale.Encode(ext)
		if err != nil {
			return nil, err
		}

		ret, err := b.rt.ApplyExtrinsic(in)
		if err != nil {
			return nil, err
		}

		if !bytes.Equal(ret, []byte{0, 0}) {
			errTxt := determineErr(ret)
			return nil, fmt.Errorf("error applying inherent: %s", errTxt)
		}
	}

	return exts.([][]byte), nil
}

func (b *BlockBuilder) addToQueue(txs []*transaction.ValidTransaction) {
	for _, t := range txs {
		hash, err := b.transactionState.Push(t)
		if err != nil {
			logger.Trace("Failed to add transaction to queue", "error", err)
		} else {
			logger.Trace("Added transaction to queue", "hash", hash)
		}
	}
}

func hasSlotEnded(slot Slot) bool {
	slotEnd := slot.start.Add(slot.duration * 2 / 3) // reserve last 1/3 of slot for block finalisation
	return time.Since(slotEnd) >= 0
}

// ExtrinsicsToBody returns scale encoded block body which contains inherent and extrinsic.
func ExtrinsicsToBody(inherents [][]byte, txs []*transaction.ValidTransaction) (*types.Body, error) {
	extrinsics := types.BytesArrayToExtrinsics(inherents)

	for _, tx := range txs {
		decExt, err := scale.Decode(tx.Extrinsic, []byte{})
		if err != nil {
			return nil, err
		}
		extrinsics = append(extrinsics, decExt.([]byte))
	}

	return types.NewBodyFromExtrinsics(extrinsics)
}
