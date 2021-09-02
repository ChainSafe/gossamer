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
	"fmt"
	"math"
	"math/big"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/services"
	"github.com/ChainSafe/gossamer/pkg/scale"
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
	imported    chan *types.BlockVdt
	importedID  byte
	finalised   chan *types.FinalisationInfoVdt
	finalisedID byte

	// GRANDPA changes
	grandpaScheduledChange *grandpaChange
	grandpaForcedChange    *grandpaChange
	grandpaPause           *pause
	grandpaResume          *resume
}

type grandpaChange struct {
	auths   []types.Authority
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
	imported := make(chan *types.BlockVdt, 16)
	finalised := make(chan *types.FinalisationInfoVdt, 16)
	iid, err := blockState.RegisterImportedChannel(imported)
	if err != nil {
		return nil, err
	}

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
		importedID:   iid,
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
	h.blockState.UnregisterImportedChannel(h.importedID)
	h.blockState.UnregisterFinalisedChannel(h.finalisedID)
	close(h.imported)
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

func (h *Handler) HandleDigestsVdt(header *types.HeaderVdt) {
	for i, d := range header.Digest.Types {
		switch val := d.Value().(type) {
		case types.ConsensusDigest:
			err := h.handleConsensusDigestVdt(&val, header)
			if err != nil {
				logger.Error("handleDigests", "block number", header.Number, "index", i, "digest", d.Value(), "error", err)
			}
		}
	}
}

// TODO Delete
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

func (h *Handler) handleConsensusDigestVdt(d *types.ConsensusDigest, header *types.HeaderVdt) error {
	t := d.DataType()

	if d.ConsensusEngineID == types.GrandpaEngineID {
		switch t {
		case byte(types.GrandpaScheduledChange{}.Index()):
			return h.handleScheduledChangeVdt(d, header)
		case byte(types.GrandpaForcedChange{}.Index()):
			return h.handleForcedChangeVdt(d, header)
		case byte(types.GrandpaOnDisabled{}.Index()):
			return nil // do nothing, as this is not implemented in substrate
		case byte(types.GrandpaPause{}.Index()):
			return h.handlePause(d)
		case byte(types.GrandpaResume{}.Index()):
			return h.handleResume(d)
		default:
			return errors.New("invalid consensus digest data")
		}
	}

	if d.ConsensusEngineID == types.BabeEngineID {
		switch t {
		case byte(types.NextEpochData{}.Index()):
			return h.handleNextEpochDataVdt(d, header)
		case byte(types.BABEOnDisabled{}.Index()):
			return h.handleBABEOnDisabledVdt(d, header)
		case byte(types.NextConfigData{}.Index()):
			return h.handleNextConfigDataVdt(d, header)
		default:
			return errors.New("invalid consensus digest data")
		}
	}

	return errors.New("unknown consensus engine ID")
}

func (h *Handler) handleConsensusDigest(d *types.ConsensusDigest, header *types.Header) error {
	t := d.DataType()

	if d.ConsensusEngineID == types.GrandpaEngineID {
		switch t {
		case byte(types.GrandpaScheduledChange{}.Index()):
			return h.handleScheduledChange(d, header)
		case byte(types.GrandpaForcedChange{}.Index()):
			return h.handleForcedChange(d, header)
		case byte(types.GrandpaOnDisabled{}.Index()):
			return nil // do nothing, as this is not implemented in substrate
		case byte(types.GrandpaPause{}.Index()):
			return h.handlePause(d)
		case byte(types.GrandpaResume{}.Index()):
			return h.handleResume(d)
		default:
			return errors.New("invalid consensus digest data")
		}
	}

	if d.ConsensusEngineID == types.BabeEngineID {
		switch t {
		case byte(types.NextEpochData{}.Index()):
			return h.handleNextEpochData(d, header)
		case byte(types.BABEOnDisabled{}.Index()):
			return h.handleBABEOnDisabled(d, header)
		case byte(types.NextConfigData{}.Index()):
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
			//if block == nil || block.Header == nil {
			if block == nil {
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
			if info == nil {
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

func (h *Handler) handleScheduledChangeVdt(d *types.ConsensusDigest, header *types.HeaderVdt) error {
	curr, err := h.blockState.BestBlockHeaderVdt()
	if err != nil {
		return err
	}

	if d.ConsensusEngineID != types.GrandpaEngineID {
		return nil
	}

	if h.grandpaScheduledChange != nil {
		return nil
	}

	var dec = types.NewGrandpaConsensusDigest()
	err = scale.Unmarshal(d.Data, &dec)
	if err != nil {
		return err
	}

	var sc types.GrandpaScheduledChange
	switch val := dec.Value().(type) {
	case types.GrandpaScheduledChange:
		sc = val
	default:
		return fmt.Errorf("expected ConsensusDigest of type GrandpaScheduledChange, but got: %T", val)
	}

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

func (h *Handler) handleScheduledChange(d *types.ConsensusDigest, header *types.Header) error {
	curr, err := h.blockState.BestBlockHeaderVdt()
	if err != nil {
		return err
	}

	if d.ConsensusEngineID != types.GrandpaEngineID {
		return nil
	}

	if h.grandpaScheduledChange != nil {
		return nil
	}

	var dec = types.NewGrandpaConsensusDigest()
	err = scale.Unmarshal(d.Data, &dec)
	if err != nil {
		return err
	}

	var sc types.GrandpaScheduledChange
	switch val := dec.Value().(type) {
	case types.GrandpaScheduledChange:
		sc = val
	default:
		return fmt.Errorf("expected ConsensusDigest of type GrandpaScheduledChange, but got: %T", val)
	}

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

func (h *Handler) handleForcedChangeVdt(d *types.ConsensusDigest, header *types.HeaderVdt) error {
	if d.ConsensusEngineID != types.GrandpaEngineID {
		return nil
	}

	if header == nil {
		return errors.New("header is nil")
	}

	if h.grandpaForcedChange != nil {
		return errors.New("already have forced change scheduled")
	}

	var dec = types.NewGrandpaConsensusDigest()
	err := scale.Unmarshal(d.Data, &dec)
	if err != nil {
		return err
	}

	var fc types.GrandpaForcedChange
	switch val := dec.Value().(type) {
	case types.GrandpaForcedChange:
		fc = val
	default:
		return fmt.Errorf("expected ConsensusDigest of type GrandpaForcedChange, but got: %T", val)
	}

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

	var dec = types.NewGrandpaConsensusDigest()
	err := scale.Unmarshal(d.Data, &dec)
	if err != nil {
		return err
	}

	var fc types.GrandpaForcedChange
	switch val := dec.Value().(type) {
	case types.GrandpaForcedChange:
		fc = val
	default:
		return fmt.Errorf("expected ConsensusDigest of type GrandpaForcedChange, but got: %T", val)
	}

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
	curr, err := h.blockState.BestBlockHeaderVdt()
	if err != nil {
		return err
	}

	var dec = types.NewGrandpaConsensusDigest()
	err = scale.Unmarshal(d.Data, &dec)
	if err != nil {
		return err
	}

	var p types.GrandpaPause
	switch val := dec.Value().(type) {
	case types.GrandpaPause:
		p = val
	default:
		return fmt.Errorf("expected ConsensusDigest of type GrandpaPause, but got: %T", val)
	}

	delay := big.NewInt(int64(p.Delay))

	h.grandpaPause = &pause{
		atBlock: big.NewInt(-1).Add(curr.Number, delay),
	}

	return h.grandpaState.SetNextPause(h.grandpaPause.atBlock)
}

func (h *Handler) handleResume(d *types.ConsensusDigest) error {
	curr, err := h.blockState.BestBlockHeaderVdt()
	if err != nil {
		return err
	}

	var dec = types.NewGrandpaConsensusDigest()
	err = scale.Unmarshal(d.Data, &dec)
	if err != nil {
		return err
	}

	var r types.GrandpaResume
	switch val := dec.Value().(type) {
	case types.GrandpaResume:
		r = val
	default:
		return fmt.Errorf("expected ConsensusDigest of type GrandpaResume, but got: %T", val)
	}

	delay := big.NewInt(int64(r.Delay))

	h.grandpaResume = &resume{
		atBlock: big.NewInt(-1).Add(curr.Number, delay),
	}

	return h.grandpaState.SetNextResume(h.grandpaResume.atBlock)
}

func newGrandpaChange(raw []types.GrandpaAuthoritiesRaw, delay uint32, currBlock *big.Int) (*grandpaChange, error) {
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

func (h *Handler) handleBABEOnDisabledVdt(d *types.ConsensusDigest, _ *types.HeaderVdt) error {
	od := &types.BABEOnDisabled{}
	logger.Debug("handling BABEOnDisabled", "data", od)
	return nil
}

func (h *Handler) handleBABEOnDisabled(d *types.ConsensusDigest, _ *types.Header) error {
	od := &types.BABEOnDisabled{}
	logger.Debug("handling BABEOnDisabled", "data", od)
	return nil
}

func (h *Handler) handleNextEpochDataVdt(d *types.ConsensusDigest, header *types.HeaderVdt) error {
	var od = types.NewBabeConsensusDigest()
	err := scale.Unmarshal(d.Data, &od)
	if err != nil {
		return err
	}

	logger.Debug("handling BABENextEpochData", "data", od)

	currEpoch, err := h.epochState.GetEpochForBlockVdt(header)
	if err != nil {
		return err
	}

	var act types.NextEpochData
	switch val := od.Value().(type) {
	case types.NextEpochData:
		act = val
	default:
		return fmt.Errorf("expected ConsensusDigest of type NextEpochData, but got: %T", val)
	}

	// set EpochState epoch data for upcoming epoch
	data, err := act.ToEpochData()
	if err != nil {
		return err
	}

	logger.Debug("setting epoch data", "blocknum", header.Number, "epoch", currEpoch+1, "data", data)
	return h.epochState.SetEpochData(currEpoch+1, data)
}

func (h *Handler) handleNextEpochData(d *types.ConsensusDigest, header *types.Header) error {
	var od = types.NewBabeConsensusDigest()
	err := scale.Unmarshal(d.Data, &od)
	if err != nil {
		return err
	}

	logger.Debug("handling BABENextEpochData", "data", od)

	currEpoch, err := h.epochState.GetEpochForBlock(header)
	if err != nil {
		return err
	}

	var act types.NextEpochData
	switch val := od.Value().(type) {
	case types.NextEpochData:
		act = val
	default:
		return fmt.Errorf("expected ConsensusDigest of type NextEpochData, but got: %T", val)
	}

	// set EpochState epoch data for upcoming epoch
	data, err := act.ToEpochData()
	if err != nil {
		return err
	}

	logger.Debug("setting epoch data", "blocknum", header.Number, "epoch", currEpoch+1, "data", data)
	return h.epochState.SetEpochData(currEpoch+1, data)
}

func (h *Handler) handleNextConfigDataVdt(d *types.ConsensusDigest, header *types.HeaderVdt) error {
	var od = types.NewBabeConsensusDigest()
	err := scale.Unmarshal(d.Data, &od)
	if err != nil {
		return err
	}

	logger.Debug("handling BABENextConfigData", "data", od)

	currEpoch, err := h.epochState.GetEpochForBlockVdt(header)
	if err != nil {
		return err
	}

	var config types.NextConfigData
	switch val := od.Value().(type) {
	case types.NextConfigData:
		config = val
	default:
		return fmt.Errorf("expected ConsensusDigest of type NextConfigData, but got: %T", val)
	}

	logger.Debug("setting BABE config data", "blocknum", header.Number, "epoch", currEpoch+1, "data", config.ToConfigData())
	// set EpochState config data for upcoming epoch
	return h.epochState.SetConfigData(currEpoch+1, config.ToConfigData())
}

func (h *Handler) handleNextConfigData(d *types.ConsensusDigest, header *types.Header) error {
	var od = types.NewBabeConsensusDigest()
	err := scale.Unmarshal(d.Data, &od)
	if err != nil {
		return err
	}

	logger.Debug("handling BABENextConfigData", "data", od)

	currEpoch, err := h.epochState.GetEpochForBlock(header)
	if err != nil {
		return err
	}

	var config types.NextConfigData
	switch val := od.Value().(type) {
	case types.NextConfigData:
		config = val
	default:
		return fmt.Errorf("expected ConsensusDigest of type NextConfigData, but got: %T", val)
	}

	logger.Debug("setting BABE config data", "blocknum", header.Number, "epoch", currEpoch+1, "data", config.ToConfigData())
	// set EpochState config data for upcoming epoch
	return h.epochState.SetConfigData(currEpoch+1, config.ToConfigData())
}
