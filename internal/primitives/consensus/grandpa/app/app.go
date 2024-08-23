// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package app

import (
	"fmt"

	"github.com/ChainSafe/gossamer/internal/primitives/core/crypto"
	"github.com/ChainSafe/gossamer/internal/primitives/core/ed25519"
)

// Public key used in grandpa
type Public = ed25519.Public

var _ crypto.Public[Signature] = Public{}

// NewPublic is constructor for Public
func NewPublic(data []byte) (Public, error) {
	if len(data) != 32 {
		return Public{}, fmt.Errorf("invalid public key from data: %v", data)
	}
	pub := Public{}
	copy(pub[:], data)
	return pub, nil
}

// Signature is signature type used in grandpa
type Signature = ed25519.Signature
