// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package babe

import (
	"context"
	"fmt"
	"time"
)

// timeUntilNextSlot calculates, based on the current system time, the remainng
// time to the next slot
func timeUntilNextSlot(slotDuration time.Duration) time.Duration {
	now := time.Now().UnixNano()
	slotDurationInNano := slotDuration.Nanoseconds()

	nextSlot := (now + slotDurationInNano) / slotDurationInNano

	remaining := nextSlot*slotDurationInNano - now
	return time.Duration(remaining)
}

type slotHandler struct {
	slotDuration time.Duration
	lastSlot     *Slot
}

func newSlotHandler(slotDuration time.Duration) slotHandler {
	return slotHandler{
		slotDuration: slotDuration,
	}
}

// waitForNextSlot returns a new Slot greater than the last one when a new slot starts
// based on the current system time similar to:
// https://github.com/paritytech/substrate/blob/fbddfbd76c60c6fda0024e8a44e82ad776033e4b/client/consensus/slots/src/slots.rs#L125
func (s *slotHandler) waitForNextSlot(ctx context.Context) (Slot, error) {
	for {
		// check if there is enough time to collaborate
		untilNextSlot := timeUntilNextSlot(s.slotDuration)
		oneThirdSlotDuration := s.slotDuration / 3
		if untilNextSlot <= oneThirdSlotDuration {
			err := waitUntilNextSlot(ctx, untilNextSlot)
			if err != nil {
				return Slot{}, fmt.Errorf("waiting next slot: %w", err)
			}
		}

		currentSystemTime := time.Now()
		currentSlotNumber := uint64(currentSystemTime.UnixNano()) / uint64(s.slotDuration.Nanoseconds())
		currentSlot := Slot{
			start:    currentSystemTime,
			duration: s.slotDuration,
			number:   currentSlotNumber,
		}

		// Never yield the same slot twice
		if s.lastSlot == nil || currentSlot.number > s.lastSlot.number {
			s.lastSlot = &currentSlot
			return currentSlot, nil
		}

		err := waitUntilNextSlot(ctx, untilNextSlot)
		if err != nil {
			return Slot{}, fmt.Errorf("waiting next slot: %w", err)
		}
	}
}

// waitUntilNextSlot is a blocking function that uses context.WithTimeout
// to "sleep", however if the parent context is canceled it releases with
// context.Canceled error
func waitUntilNextSlot(ctx context.Context, untilNextSlot time.Duration) error {
	withTimeout, cancelWithTimeout := context.WithTimeout(ctx, untilNextSlot)
	defer cancelWithTimeout()

	<-withTimeout.Done()

	parentCtxErr := ctx.Err()
	if parentCtxErr != nil {
		return parentCtxErr
	}

	return nil
}
