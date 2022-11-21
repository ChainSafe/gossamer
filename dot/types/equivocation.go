// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package types

// /// Represents an equivocation proof. An equivocation happens when a validator
// /// produces more than one block on the same slot. The proof of equivocation
// /// are the given distinct headers that were signed by the validator and which
// /// include the slot number.
// #[derive(Clone, Debug, Decode, Encode, PartialEq, TypeInfo)]
// pub struct EquivocationProof<Header, Id> {
// 	/// Returns the authority id of the equivocator.
// 	pub offender: Id,
// 	/// The slot at which the equivocation happened.
// 	pub slot: Slot,
// 	/// The first header involved in the equivocation.
// 	pub first_header: Header,
// 	/// The second header involved in the equivocation.
// 	pub second_header: Header,
// }

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

// type slot uint64
type AuthorityId [32]byte

// /// A Babe authority identifier. Necessarily equivalent to the schnorrkel public key used in
// /// the main Babe module. If that ever changes, then this must, too.
// pub type AuthorityId = app::Public;

type OpaqueKeyOwnershipProof []byte

// pub struct OpaqueKeyOwnershipProof(Vec<u8>);
