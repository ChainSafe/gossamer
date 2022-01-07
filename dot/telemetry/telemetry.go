// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package telemetry

// telemetry message types
const (
	afgAuthoritySetMsg        = "afg.authority_set"
	afgFinalizedBlocksUpToMsg = "afg.finalized_blocks_up_to"
	afgReceivedCommitMsg      = "afg.received_commit"
	afgReceivedPrecommitMsg   = "afg.received_precommit"
	afgReceivedPrevoteMsg     = "afg.received_prevote"

	blockImportMsg = "block.import"

	notifyFinalizedMsg = "notify.finalized"

	preparedBlockForProposingMsg = "prepared_block_for_proposing"

	systemConnectedMsg = "system.connected"
	systemIntervalMsg  = "system.interval"

	txPoolImportMsg = "txpool.import"
)

// Client is the interface required by send messages to telemetry servers
type Client interface {
	SendMessage(msg Message)
}

// Message interface for Message functions
type Message interface {
	messageType() string
}
