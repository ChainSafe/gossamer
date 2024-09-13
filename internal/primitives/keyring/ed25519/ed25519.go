// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package ed25519

import (
	"fmt"

	"github.com/ChainSafe/gossamer/internal/primitives/core/ed25519"
)

type Keyring uint

const (
	Alice Keyring = iota
	Bob
	Charlie
	Dave
	Eve
	Ferdie
	One
	Two
)

func (k Keyring) Sign(msg []byte) ed25519.Signature {
	return k.Pair().Sign(msg)
}

func (k Keyring) Pair() ed25519.Pair {
	pair, err := ed25519.NewPairFromString(fmt.Sprintf("//%s", k), nil)
	if err != nil {
		panic("static values are known good; qed")
	}
	return pair.(ed25519.Pair)
}

func (k Keyring) String() string {
	switch k {
	case Alice:
		return "Alice"
	case Bob:
		return "Bob"
	case Charlie:
		return "Charlie"
	case Dave:
		return "Dave"
	case Eve:
		return "Eve"
	case Ferdie:
		return "Ferdie"
	case One:
		return "One"
	case Two:
		return "Two"
	default:
		panic("unsupported Keyring")
	}
}
