// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachaintypes

// AsyncBackingParams contains the parameters for the async backing.
type AsyncBackingParams struct {
	// The maximum number of para blocks between the para head in a relay parent
	// and a new candidate. Restricts nodes from building arbitrary long chains
	// and spamming other validators.
	//
	// When async backing is disabled, the only valid value is 0.
	MaxCandidateDepth uint32 `scale:"1"`
	// How many ancestors of a relay parent are allowed to build candidates on top
	// of.
	//
	// When async backing is disabled, the only valid value is 0.
	AllowedAncestryLen uint32 `scale:"2"`
}
