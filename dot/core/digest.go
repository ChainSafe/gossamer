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
	"context"
	"errors"
	"math/big"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/scale"
)

var maxUint64 = uint64(2^64) - 1

// DigestHandler is used to handle consensus messages and relevant authority updates to BABE and GRANDPA
type DigestHandler struct {
	ctx    context.Context
	cancel context.CancelFunc

	// interfaces
	blockState          BlockState
	epochState          EpochState
	grandpa             FinalityGadget
	babe                BlockProducer
	verifier            Verifier
	isFinalityAuthority bool
	isBlockProducer     bool

	// block notification channels
	imported    chan *types.Block
	importedID  byte
	finalized   chan *types.Header
	finalizedID byte

	// GRANDPA changes
	grandpaScheduledChange *grandpaChange
	grandpaForcedChange    *grandpaChange
	grandpaPause           *pause
	grandpaResume          *resume
	grandpaAuths           []*types.Authority // saved in case of pause
}

type grandpaChange struct {
	auths   []*types.Authority
	atBlock *big.Int
}

type pause struct {
	atBlock *big.Int
}

type resume struct {
	atBlock *big.Int
}

// NewDigestHandler returns a new DigestHandler
func NewDigestHandler(blockState BlockState, epochState EpochState, babe BlockProducer, grandpa FinalityGadget, verifier Verifier) (*DigestHandler, error) {
	imported := make(chan *types.Block, 16)
	finalized := make(chan *types.Header, 16)
	iid, err := blockState.RegisterImportedChannel(imported)
	if err != nil {
		return nil, err
	}

	fid, err := blockState.RegisterFinalizedChannel(finalized)
	if err != nil {
		return nil, err
	}

	isFinalityAuthority := grandpa != nil
	isBlockProducer := babe != nil

	ctx, cancel := context.WithCancel(context.Background())

	return &DigestHandler{
		ctx:                 ctx,
		cancel:              cancel,
		blockState:          blockState,
		epochState:          epochState,
		grandpa:             grandpa,
		babe:                babe,
		verifier:            verifier,
		isFinalityAuthority: isFinalityAuthority,
		isBlockProducer:     isBlockProducer,
		imported:            imported,
		importedID:          iid,
		finalized:           finalized,
		finalizedID:         fid,
	}, nil
}

// Start starts the DigestHandler
func (h *DigestHandler) Start() {
	go h.handleBlockImport(h.ctx)
	go h.handleBlockFinalization(h.ctx)
}

// Stop stops the DigestHandler
func (h *DigestHandler) Stop() {
	h.cancel()
	h.blockState.UnregisterImportedChannel(h.importedID)
	h.blockState.UnregisterFinalizedChannel(h.finalizedID)
	close(h.imported)
	close(h.finalized)
}

// SetFinalityGadget sets the digest handler's grandpa instance
func (h *DigestHandler) SetFinalityGadget(grandpa FinalityGadget) {
	h.grandpa = grandpa
}

// NextGrandpaAuthorityChange returns the block number of the next upcoming grandpa authorities change.
// It returns 0 if no change is scheduled.
func (h *DigestHandler) NextGrandpaAuthorityChange() uint64 {
	next := maxUint64

	if h.grandpaScheduledChange != nil {
		next = h.grandpaScheduledChange.atBlock.Uint64()
	}

	if h.grandpaForcedChange != nil && h.grandpaForcedChange.atBlock.Uint64() < next {
		next = h.grandpaForcedChange.atBlock.Uint64()
	}

	if h.grandpaPause != nil && h.grandpaPause.atBlock.Uint64() < next {
		next = h.grandpaPause.atBlock.Uint64()
	}

	if h.grandpaResume != nil && h.grandpaResume.atBlock.Uint64() < next {
		next = h.grandpaResume.atBlock.Uint64()
	}

	return next
}

