// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package peerset

import "errors"

var (
	ErrDisconnectReceivedForNonConnectedPeer = errors.New("received disconnect for non-connected node")

	ErrConfigSetIsEmpty = errors.New("config set is empty")

	ErrPeerDoesNotExist = errors.New("peer doesn't exist")

	ErrPeerDisconnected = errors.New("node is already disconnected")

	ErrOutgoingSlotsUnavailable = errors.New("not enough outgoing slots")

	ErrIncomingSlotsUnavailable = errors.New("not enough incoming slots")
)
