// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package babe

import (
	"bytes"
	"errors"
	"fmt"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/transaction"
	"github.com/ChainSafe/gossamer/pkg/scale"
	ethmetrics "github.com/ethereum/go-ethereum/metrics"
)

const (
	buildBlockTimer  = "gossamer/proposer/block/constructed"
	buildBlockErrors = "gossamer/proposer/block/constructed/errors"
)

// construct a block for this slot with the given parent
func (b *Service) buildBlock(parent *types.Header, slot Slot, rt Runtime,
	authorityIndex uint32, preRuntimeDigest *types.PreRuntimeDigest) (*types.Block, error) {
	builder := NewBlockBuilder(
		b.keypair,
		b.transactionState,
		b.blockState,
		authorityIndex,
		preRuntimeDigest,
	)

	// is necessary to enable ethmetrics to be possible register values
	ethmetrics.Enabled = true

	start := time.Now()
	block, err := builder.buildBlock(parent, slot, rt)
	if err != nil {
		builderErrors := ethmetrics.GetOrRegisterCounter(buildBlockErrors, nil)
		builderErrors.Inc(1)
		return nil, err
	}

	timerMetrics := ethmetrics.GetOrRegisterTimer(buildBlockTimer, nil)
	timerMetrics.Update(time.Since(start))
	return block, nil
}

// BlockBuilder builds blocks.
type BlockBuilder struct {
	keypair               *sr25519.Keypair
	transactionState      TransactionState
	blockState            BlockState
	currentAuthorityIndex uint32
	preRuntimeDigest      *types.PreRuntimeDigest
}

// NewBlockBuilder creates a new block builder.
func NewBlockBuilder(
	kp *sr25519.Keypair,
	ts TransactionState,
	bs BlockState,
	authidx uint32,
	preRuntimeDigest *types.PreRuntimeDigest,
) *BlockBuilder {
	return &BlockBuilder{
		keypair:               kp,
		transactionState:      ts,
		blockState:            bs,
		currentAuthorityIndex: authidx,
		preRuntimeDigest:      preRuntimeDigest,
	}
}

func (b *BlockBuilder) buildBlock(parent *types.Header, slot Slot, rt Runtime) (*types.Block, error) {
	logger.Tracef("build block with parent %s and slot: %s", parent, slot)

	// create new block header
	number := parent.Number + 1
	digest := types.NewDigest()
	err := digest.Add(*b.preRuntimeDigest)
	if err != nil {
		return nil, err
	}
	header := types.NewHeader(parent.Hash(), common.Hash{}, common.Hash{}, number, digest)

	// initialise block header
	err = rt.InitializeBlock(header)
	if err != nil {
		return nil, err
	}

	logger.Trace("initialised block")

	// add block inherents
	inherents, err := buildBlockInherents(slot, rt, parent)
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

// buildBlockExtrinsics applies extrinsics to the block. it returns an array of included extrinsics.
// for each extrinsic in queue, add it to the block, until the slot ends or the block is full.
// if any extrinsic fails, it returns an empty array and an error.
func (b *BlockBuilder) buildBlockExtrinsics(slot Slot, rt ExtrinsicHandler) []*transaction.ValidTransaction {
	var included []*transaction.ValidTransaction

	slotEnd := slot.start.Add(slot.duration * 2 / 3) // reserve last 1/3 of slot for block finalisation
	timeout := time.Until(slotEnd)
	slotTimer := time.NewTimer(timeout)

	for {
		txn := b.transactionState.PopWithTimer(slotTimer.C)
		slotTimerExpired := txn == nil
		if slotTimerExpired {
			break
		}

		extrinsic := txn.Extrinsic
		logger.Tracef("build block, applying extrinsic %s", extrinsic)

		ret, err := rt.ApplyExtrinsic(extrinsic)
		if err != nil {
			logger.Warnf("determining apply extrinsic call error: %s", err)
			continue
		}

		err = determineErr(ret)
		if err != nil {
			logger.Warnf("error when applying extrinsic %s: %s", extrinsic, err)

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

func buildBlockInherents(slot Slot, rt ExtrinsicHandler, parent *types.Header) ([][]byte, error) {
	// Setup inherents: add timstap0
	idata := types.NewInherentData()
	err := idata.SetInherent(types.Timstap0, uint64(slot.start.UnixMilli()))
	if err != nil {
		return nil, err
	}

	// add babeslot
	err = idata.SetInherent(types.Babeslot, slot.number)
	if err != nil {
		return nil, err
	}

	// parachainInherent := inherents.ParachainInherentData{
	// 	ParentHeader: *parent,
	// }

	// add parachn0 and newheads
	// for now we can use "empty" values, as we require parachain-specific
	// logic to actually provide the data.

	// if err = idata.SetInherent(types.Parachn0, parachainInherent); err != nil {
	// 	return nil, fmt.Errorf("setting inherent %q: %w", types.Parachn0, err)
	// }

	// if err = idata.SetInherent(types.Newheads, []byte{0}); err != nil {
	// 	return nil, fmt.Errorf("setting inherent %q: %w", types.Newheads, err)
	// }

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
