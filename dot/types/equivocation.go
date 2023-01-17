// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package types

import "github.com/ChainSafe/gossamer/lib/crypto/sr25519"

// BabeEquivocationProof represents a babe equivocation proof.
// An equivocation happens when a validator produces more than one block on the same slot.
// The proof of equivocation are the given distinct headers that were signed by the validator
// and which include the slot number.
type BabeEquivocationProof struct {
	// Offender is the public key of the equivocator.
	Offender AuthorityID
	// Slot at which the equivocation happened.
	Slot uint64
	// FirstHeader is the first header involved in the equivocation.
	FirstHeader Header
	// SecondHeader is the second header involved in the equivocation.
	SecondHeader Header
}

// AuthorityID represents a babe authority identifier.
type AuthorityID [sr25519.PublicKeyLength]byte

// OpaqueKeyOwnershipProof is an opaque type used to represent the key ownership proof at the
// runtime API boundary. The inner value is an encoded representation of the actual key
// ownership proof which will be parameterized when defining the runtime. At
// the runtime API boundary this type is unknown and as such we keep this
// opaque representation, implementers of the runtime API will have to make
// sure that all usages of `OpaqueKeyOwnershipProof` refer to the same type.
type OpaqueKeyOwnershipProof []byte
