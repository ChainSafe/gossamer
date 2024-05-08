// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"github.com/ChainSafe/gossamer/internal/log"
	primitives "github.com/ChainSafe/gossamer/internal/primitives/consensus/grandpa"
)

var logger = log.NewFromGlobal(log.AddContext("consensus", "grandpa"))

// newAuthoritySet A new authority set along with the canonical block it changed at.
type newAuthoritySet[H, N any] struct {
	CanonNumber N
	CanonHash   H
	SetId       primitives.SetID
	Authorities primitives.AuthorityList
}
