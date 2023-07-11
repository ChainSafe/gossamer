// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
)

var logger = log.NewFromGlobal(log.AddContext("consensus", "grandpa"))

// Hash represents a grandpa hash
type Hash [32]byte

// Authority represents a grandpa authority
type Authority struct {
	Key    ed25519.PublicKey
	Weight uint64
}

// AuthorityList A list of Grandpa authorities with associated weights.
type AuthorityList []Authority
