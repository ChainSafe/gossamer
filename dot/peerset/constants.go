// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package peerset

import "math"

const (
	CostMinor         Reputation = -100000
	CostMajor         Reputation = -300000
	CostMinorRepeated Reputation = -200000
	CostMajorRepeated Reputation = -600000
	Malicious         Reputation = math.MinInt32
	BenefitMajorFirst Reputation = 300000
	BenefitMajor      Reputation = 200000
	BenefitMinorFirst Reputation = 15000
	BenefitMinor      Reputation = 10000
)

// ReputationChange value and reason
const (

	// BadMessageValue used when fail to decode message.
	BadMessageValue Reputation = -(1 << 12)
	// BadMessageReason used when fail to decode message.
	BadMessageReason = "Bad message"

	// BadProtocolValue used when a peer is on unsupported protocol version.
	BadProtocolValue Reputation = Malicious
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
	GenesisMismatch Reputation = Malicious
	// GenesisMismatchReason used when a peer has a different genesis
	GenesisMismatchReason = "Genesis mismatch"

	// SameBlockSyncRequest used when a peer send us more than the max number of the same request.
	SameBlockSyncRequest       Reputation = math.MinInt32
	SameBlockSyncRequestReason            = "same block sync request"

	// BenefitNotifyGoodValue is used when a collator was noted good by another subsystem
	BenefitNotifyGoodValue Reputation = BenefitMinor
	// BenefitNotifyGoodReason is used when a collator was noted good by another subsystem
	BenefitNotifyGoodReason = "A collator was noted good by another subsystem"

	// UnexpectedMessageValue is used when validator side of the collator protocol receives an unexpected message
	UnexpectedMessageValue Reputation = CostMinor
	// UnexpectedMessageReason is used when validator side of the collator protocol receives an unexpected message
	UnexpectedMessageReason = "An unexpected message"

	// CurruptedMessageValue is used when message could not be decoded properly.
	CurruptedMessageValue = CostMinor
	// CurruptedMessageReason is used when message could not be decoded properly.
	CurruptedMessageReason = "Message was corrupt"

	// NetworkErrorValue is used when network errors that originated at the remote host should have same cost as timeout.
	NetworkErrorValue = CostMinor
	// NetworkErrorReason is used when network errors that originated at the remote host should have same cost as timeout.
	NetworkErrorReason = "Some network error"

	// InvalidSignatureValue is used when signature of the network message is invalid.
	InvalidSignatureValue Reputation = Malicious
	// InvalidSignatureReason is used when signature of the network message is invalid.
	InvalidSignatureReason = "Invalid network message signature"

	// ReportBadCollatorValue is used when a collator was reported to be bad by another subsystem
	ReportBadCollatorValue Reputation = Malicious
	// ReportBadCollatorReason is used when a collator was reported to be bad by another subsystem
	ReportBadCollatorReason = "A collator was reported by another subsystem"

	// WrongParaValue is used when a collator provided a collation for the wrong para
	WrongParaValue Reputation = Malicious
	// WrongParaReason is used when a collator provided a collation for the wrong para
	WrongParaReason = "A collator provided a collation for the wrong para"

	// UnneededCollatorValue is used when an unneeded collator connected
	UnneededCollatorValue = CostMinor
	// UnneededCollatorReason is used when an unneeded collator connected
	UnneededCollatorReason = "An unneeded collator connected"
)
