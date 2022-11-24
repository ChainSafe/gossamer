// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package types

// Represents an equivocation proof. An equivocation happens when a validator
// produces more than one block on the same slot. The proof of equivocation
// are the given distinct headers that were signed by the validator and which
// include the slot number.
type BabeEquivocationProof struct {
	// The public key of the equivocator.
	Offender AuthorityId
	// The slot at which the equivocation happened.
	Slot uint64
	// The first header involved in the equivocation.
	FirstHeader Header
	// The second header involved in the equivocation.
	SecondHeader Header
}

// A Babe authority identifier. Necessarily equivalent to the schnorrkel public key used in
// the main Babe module. If that ever changes, then this must, too.
type AuthorityId [32]byte

type OpaqueKeyOwnershipProof []byte
