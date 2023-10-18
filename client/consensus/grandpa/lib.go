// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"github.com/ChainSafe/gossamer/internal/log"
	"golang.org/x/exp/constraints"
)

var logger = log.NewFromGlobal(log.AddContext("consensus", "grandpa"))

type AuthorityID any

type AuthoritySignature any

// Authority represents a grandpa authority
type Authority[ID AuthorityID] struct {
	Key    ID
	Weight uint64
}

// NewAuthoritySetStruct A new authority set along with the canonical block it changed at.
type NewAuthoritySetStruct[H comparable, N constraints.Unsigned, ID AuthorityID] struct {
	CanonNumber N
	CanonHash   H
	SetId       N
	Authorities []Authority[ID]
}
