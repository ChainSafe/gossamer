// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package babe

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
)

type handleSlotFunc = func(epoch, slotNum uint64, authorityIndex uint32, preRuntimeDigest *types.PreRuntimeDigest) error

var (
	errEpochPast = errors.New("cannot run epoch that has already passed")
)

type epochHandler struct {
	epochNumber uint64
	firstSlot   uint64

	constants constants
	epochData *epochData

	slotToPreRuntimeDigest map[uint64]*types.PreRuntimeDigest

	handleSlot handleSlotFunc
}

func newEpochHandler(epochNumber, firstSlot uint64, epochData *epochData, constants constants,
	handleSlot handleSlotFunc, keypair *sr25519.Keypair) (*epochHandler, error) {
	// determine which slots we'll be authoring in by pre-calculating VRF output
	slotToPreRuntimeDigest := make(map[uint64]*types.PreRuntimeDigest, constants.epochLength)
	for i := firstSlot; i < firstSlot+constants.epochLength; i++ {
		preRuntimeDigest, err := claimSlot(epochNumber, i, epochData, keypair)
		if err == nil {
			slotToPreRuntimeDigest[i] = preRuntimeDigest
			continue
		}

		if errors.Is(err, errNotOurTurnToPropose) {
			continue
		}

		return nil, fmt.Errorf("failed to create new epoch handler: %w", err)
	}

	return &epochHandler{
		epochNumber:            epochNumber,
		firstSlot:              firstSlot,
		constants:              constants,
		epochData:              epochData,
		handleSlot:             handleSlot,
		slotToPreRuntimeDigest: slotToPreRuntimeDigest,
	}, nil
}

func (h *epochHandler) run(ctx context.Context, errCh chan<- error) {
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

	// for each slot we're handling, create a timer that will fire when it starts
	// we create timers only for slots where we're authoring
	authoringSlots := getAuthoringSlots(h.slotToPreRuntimeDigest)

	type slotWithTimer struct {
		startTime time.Time
		timer     *time.Timer
		slotNum   uint64
	}

	slotTimeTimers := make([]*slotWithTimer, 0, len(authoringSlots))
	for _, authoringSlot := range authoringSlots {
		if authoringSlot < currSlot {
			// ignore slots already passed
			continue
		}

		startTime := getSlotStartTime(authoringSlot, h.constants.slotDuration)
		waitTime := startTime.Sub(time.Now())
		timer := time.NewTimer(waitTime)

		slotTimeTimers = append(slotTimeTimers, &slotWithTimer{
			timer:     timer,
			slotNum:   authoringSlot,
			startTime: startTime,
		})

		logger.Debugf("start time of slot %d: %v", authoringSlot, startTime)
	}

	defer func() {
		// cleanup timers if ctx was cancelled
		for _, swt := range slotTimeTimers {
			if !swt.timer.Stop() {
				<-swt.timer.C
			}
		}
	}()

	logger.Debugf("authoring in %d slots in epoch %d", len(slotTimeTimers), h.epochNumber)

	for _, swt := range slotTimeTimers {
		logger.Debugf("waiting for next authoring slot %d", swt.slotNum)

		select {
		case <-ctx.Done():
			return
		case <-swt.timer.C:
			// we must do a time correction as the slot timer sometimes is triggered
			// before the time defined in the constructor due to an inconsistency
			// of the language -> https://github.com/golang/go/issues/17696

			diff := time.Since(swt.startTime)
			if diff < 0 {
				time.Sleep(-diff)
			}

			if _, has := h.slotToPreRuntimeDigest[swt.slotNum]; !has {
				// this should never happen
				panic(fmt.Sprintf("no VRF proof for authoring slot! slot=%d", swt.slotNum))
			}

			err := h.handleSlot(h.epochNumber, swt.slotNum, h.epochData.authorityIndex, h.slotToPreRuntimeDigest[swt.slotNum])
			if err != nil {
				logger.Warnf("failed to handle slot %d: %s", swt.slotNum, err)
				continue
			}
		}
	}
}

// getAuthoringSlots returns an ordered slice of slot numbers where we can author blocks,
// based on the given VRF output and proof map.
func getAuthoringSlots(slotToPreRuntimeDigest map[uint64]*types.PreRuntimeDigest) []uint64 {
	authoringSlots := make([]uint64, 0, len(slotToPreRuntimeDigest))
	for authoringSlot := range slotToPreRuntimeDigest {
		authoringSlots = append(authoringSlots, authoringSlot)
	}

	sort.Slice(authoringSlots, func(i, j int) bool {
		return authoringSlots[i] < authoringSlots[j]
	})

	return authoringSlots
}
