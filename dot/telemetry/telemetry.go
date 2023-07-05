// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package telemetry

import "encoding/json"

// telemetry message types
const (
	afgAuthoritySetMsg                        = "afg.authority_set"
	afgFinalizedBlocksUpToMsg                 = "afg.finalized_blocks_up_to"
	afgReceivedCommitMsg                      = "afg.received_commit"
	afgReceivedPrecommitMsg                   = "afg.received_precommit"
	afgReceivedPrevoteMsg                     = "afg.received_prevote"
	afgApplyingScheduledAuthoritySetChangeMsg = "afg.applying_scheduled_authority_set_change"
	afgApplyingForcedAuthoritySetChangeMsg    = "afg.applying_forced_authority_set_change"

	blockImportMsg = "block.import"

	notifyFinalizedMsg = "notify.finalized"

	preparedBlockForProposingMsg = "prepared_block_for_proposing"

	systemConnectedMsg = "system.connected"
	systemIntervalMsg  = "system.interval"

	txPoolImportMsg = "txpool.import"
)

// Client is the interface to send messages to telemetry servers
type Client interface {
	SendMessage(msg json.Marshaler)
}

// NoopClient used for minimal implementation of the Client interface
type NoopClient struct{}

// SendMessage is an empty implementation used for testing
func (NoopClient) SendMessage(_ json.Marshaler) {}
