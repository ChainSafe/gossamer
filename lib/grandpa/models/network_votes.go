// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package models

import "github.com/libp2p/go-libp2p-core/peer"

// NetworkVoteMessage contains a vote message and the peer id
// this vote is originating from.
type NetworkVoteMessage struct {
	From peer.ID
	Msg  *VoteMessage
}
