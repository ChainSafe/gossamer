package babe

import (
	"fmt"
	"time"
)

// timeUntilNextSlotInNanos calculates, based on the current system time, the remainng
// time to the next slot
func timeUntilNextSlotInMilli(slotDuration time.Duration) time.Duration {
	now := time.Now().UnixNano()
	slotDurationInMilli := slotDuration.Nanoseconds()

	nextSlot := (now + slotDurationInMilli) / slotDurationInMilli

	remaining := nextSlot*slotDurationInMilli - now
	return time.Duration(remaining)
}

type slotHandler struct {
	slotDuration  time.Duration
	untilNextSlot *time.Duration
	lastSlot      *Slot
}

func newSlotHandler(slotDuration time.Duration) *slotHandler {
	return &slotHandler{
		slotDuration: slotDuration,
	}
}

func (s *slotHandler) waitForNextSlot() (Slot, error) {
	for {
		if s.untilNextSlot != nil {
			time.Sleep(*s.untilNextSlot)
		} else {
			// first timeout
			waitDuration := timeUntilNextSlotInMilli(s.slotDuration)
			time.Sleep(waitDuration)
		}

		waitDuration := timeUntilNextSlotInMilli(s.slotDuration)
		fmt.Printf("time until next slot: %d\n", waitDuration)
		s.untilNextSlot = &waitDuration

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
	}
}
