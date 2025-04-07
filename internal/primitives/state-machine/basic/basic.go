// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package basic

import (
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/externalities"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/internal/primitives/state-machine/overlayedchanges"
)

type BasicExternalities struct {
	overlay    overlayedchanges.OverlayedChanges[hash.H256, runtime.BlakeTwo256]
	extensions externalities.Extensions
}

func NewBasicExternalities() *BasicExternalities {
	return &BasicExternalities{
		overlay:    *overlayedchanges.NewOverlayedChanges[hash.H256, runtime.BlakeTwo256](),
		extensions: externalities.NewExtensions(),
	}
}
