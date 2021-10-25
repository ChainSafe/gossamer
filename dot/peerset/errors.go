package peerset

import "errors"

var (
	errDisconnectReceivedForNonConnectedPeer = errors.New("received disconnect for non-connected node")

	errConfigSetIsEmpty = errors.New("config set is empty")

	errPeerDoesNotExist = errors.New("peer doesn't exist")

	errPeerDisconnected = errors.New("node is already disconnected")

	errSlotsUnavailable = errors.New("not enough outgoing slots")
)
