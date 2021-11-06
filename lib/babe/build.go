// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package babe

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/transaction"
	"github.com/ChainSafe/gossamer/pkg/scale"
	ethmetrics "github.com/ethereum/go-ethereum/metrics"
)

const (
	buildBlockTimer  = "gossamer/proposer/block/constructed"
	buildBlockErrors = "gossamer/proposer/block/constructed/errors"
)

// construct a block for this slot with the given parent
func (b *Service) buildBlock(parent *types.Header, slot Slot, rt runtime.Instance) (*types.Block, error) {
	builder, err := NewBlockBuilder(
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
	block, err := builder.buildBlock(parent, slot, rt)

	// is necessary to enable ethmetrics to be possible register values
	ethmetrics.Enabled = true

	if err != nil {
		builderErrors := ethmetrics.GetOrRegisterCounter(buildBlockErrors, nil)
		builderErrors.Inc(1)

		return nil, err
	}

	timerMetrics := ethmetrics.GetOrRegisterTimer(buildBlockTimer, nil)
	timerMetrics.Update(time.Since(startBuilt))
	return block, nil
}

// BlockBuilder builds blocks.
type BlockBuilder struct {
	keypair               *sr25519.Keypair
	transactionState      TransactionState
	blockState            BlockState
	slotToProof           map[uint64]*VrfOutputAndProof
	currentAuthorityIndex uint32
}

// NewBlockBuilder creates a new block builder.
func NewBlockBuilder(kp *sr25519.Keypair, ts TransactionState, bs BlockState, sp map[uint64]*VrfOutputAndProof, authidx uint32) (*BlockBuilder, error) {
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
		keypair:               kp,
		transactionState:      ts,
		blockState:            bs,
		slotToProof:           sp,
		currentAuthorityIndex: authidx,
	}

	return bb, nil
}

func (b *BlockBuilder) buildBlock(parent *types.Header, slot Slot, rt runtime.Instance) (*types.Block, error) {
	logger.Tracef("build block with parent %s and slot: %s", parent, slot)

	// create pre-digest
	preDigest, err := b.buildBlockPreDigest(slot)
	if err != nil {
		return nil, err
	}

	logger.Trace("built pre-digest")

	// create new block header
	number := big.NewInt(0).Add(parent.Number, big.NewInt(1))
	digest := types.NewDigest()
	err = digest.Add(*preDigest)
	if err != nil {
		return nil, err
	}
	header, err := types.NewHeader(parent.Hash(), common.Hash{}, common.Hash{}, number, digest)
	if err != nil {
		return nil, err
	}

	// initialise block header
	err = rt.InitializeBlock(header)
	if err != nil {
		return nil, err
	}

	logger.Trace("initialised block")

	// add block inherents
	inherents, err := b.buildBlockInherents(slot, rt)
	if err != nil {
		return nil, fmt.Errorf("cannot build inherents: %s", err)
	}

	logger.Tracef("built block encoded inherents: %v", inherents)

	// add block extrinsics
	included := b.buildBlockExtrinsics(slot, rt)

	logger.Trace("built block extrinsics")

	// finalise block
	header, err = rt.FinalizeBlock()
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

	err = header.Digest.Add(*seal)
	if err != nil {
		return nil, err
	}

	logger.Trace("built block seal")

	body, err := extrinsicsToBody(inherents, included)
	if err != nil {
		return nil, err
	}

	block := &types.Block{
		Header: *header,
		Body:   body,
	}

	return block, nil
}

// buildBlockSeal creates the seal for the block header.
// the seal consists of the ConsensusEngineID and a signature of the encoded block header.
func (b *BlockBuilder) buildBlockSeal(header *types.Header) (*types.SealDigest, error) {
	encHeader, err := scale.Marshal(*header)
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
	babeHeader := types.NewBabeDigest()
	data, err := b.buildBlockBABEPrimaryPreDigest(slot)
	if err != nil {
		return nil, err
	}

	err = babeHeader.Set(*data)
	if err != nil {
		return nil, err
	}

	encBABEPrimaryPreDigest, err := scale.Marshal(babeHeader)
	if err != nil {
		return nil, err
	}

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
func (b *BlockBuilder) buildBlockExtrinsics(slot Slot, rt runtime.Instance) []*transaction.ValidTransaction {
	var included []*transaction.ValidTransaction

	for !hasSlotEnded(slot) {
		txn := b.transactionState.Pop()
		// Transaction queue is empty.
		if txn == nil {
			continue
		}

		extrinsic := txn.Extrinsic
		logger.Tracef("build block, applying extrinsic %s", extrinsic)

		ret, err := rt.ApplyExtrinsic(extrinsic)
		if err != nil {
			logger.Warnf("failed to apply extrinsic %s: %s", extrinsic, err)
			continue
		}

		err = determineErr(ret)
		if err != nil {
			logger.Warnf("failed to apply extrinsic %s: %s", extrinsic, err)

			// Failure of the module call dispatching doesn't invalidate the extrinsic.
			// It is included in the block.
			if _, ok := err.(*DispatchOutcomeError); !ok {
				continue
			}

			// don't drop transactions that may be valid in a later block ie.
			// run out of gas for this block or have a nonce that may be valid in a later block
			var e *TransactionValidityError
			if !errors.As(err, &e) {
				continue
			}

			if errors.Is(e.msg, errExhaustsResources) || errors.Is(e.msg, errInvalidTransaction) {
				hash, err := b.transactionState.Push(txn)
				if err != nil {
					logger.Debugf("failed to re-add transaction with hash %s to queue: %s", hash, err)
				}
			}
		}

		logger.Debugf("build block applied extrinsic %s", extrinsic)
		included = append(included, txn)
	}

	return included
}

func (b *BlockBuilder) buildBlockInherents(slot Slot, rt runtime.Instance) ([][]byte, error) {
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
	inherentExts, err := rt.InherentExtrinsics(ienc)
	if err != nil {
		return nil, err
	}

	// decode inherent extrinsics
	var exts [][]byte
	err = scale.Unmarshal(inherentExts, &exts)
	if err != nil {
		return nil, err
	}

	// apply each inherent extrinsic
	for _, ext := range exts {
		in, err := scale.Marshal(ext)
		if err != nil {
			return nil, err
		}

		ret, err := rt.ApplyExtrinsic(in)
		if err != nil {
			return nil, err
		}

		if !bytes.Equal(ret, []byte{0, 0}) {
			errTxt := determineErr(ret)
			return nil, fmt.Errorf("error applying inherent: %s", errTxt)
		}
	}

	return exts, nil
}

func (b *BlockBuilder) addToQueue(txs []*transaction.ValidTransaction) {
	for _, t := range txs {
		hash, err := b.transactionState.Push(t)
		if err != nil {
			logger.Tracef("Failed to add transaction to queue: %s", err)
		} else {
			logger.Tracef("Added transaction with hash %s to queue", hash)
		}
	}
}

func hasSlotEnded(slot Slot) bool {
	slotEnd := slot.start.Add(slot.duration * 2 / 3) // reserve last 1/3 of slot for block finalisation
	return time.Since(slotEnd) >= 0
}

func extrinsicsToBody(inherents [][]byte, txs []*transaction.ValidTransaction) (types.Body, error) {
	extrinsics := types.BytesArrayToExtrinsics(inherents)

	for _, tx := range txs {
		var decExt []byte
		err := scale.Unmarshal(tx.Extrinsic, &decExt)
		if err != nil {
			return nil, err
		}
		extrinsics = append(extrinsics, decExt)
	}

	return types.Body(extrinsics), nil
}
