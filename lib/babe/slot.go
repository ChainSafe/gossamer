package babe

import (
	"errors"
	"fmt"
	"time"
)

func timeUntilNextSlot(slotDuration time.Duration) time.Duration {
	fmt.Printf("slot duration: %v\nslot duration in milli: %v\n", slotDuration, slotDuration.Milliseconds())

	nowInMillis := time.Now().UnixMilli()
	slotDurationInMillis := slotDuration.Milliseconds()

	nextSlot := (nowInMillis + slotDurationInMillis) / slotDurationInMillis

	remainingMillis := nextSlot*slotDurationInMillis - nowInMillis
	return time.Duration(remainingMillis)
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
	if s.untilNextSlot == nil {
		dur := timeUntilNextSlot(s.slotDuration)
		fmt.Printf("waiting %d mill\n", dur)

		time.Sleep(dur)
	} else {
		fmt.Printf("waiting %d milli\n", s.untilNextSlot.Milliseconds())
		time.Sleep(*s.untilNextSlot)
	}

	waitDuration := timeUntilNextSlot(s.slotDuration)
	s.untilNextSlot = &waitDuration

	currentSystemTime := time.Now()
	currentSlotNumber := uint64(currentSystemTime.UnixNano()) / uint64(s.slotDuration.Nanoseconds())
	currentSlot := Slot{
		start:    currentSystemTime,
		duration: s.slotDuration,
		number:   currentSlotNumber,
	}

	if s.lastSlot == nil {
		s.lastSlot = &currentSlot
		return currentSlot, nil
	}

	// Never yield the same slot twice.
	if currentSlot.number <= s.lastSlot.number {
		return Slot{}, errors.New("issue a slot equal or lower than the latest one")
	}

	s.lastSlot = &currentSlot
	return currentSlot, nil
}
