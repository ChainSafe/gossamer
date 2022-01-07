package babe

import (
	"context"
	"errors"
	"fmt"
	"sort"
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
	slotToProof := make(map[uint64]*VrfOutputAndProof)
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
		logger.Debugf("epoch %d: claimed slot %d", epochNumber, i)
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

	// if currSlot < h.firstSlot, it means we're at genesis and waiting for the first slot to arrive.
	// we have to check it here to prevent int overflow.
	if currSlot >= h.firstSlot && currSlot-h.firstSlot > h.constants.epochLength {
		logger.Warnf("attempted to start epoch that has passed: current slot=%d, start slot of epoch=%d",
			currSlot, h.firstSlot,
		)
		errCh <- errEpochPast
		return
	}

	// invoke block authoring in the next slot, this gives us ample time to setup
	// and make sure the timing is correct.
	invokationSlot := currSlot + 1

	// for each slot we're handling, create a timer that will fire when it starts
	// we create timers only for slots where we're authoring
	authoringSlots := getAuthoringSlots(h.slotToProof)

	type slotWithTimer struct {
		timer   <-chan time.Time
		slotNum uint64
	}

	slotTimeTimers := []*slotWithTimer{}
	for _, authoringSlot := range authoringSlots {
		// ignore slots already passed
		if authoringSlot < invokationSlot {
			continue
		}

		startTime := getSlotStartTime(authoringSlot, h.constants.slotDuration)
		slotTimeTimers = append(slotTimeTimers, &slotWithTimer{
			timer:   time.After(time.Until(startTime)),
			slotNum: authoringSlot,
		})
		logger.Debugf("start time of slot %d: %v", authoringSlot, startTime)
	}

	logger.Debugf("authoring in %d slots in epoch %d", len(slotTimeTimers), h.epochNumber)

	for _, swt := range slotTimeTimers {
		logger.Debugf("waiting for next authoring slot %d", swt.slotNum)

		select {
		case <-h.ctx.Done():
			return
		case <-swt.timer:
			if _, has := h.slotToProof[swt.slotNum]; !has {
				logger.Errorf("no VRF proof for authoring slot! slot=%d", swt.slotNum)
				continue
			}

			if err := h.handleSlot(h.epochNumber, swt.slotNum, h.epochData.authorityIndex, h.slotToProof[swt.slotNum]); err != nil { //nolint:lll
				logger.Warnf("failed to handle slot %d: %s", swt.slotNum, err)
				continue
			}
		}
	}
}

func getAuthoringSlots(slotToProof map[uint64]*VrfOutputAndProof) []uint64 {
	authoringSlots := make([]uint64, len(slotToProof))
	i := 0
	for authoringSlot := range slotToProof {
		authoringSlots[i] = authoringSlot
		i++
	}

	sort.Slice(authoringSlots, func(i, j int) bool {
		return authoringSlots[i] < authoringSlots[j]
	})

	return authoringSlots
}
