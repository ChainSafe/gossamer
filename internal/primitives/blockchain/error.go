// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package blockchain

import "errors"

var (
	ErrBackend                   = errors.New("backend error")
	ErrUnknownBlock              = errors.New("unknown block")
	ErrNonSequentialFinalization = errors.New("did not finalize blocks in sequential order")
	ErrNotInFinalizedChain       = errors.New("potential long-range attack: block not in finalized chain")
	ErrInvalidState              = errors.New("provided state is invalid")
	ErrBadJustification          = errors.New("bad justification for header")
	ErrStateDatabase             = errors.New("state database error")
	ErrSetHeadTooOld             = errors.New("failed to set the chain head to a block that's too old")
	ErrInvalidStateRoot          = errors.New("calculated state root does not match")
	ErrIncompletePipeline        = errors.New("incomplete block import pipeline")
)
