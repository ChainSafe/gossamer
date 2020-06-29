package core

import (
	"errors"
	"math/big"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/scale"
)

type digestHandlerI interface {
	handleConsensusDigest(*types.ConsensusDigest) error
}

type digestHandler struct {
	// interfaces
	blockState BlockState
	grandpa    FinalityGadget
	babe       BlockProducer

	// state variables
	stopped bool

	// BABE changes
	babeScheduledChange *babeChange
	babeForcedChange    *babeChange
	babePause           *pause
	babeResume          *resume

	// GRANDPA changes
	grandpaScheduledChange *grandpaChange
	grandpaForcedChange    *grandpaChange
	grandpaPause           *pause
	grandpaResume          *resume
}

type babeChange struct {
	auths   []*types.BABEAuthorityData
	atBlock *big.Int
}

type grandpaChange struct {
	auths   []*types.GrandpaAuthorityData
	atBlock *big.Int
}

type pause struct {
	atBlock *big.Int
}

type resume struct {
	atBlock *big.Int
}

func newDigestHandler() *digestHandler {
	return &digestHandler{}
}

func (h *digestHandler) start() {
	go h.handleGrandpaChanges()
	h.stopped = false
}

func (h *digestHandler) stop() {
	h.stopped = true
}

func (h *digestHandler) handleGrandpaChanges() {
	for {
		if h.stopped {
			return
		}

		curr, err := h.blockState.BestBlockHeader()
		if err != nil {
			continue
		}

		sc := h.grandpaScheduledChange
		if sc != nil {
			if curr.Number.Cmp(sc.atBlock) == 0 {
				h.grandpa.UpdateAuthorities(sc.auths)
			}
		}

		fc := h.grandpaForcedChange
		if fc != nil {
			if curr.Number.Cmp(fc.atBlock) == 0 {
				h.grandpa.UpdateAuthorities(fc.auths)
			}
		}
	}
}

func (h *digestHandler) handleConsensusDigest(d *types.ConsensusDigest) error {
	t := d.DataType()

	switch t {
	case types.ScheduledChangeType:
		return h.handleScheduledChange(d)
	case types.ForcedChangeType:
		return h.handleForcedChange(d)
	case types.DisabledType:
	case types.PauseType:
	case types.ResumeType:
	default:
		return errors.New("invalid consensus digest data")
	}

	return nil
}

func (h *digestHandler) handleScheduledChange(d *types.ConsensusDigest) error {
	curr, err := h.blockState.BestBlockHeader()
	if err != nil {
		return err
	}

	if d.ConsensusEngineID == types.BabeEngineID {
		// TODO
	} else {
		if h.grandpaScheduledChange != nil {
			return errors.New("already have scheduled change scheduled")
		}

		sc := &types.GrandpaScheduledChange{}
		dec, err := scale.Decode(d.Data, sc)
		if err != nil {
			return err
		}
		sc = dec.(*types.GrandpaScheduledChange)

		c, err := newGrandpaChange(sc.Auths, sc.Delay, curr.Number)
		if err != nil {
			return err
		}

		h.grandpaScheduledChange = c
	}

	return nil
}

func (h *digestHandler) handleForcedChange(d *types.ConsensusDigest) error {
	curr, err := h.blockState.BestBlockHeader()
	if err != nil {
		return err
	}

	if d.ConsensusEngineID == types.BabeEngineID {
		// TODO
	} else {
		if h.grandpaForcedChange != nil {
			return errors.New("already have forced change scheduled")
		}

		fc := &types.GrandpaForcedChange{}
		dec, err := scale.Decode(d.Data, fc)
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

func newGrandpaChange(raw []*types.GrandpaAuthorityDataRaw, delay uint32, currBlock *big.Int) (*grandpaChange, error) {
	auths, err := types.GrandpaAuthorityDataRawToAuthorityData(raw)
	if err != nil {
		return nil, err
	}

	d := big.NewInt(int64(delay))

	return &grandpaChange{
		auths:   auths,
		atBlock: big.NewInt(0).Add(currBlock, d),
	}, nil
}
