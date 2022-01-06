package babe

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
)

type handleSlotFunc = func(epoch, slotNum uint64, authorityIndex uint32, proof *VrfOutputAndProof) error

var (
	errEpochPast = errors.New("cannot run epoch that has already passed")
)

type constants struct {
	slotDuration time.Duration
	epochLength  uint64
}

type epochHandler struct {
	ctx context.Context

	epochNumber uint64
	startTime   time.Time
	firstSlot   uint64

	constants *constants
	epochData *epochData

	// for slots where we are a producer, store the vrf output (bytes 0-32) + proof (bytes 32-96)
	slotToProof map[uint64]*VrfOutputAndProof

	handleSlot handleSlotFunc
}

func newEpochHandler(ctx context.Context, epochNumber, firstSlot uint64, epochData *epochData, constants *constants,
	handleSlot handleSlotFunc, keypair *sr25519.Keypair) (*epochHandler, error) {

	startTime := getSlotStartTime(firstSlot, constants.slotDuration)

	// determine which slots we'll be authoring in by pre-calculating VRF output
	slotToProof := make(map[uint64]*VrfOutputAndProof, constants.epochLength)
	for i := firstSlot; i < firstSlot+constants.epochLength; i++ {
		proof, err := claimPrimarySlot(
			epochData.randomness,
			i,
			epochNumber,
			epochData.threshold,
			keypair,
		)
		if err != nil {
			if errors.Is(err, errOverPrimarySlotThreshold) {
				continue
			}
			return nil, fmt.Errorf("error running slot lottery at slot %d: error %w", i, err)
		}

		slotToProof[i] = proof
		logger.Tracef("claimed slot %d", i)
	}

	return &epochHandler{
		ctx:         ctx,
		epochNumber: epochNumber,
		firstSlot:   firstSlot,
		startTime:   startTime,
		constants:   constants,
		epochData:   epochData,
		slotToProof: slotToProof,
		handleSlot:  handleSlot,
	}, nil
}

func (h *epochHandler) run(errCh chan<- error) {
	currSlot := getCurrentSlot(h.constants.slotDuration)
	if currSlot-h.firstSlot > h.constants.epochLength {
		errCh <- errEpochPast
		return
	}

	// invoke block authoring in the next slot, this gives us ample time to setup
	// and make sure the timing is correct.
	invokationSlot := currSlot + 1

	// calculate how many slots we are handling this epoch
	numSlots := h.constants.epochLength - (invokationSlot - h.firstSlot)

	// for each slot we're handling, create a timer that will fire when it starts
	// TODO: create timers only for slots where we're authoring
	slotTimeTimers := make([]<-chan time.Time, numSlots)
	for i := uint64(0); i < numSlots; i++ {
		startTime := getSlotStartTime(invokationSlot+i, h.constants.slotDuration)
		slotTimeTimers[i] = time.After(time.Until(startTime))
	}

	for i := uint64(0); i < numSlots; i++ {
		select {
		case <-h.ctx.Done():
			return
		case <-slotTimeTimers[i]:
			slotNum := invokationSlot + i

			// check if we can author a block in this slot
			if _, has := h.slotToProof[slotNum]; !has {
				continue
			}

			if err := h.handleSlot(h.epochNumber, slotNum, h.epochData.authorityIndex, h.slotToProof[slotNum]); err != nil {
				logger.Warnf("failed to handle slot %d: %s", slotNum, err)
				continue
			}
		}
	}
}
