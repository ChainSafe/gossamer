// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package babe

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSlotHandlerConstructor(t *testing.T) {
	t.Parallel()

	expected := slotHandler{
		slotDuration: time.Duration(6000),
	}

	handler := newSlotHandler(time.Duration(6000))
	require.Equal(t, expected, handler)
}

func TestSlotHandlerNextSlot(t *testing.T) {
	t.Parallel()

	const slotDuration = 2 * time.Second
	handler := newSlotHandler(slotDuration)

	firstIteration, err := handler.waitForNextSlot(context.Background())
	require.NoError(t, err)

	secondIteration, err := handler.waitForNextSlot(context.Background())
	require.NoError(t, err)

	require.Greater(t, secondIteration.number, firstIteration.number)
}

func TestSlotHandlerNextSlot_ContextCanceled(t *testing.T) {
	t.Parallel()

	const slotDuration = 2 * time.Second
	handler := newSlotHandler(slotDuration)

	ctx, cancel := context.WithCancel(context.Background())

	firstIteration, err := handler.waitForNextSlot(ctx)
	require.NoError(t, err)
	require.NotEqual(t, Slot{}, firstIteration)

	cancel()

	secondIteration, err := handler.waitForNextSlot(ctx)
	require.Equal(t, Slot{}, secondIteration)
	require.ErrorIs(t, err, context.Canceled)
	require.EqualError(t, err, "waiting next slot: context canceled")
}
