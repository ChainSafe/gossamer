// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package peerset

import "fmt"

var (
	ErrDisconnectReceivedForNonConnectedPeer = fmt.Errorf("received disconnect for non-connected node")

	ErrConfigSetIsEmpty = fmt.Errorf("config set is empty")

	ErrPeerDoesNotExist = fmt.Errorf("peer doesn't exist")

	ErrPeerDisconnected = fmt.Errorf("node is already disconnected")

	ErrOutgoingSlotsUnavailable = fmt.Errorf("not enough outgoing slots")

	ErrIncomingSlotsUnavailable = fmt.Errorf("not enough incoming slots")
)
