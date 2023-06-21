// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package digest

import (
	"context"
	"errors"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/log"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "digest"))

var (
	ErrUnknownConsensusEngineID = errors.New("unknown consensus engine ID")
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
	imported  chan *types.Block
	finalised chan *types.FinalisationInfo
}

// NewHandler returns a new Handler
func NewHandler(blockState BlockState, epochState EpochState,
	grandpaState GrandpaState) (*Handler, error) {
	imported := blockState.GetImportedBlockNotifierChannel()
	finalised := blockState.GetFinalisedNotifierChannel()

	ctx, cancel := context.WithCancel(context.Background())
	return &Handler{
		ctx:          ctx,
		cancel:       cancel,
		blockState:   blockState,
		epochState:   epochState,
		grandpaState: grandpaState,
		imported:     imported,
		finalised:    finalised,
	}, nil
}

// Start starts the Handler
func (h *Handler) Start() error {
	go h.handleBlockFinalisation(h.ctx)
	return nil
}

// Stop stops the Handler
func (h *Handler) Stop() error {
	h.cancel()
	h.blockState.FreeImportedBlockNotifierChannel(h.imported)
	h.blockState.FreeFinalisedNotifierChannel(h.finalised)
	return nil
}

func (h *Handler) handleBlockFinalisation(ctx context.Context) {
	for {
		select {
		case info := <-h.finalised:
			if info == nil {
				continue
			}

			err := h.epochState.FinalizeBABENextEpochData(&info.Header)
			if err != nil {
				logger.Errorf("failed to persist babe next epoch data: %s", err)
			}

			err = h.epochState.FinalizeBABENextConfigData(&info.Header)
			if err != nil {
				logger.Errorf("failed to persist babe next epoch config: %s", err)
			}

			err = h.grandpaState.ApplyScheduledChanges(&info.Header)
			if err != nil {
				logger.Errorf("failed to apply scheduled change: %s", err)
			}

		case <-ctx.Done():
			return
		}
	}
}