// HandleConsensusDigest is the function used by the syncer to handle a consensus digest
func (h *DigestHandler) HandleConsensusDigest(d *types.ConsensusDigest, header *types.Header) error {
	t := d.DataType()

	if d.ConsensusEngineID == types.GrandpaEngineID {
		switch t {
		case types.GrandpaScheduledChangeType:
			return h.handleScheduledChange(d)
		case types.GrandpaForcedChangeType:
			return h.handleForcedChange(d)
		case types.GrandpaOnDisabledType:
			return h.handleGrandpaOnDisabled(d, header)
		case types.GrandpaPauseType:
			return h.handlePause(d)
		case types.GrandpaResumeType:
			return h.handleResume(d)
		default:
			return errors.New("invalid consensus digest data")
		}
	}

	if d.ConsensusEngineID == types.BabeEngineID {
		switch t {
		case types.NextEpochDataType:
			return h.handleNextEpochData(d, header)
		case types.BABEOnDisabledType:
			return h.handleBABEOnDisabled(d, header)
		case types.NextConfigDataType:
			return h.handleNextConfigData(d, header)
		default:
			return errors.New("invalid consensus digest data")
		}
	}

	return errors.New("unknown consensus engine ID")
}

func (h *DigestHandler) handleBlockImport(ctx context.Context) {
	for {
		select {
		case block := <-h.imported:
			if block == nil || block.Header == nil {
				continue
			}

			if h.isFinalityAuthority {
				h.handleGrandpaChangesOnImport(block.Header.Number)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (h *DigestHandler) handleBlockFinalization(ctx context.Context) {
	for {
		select {
		case header := <-h.finalized:
			if header == nil {
				continue
			}

			if h.isFinalityAuthority {
				h.handleGrandpaChangesOnFinalization(header.Number)
			}

			// TODO: check if there's a NextEpochData or NextConfigData digest, if there is,
			// make sure it matches what's in the EpochState for the upcoming epoch
		case <-ctx.Done():
			return
		}
	}
}

func (h *DigestHandler) handleGrandpaChangesOnImport(num *big.Int) {
	resume := h.grandpaResume
	if resume != nil && num.Cmp(resume.atBlock) == 0 {
		h.grandpa.UpdateAuthorities(h.grandpaAuths)
		h.grandpaResume = nil
	}

	fc := h.grandpaForcedChange
	if fc != nil && num.Cmp(fc.atBlock) == 0 {
		h.grandpa.UpdateAuthorities(fc.auths)
		h.grandpaForcedChange = nil
	}
}

func (h *DigestHandler) handleGrandpaChangesOnFinalization(num *big.Int) {
	pause := h.grandpaPause
	if pause != nil && num.Cmp(pause.atBlock) == 0 {
		// save authority data for Resume
		h.grandpaAuths = h.grandpa.Authorities()
		h.grandpa.UpdateAuthorities([]*types.Authority{})
		h.grandpaPause = nil
	}

	sc := h.grandpaScheduledChange
	if sc != nil && num.Cmp(sc.atBlock) == 0 {
		h.grandpa.UpdateAuthorities(sc.auths)
		h.grandpaScheduledChange = nil
	}

	// if blocks get finalized before forced change takes place, disregard it
	h.grandpaForcedChange = nil
}

func (h *DigestHandler) handleScheduledChange(d *types.ConsensusDigest) error {
	curr, err := h.blockState.BestBlockHeader()
	if err != nil {
		return err
	}

	if d.ConsensusEngineID == types.GrandpaEngineID {
		if h.grandpaScheduledChange != nil {
			return nil
		}

		sc := &types.GrandpaScheduledChange{}
		dec, err := scale.Decode(d.Data[1:], sc)
		if err != nil {
			return err
		}
		sc = dec.(*types.GrandpaScheduledChange)

		logger.Debug("handling GrandpaScheduledChange", "data", sc)

		if h.grandpa == nil {
			// this should never happen
			return nil
		}

		c, err := newGrandpaChange(sc.Auths, sc.Delay, curr.Number)
		if err != nil {
			return err
		}

		h.grandpaScheduledChange = c
	}

	return nil
}

func (h *DigestHandler) handleForcedChange(d *types.ConsensusDigest) error {
	curr, err := h.blockState.BestBlockHeader()
	if err != nil {
		return err
	}

	if d.ConsensusEngineID == types.GrandpaEngineID {
		if h.grandpaForcedChange != nil {
			return errors.New("already have forced change scheduled")
		}

		fc := &types.GrandpaForcedChange{}
		dec, err := scale.Decode(d.Data[1:], fc)
		if err != nil {
			return err
		}
		fc = dec.(*types.GrandpaForcedChange)

		c, err := newGrandpaChange(fc.Auths, fc.Delay, curr.Number)
		if err != nil {
			return err
		}

		h.grandpaForcedChange = c
	}

	return nil
}

func (h *DigestHandler) handleGrandpaOnDisabled(d *types.ConsensusDigest, _ *types.Header) error {
	od := &types.GrandpaOnDisabled{}
	dec, err := scale.Decode(d.Data[1:], od)
	if err != nil {
		return err
	}
	od = dec.(*types.GrandpaOnDisabled)

	logger.Debug("handling GrandpaOnDisabled", "data", od)

	if h.grandpa == nil {
		// this should never happen
		return nil
	}

	curr := h.grandpa.Authorities()
	next := []*types.Authority{}

	for _, auth := range curr {
		if auth.Weight != od.ID {
			next = append(next, auth)
		}
	}

	// TODO: this needs to be updated not to remove the authority from the list,
	// but to flag them as disabled. thus, if we are disabled, we should stop voting.
	// if we receive vote or finalization messages, we should ignore anything signed by the
	// disabled authority
	h.grandpa.UpdateAuthorities(next)
	return nil
}

func (h *DigestHandler) handlePause(d *types.ConsensusDigest) error {
	curr, err := h.blockState.BestBlockHeader()
	if err != nil {
		return err
	}

	p := &types.GrandpaPause{}
	dec, err := scale.Decode(d.Data[1:], p)
	if err != nil {
		return err
	}
	p = dec.(*types.GrandpaPause)

	delay := big.NewInt(int64(p.Delay))

	h.grandpaPause = &pause{
		atBlock: big.NewInt(-1).Add(curr.Number, delay),
	}

	return nil
}

func (h *DigestHandler) handleResume(d *types.ConsensusDigest) error {
	curr, err := h.blockState.BestBlockHeader()
	if err != nil {
		return err
	}

	p := &types.GrandpaResume{}
	dec, err := scale.Decode(d.Data[1:], p)
	if err != nil {
		return err
	}
	p = dec.(*types.GrandpaResume)

	delay := big.NewInt(int64(p.Delay))

	h.grandpaResume = &resume{
		atBlock: big.NewInt(-1).Add(curr.Number, delay),
	}

	return nil
}

func newGrandpaChange(raw []*types.GrandpaAuthoritiesRaw, delay uint32, currBlock *big.Int) (*grandpaChange, error) {
	auths, err := types.GrandpaAuthoritiesRawToAuthorities(raw)
	if err != nil {
		return nil, err
	}

	d := big.NewInt(int64(delay))

	return &grandpaChange{
		auths:   auths,
		atBlock: big.NewInt(-1).Add(currBlock, d),
	}, nil
}

func (h *DigestHandler) handleBABEOnDisabled(d *types.ConsensusDigest, header *types.Header) error {
	od := &types.BABEOnDisabled{}
	dec, err := scale.Decode(d.Data[1:], od)
	if err != nil {
		return err
	}
	od = dec.(*types.BABEOnDisabled)

	logger.Debug("handling BABEOnDisabled", "data", od)

	err = h.verifier.SetOnDisabled(od.ID, header)
	if err != nil {
		return err
	}

	h.babe.SetOnDisabled(od.ID)
	return nil
}

func (h *DigestHandler) handleNextEpochData(d *types.ConsensusDigest, header *types.Header) error {
	od := &types.NextEpochData{}
	dec, err := scale.Decode(d.Data[1:], od)
	if err != nil {
		return err
	}
	od = dec.(*types.NextEpochData)

	logger.Debug("handling BABENextEpochData", "data", od)

	currEpoch, err := h.epochState.GetEpochForBlock(header)
	if err != nil {
		return err
	}

	// set EpochState epoch data for upcoming epoch
	data, err := od.ToEpochData()
	if err != nil {
		return err
	}

	logger.Debug("setting epoch data", "blocknum", header.Number, "epoch", currEpoch+1, "data", data)
	return h.epochState.SetEpochData(currEpoch+1, data)
}

func (h *DigestHandler) handleNextConfigData(d *types.ConsensusDigest, header *types.Header) error {
	od := &types.NextConfigData{}
	dec, err := scale.Decode(d.Data[1:], od)
	if err != nil {
		return err
	}
	od = dec.(*types.NextConfigData)

	logger.Debug("handling BABENextConfigData", "data", od)

	currEpoch, err := h.epochState.GetEpochForBlock(header)
	if err != nil {
		return err
	}

	logger.Debug("setting BABE config data", "blocknum", header.Number, "epoch", currEpoch+1, "data", od.ToConfigData())
	// set EpochState config data for upcoming epoch
	return h.epochState.SetConfigData(currEpoch+1, od.ToConfigData())
}
