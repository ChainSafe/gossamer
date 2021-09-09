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

package digest

import (
	"context"
	"errors"
	"math"
	"math/big"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/scale"
	"github.com/ChainSafe/gossamer/lib/services"

	log "github.com/ChainSafe/log15"
)

var maxUint64 = uint64(math.MaxUint64)

var (
	_      services.Service = &Handler{}
	logger log.Logger       = log.New("pkg", "digest") // TODO: add to config options
)

// Handler is used to handle consensus messages and relevant authority updates to BABE and GRANDPA
type Handler struct {
	ctx    context.Context
	cancel context.CancelFunc

	// interfaces
	blockState   BlockState
	epochState   EpochState
	grandpaState GrandpaState

	// block notification channels
	imported    <-chan *types.Block
	finalised   chan *types.FinalisationInfo
	finalisedID byte

	// GRANDPA changes
	grandpaScheduledChange *grandpaChange
	grandpaForcedChange    *grandpaChange
	grandpaPause           *pause
	grandpaResume          *resume
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

// NewHandler returns a new Handler
func NewHandler(blockState BlockState, epochState EpochState, grandpaState GrandpaState) (*Handler, error) {
	imported, err := blockState.GetImportedBlockNotifierChannel(blockState)
	if err != nil {
		return nil, err
	}

	finalised := make(chan *types.FinalisationInfo, 16)

	fid, err := blockState.RegisterFinalizedChannel(finalised)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Handler{
		ctx:          ctx,
		cancel:       cancel,
		blockState:   blockState,
		epochState:   epochState,
		grandpaState: grandpaState,
		imported:     imported,
		finalised:    finalised,
		finalisedID:  fid,
	}, nil
}

// Start starts the Handler
func (h *Handler) Start() error {
	go h.handleBlockImport(h.ctx)
	go h.handleBlockFinalisation(h.ctx)
	return nil
}

// Stop stops the Handler
func (h *Handler) Stop() error {
	h.cancel()
	h.blockState.FreeImportedBlockNotifierChannel(h.blockState)
	h.blockState.UnregisterFinalisedChannel(h.finalisedID)
	close(h.finalised)
	return nil
}

// NextGrandpaAuthorityChange returns the block number of the next upcoming grandpa authorities change.
// It returns 0 if no change is scheduled.
func (h *Handler) NextGrandpaAuthorityChange() uint64 {
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

// HandleDigests handles consensus digests for an imported block
func (h *Handler) HandleDigests(header *types.Header) {
	for i, d := range header.Digest {
		if d.Type() == types.ConsensusDigestType {
			cd, ok := d.(*types.ConsensusDigest)
			if !ok {
				logger.Error("handleDigests", "block number", header.Number, "index", i, "error", "cannot cast invalid consensus digest item")
				continue
			}

			err := h.handleConsensusDigest(cd, header)
			if err != nil {
				logger.Error("handleDigests", "block number", header.Number, "index", i, "digest", cd, "error", err)
			}
		}
	}
}

func (h *Handler) handleConsensusDigest(d *types.ConsensusDigest, header *types.Header) error {
	t := d.DataType()

	if d.ConsensusEngineID == types.GrandpaEngineID {
		switch t {
		case types.GrandpaScheduledChangeType:
			return h.handleScheduledChange(d, header)
		case types.GrandpaForcedChangeType:
			return h.handleForcedChange(d, header)
		case types.GrandpaOnDisabledType:
			return nil // do nothing, as this is not implemented in substrate
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

func (h *Handler) handleBlockImport(ctx context.Context) {
	for {
		select {
		case block := <-h.imported:
			if block == nil || block.Header == nil {
				continue
			}

			err := h.handleGrandpaChangesOnImport(block.Header.Number)
			if err != nil {
				logger.Error("failed to handle grandpa changes on block import", "error", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (h *Handler) handleBlockFinalisation(ctx context.Context) {
	for {
		select {
		case info := <-h.finalised:
			if info == nil || info.Header == nil {
				continue
			}

			err := h.handleGrandpaChangesOnFinalization(info.Header.Number)
			if err != nil {
				logger.Error("failed to handle grandpa changes on block finalisation", "error", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (h *Handler) handleGrandpaChangesOnImport(num *big.Int) error {
	resume := h.grandpaResume
	if resume != nil && num.Cmp(resume.atBlock) > -1 {
		h.grandpaResume = nil
	}

	fc := h.grandpaForcedChange
	if fc != nil && num.Cmp(fc.atBlock) > -1 {
		err := h.grandpaState.IncrementSetID()
		if err != nil {
			return err
		}

		h.grandpaForcedChange = nil
		curr, err := h.grandpaState.GetCurrentSetID()
		if err != nil {
			return err
		}

		logger.Debug("incremented grandpa set ID", "set ID", curr)
	}

	return nil
}

func (h *Handler) handleGrandpaChangesOnFinalization(num *big.Int) error {
	pause := h.grandpaPause
	if pause != nil && num.Cmp(pause.atBlock) > -1 {
		h.grandpaPause = nil
	}

	sc := h.grandpaScheduledChange
	if sc != nil && num.Cmp(sc.atBlock) > -1 {
		err := h.grandpaState.IncrementSetID()
		if err != nil {
			return err
		}

		h.grandpaScheduledChange = nil
		curr, err := h.grandpaState.GetCurrentSetID()
		if err != nil {
			return err
		}

		logger.Debug("incremented grandpa set ID", "set ID", curr)
	}

	// if blocks get finalised before forced change takes place, disregard it
	h.grandpaForcedChange = nil
	return nil
}

func (h *Handler) handleScheduledChange(d *types.ConsensusDigest, header *types.Header) error {
	curr, err := h.blockState.BestBlockHeader()
	if err != nil {
		return err
	}

	if d.ConsensusEngineID != types.GrandpaEngineID {
		return nil
	}

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

	c, err := newGrandpaChange(sc.Auths, sc.Delay, curr.Number)
	if err != nil {
		return err
	}

	h.grandpaScheduledChange = c

	auths, err := types.GrandpaAuthoritiesRawToAuthorities(sc.Auths)
	if err != nil {
		return err
	}

	logger.Debug("setting GrandpaScheduledChange", "at block", big.NewInt(0).Add(header.Number, big.NewInt(int64(sc.Delay))))
	return h.grandpaState.SetNextChange(
		types.NewGrandpaVotersFromAuthorities(auths),
		big.NewInt(0).Add(header.Number, big.NewInt(int64(sc.Delay))),
	)
}

func (h *Handler) handleForcedChange(d *types.ConsensusDigest, header *types.Header) error {
	if d.ConsensusEngineID != types.GrandpaEngineID {
		return nil
	}

	if header == nil {
		return errors.New("header is nil")
	}

	if h.grandpaForcedChange != nil {
		return errors.New("already have forced change scheduled")
	}

	fc := &types.GrandpaForcedChange{}
	dec, err := scale.Decode(d.Data[1:], fc)
	if err != nil {
		return err
	}
	fc = dec.(*types.GrandpaForcedChange)

	logger.Debug("handling GrandpaForcedChange", "data", fc)

	c, err := newGrandpaChange(fc.Auths, fc.Delay, header.Number)
	if err != nil {
		return err
	}

	h.grandpaForcedChange = c

	auths, err := types.GrandpaAuthoritiesRawToAuthorities(fc.Auths)
	if err != nil {
		return err
	}

	logger.Debug("setting GrandpaForcedChange", "at block", big.NewInt(0).Add(header.Number, big.NewInt(int64(fc.Delay))))
	return h.grandpaState.SetNextChange(
		types.NewGrandpaVotersFromAuthorities(auths),
		big.NewInt(0).Add(header.Number, big.NewInt(int64(fc.Delay))),
	)
}

func (h *Handler) handlePause(d *types.ConsensusDigest) error {
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

	return h.grandpaState.SetNextPause(h.grandpaPause.atBlock)
}

func (h *Handler) handleResume(d *types.ConsensusDigest) error {
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

	return h.grandpaState.SetNextResume(h.grandpaResume.atBlock)
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

func (*Handler) handleBABEOnDisabled(_ *types.ConsensusDigest, _ *types.Header) error {
	od := &types.BABEOnDisabled{}
	logger.Debug("handling BABEOnDisabled", "data", od)
	return nil
}

func (h *Handler) handleNextEpochData(d *types.ConsensusDigest, header *types.Header) error {
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

func (h *Handler) handleNextConfigData(d *types.ConsensusDigest, header *types.Header) error {
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
