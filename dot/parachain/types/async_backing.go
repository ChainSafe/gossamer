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

// InboundHRMPLimitations constraints on inbound HRMP channels.
type InboundHRMPLimitations struct {
	// An exhaustive set of all valid watermarks, sorted in ascending order.
	//
	// It's only expected to contain block numbers at which messages were
	// previously sent to a para, excluding most recent head.
	ValidWatermarks []BlockNumber
}

// OutboundHRMPChannelLimitations constraints on outbound HRMP channels.
type OutboundHRMPChannelLimitations struct {
	// The maximum bytes that can be written to the channel.
	BytesRemaining uint32
	// The maximum messages that can be written to the channel.
	MessagesRemaining uint32
}

// Constraints on the actions that can be taken by a new parachain block. These
// limitations are implicitly associated with some particular parachain, which should
// be apparent from usage.
type Constraints struct {
	// The minimum relay-parent number accepted under these constraints.
	MinRelayParentNumber BlockNumber
	// The maximum Proof-of-Validity size allowed, in bytes.
	MaxPoVSize uint32
	// The maximum new validation code size allowed, in bytes.
	MaxCodeSize uint32
	// The amount of UMP messages remaining.
	UMPRemaining uint32
	// The amount of UMP bytes remaining.
	UMPRemainingBytes uint32
	// The maximum number of UMP messages allowed per candidate.
	MaxNumUMPPerCandidate uint32
	// Remaining DMP queue. Only includes sent-at block numbers.
	DMPRemainingMessages []uint32
	// The limitations of all registered inbound HRMP channels.
	HRMPInbound InboundHRMPLimitations
	// The limitations of all registered outbound HRMP channels.
	HRMPChannelsOut map[ParaID]OutboundHRMPChannelLimitations
	// The maximum number of HRMP messages allowed per candidate.
	MaxNumHRMPPerCandidate uint32
	// The required parent head-data of the parachain.
	RequiredParent HeadData
	// The expected validation-code-hash of this parachain.
	ValidationCodeHash ValidationCodeHash
	// The code upgrade restriction signal as-of this parachain.
	UpgradeRestriction *UpgradeRestriction
	// The future validation code hash, if any, and at what relay-parent
	// number the upgrade would be minimally applied.
	FutureValidationCode *FutureValidationCode
}

// FutureValidationCode represents a tuple of BlockNumber and ValidationCodeHash
type FutureValidationCode struct {
	BlockNumber        BlockNumber
	ValidationCodeHash ValidationCodeHash
}

// CandidatePendingAvailability represents informations about one candidate pending availability
type CandidatePendingAvailability struct {
	CandidateHash     CandidateHash
	Descriptor        CandidateDescriptorV2
	Commitments       CandidateCommitments
	RelayParentNumber BlockNumber
	MaxPoVSize        uint32
}

// BackingState holds the state of the backing system per-parachain, including
// state-machine constraints and candidates pending availability
type BackingState struct {
	Constraints         Constraints
	PendingAvailability []CandidatePendingAvailability
}
