// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package peerset

import "math"

// ReputationChange value and reason
const (

	// BadMessageValue used when fail to decode message.
	BadMessageValue Reputation = -(1 << 12)
	// BadMessageReason used when fail to decode message.
	BadMessageReason = "Bad message"

	// BadProtocolValue used when a peer is on unsupported protocol version.
	BadProtocolValue Reputation = math.MinInt32
	// BadProtocolReason used when a peer is on unsupported protocol version.
	BadProtocolReason = "Unsupported protocol"

	// TimeOutValue used when a peer doesn't respond in time to our messages.
	TimeOutValue Reputation = -(1 << 10)
	// TimeOutReason used when a peer doesn't respond in time to our messages.
	TimeOutReason = "Request timeout"

	// GossipSuccessValue used when a peer successfully sends a gossip messages.
	GossipSuccessValue Reputation = 1 << 4
	// GossipSuccessReason used when a peer successfully sends a gossip messages.
	GossipSuccessReason = "Successful gossip"

	// DuplicateGossipValue used when a peer sends us a gossip message that we already knew about.
	DuplicateGossipValue Reputation = -(1 << 2)
	// DuplicateGossipReason used when a peer send duplicate gossip message.
	DuplicateGossipReason = "Duplicate gossip"

	// GoodTransactionValue is the used for good transaction.
	GoodTransactionValue Reputation = 1 << 7
	// GoodTransactionReason is the reason for used for good transaction.
	GoodTransactionReason = "Good Transaction"

	// BadTransactionValue used when transaction import was not performed.
	BadTransactionValue Reputation = -(1 << 12)
	// BadTransactionReason when transaction import was not performed.
	BadTransactionReason = "Bad Transaction"

	// BadBlockAnnouncementValue is used when peer announces invalid block.
	BadBlockAnnouncementValue Reputation = -(1 << 12)
	// BadBlockAnnouncementReason is used when peer announces invalid block.
	BadBlockAnnouncementReason = "Bad block announcement"

	// IncompleteHeaderValue  is used when peer sends block with invalid header.
	IncompleteHeaderValue Reputation = -(1 << 20)
	// IncompleteHeaderReason is used when peer sends block with invalid header.
	IncompleteHeaderReason = "Incomplete header"

	// BannedThresholdValue used when we need to ban peer.
	BannedThresholdValue Reputation = 82 * (math.MinInt32 / 100)
	// BannedReason used when we need to ban peer.
	BannedReason = "Banned"

	// BadJustificationValue is used when peer send invalid justification.
	BadJustificationValue Reputation = -(1 << 16)
	// BadJustificationReason is used when peer send invalid justification.
	BadJustificationReason = "Bad justification"

	// GenesisMismatch is used when peer has a different genesis
	GenesisMismatch Reputation = math.MinInt32
	// GenesisMismatchReason used when a peer has a different genesis
	GenesisMismatchReason = "Genesis mismatch"

	// SameBlockSyncRequest used when a peer send us more than the max number of the same request.
	SameBlockSyncRequest       Reputation = math.MinInt32
	SameBlockSyncRequestReason            = "same block sync request"
)
