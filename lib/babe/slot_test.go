package babe

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSlotHandeConstructor(t *testing.T) {
	expected := &slotHandler{
		slotDuration: time.Duration(6000),
	}

	handler := newSlotHandler(time.Duration(6000))
	require.Equal(t, expected, handler)
}

func TestSlotHandlerNextSlot(t *testing.T) {
	slotDuration := 2 * time.Second
	handler := newSlotHandler(slotDuration)

	firstIteration, err := handler.waitForNextSlot()
	require.NoError(t, err)

	secondIteration, err := handler.waitForNextSlot()
	require.NoError(t, err)

	require.Greater(t, secondIteration.number, firstIteration.number)
}
