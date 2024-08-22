// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"errors"
)

var (
	ErrNoPeersConnected     = errors.New("no peers connected")
	ErrReceivedEmptyMessage = errors.New("received empty message")

	errCannotValidateHandshake   = errors.New("failed to validate handshake")
	errMessageTypeNotValid       = errors.New("message type is not valid")
	errInvalidHandshakeForPeer   = errors.New("peer previously sent invalid handshake")
	errHandshakeTimeout          = errors.New("handshake timeout reached")
	errInboundHanshakeExists     = errors.New("an inbound handshake already exists for given peer")
	errInvalidRole               = errors.New("invalid role")
	ErrFailedToReadEntireMessage = errors.New("failed to read entire message")
	ErrNilStream                 = errors.New("nil stream")
	ErrInvalidLEB128EncodedData  = errors.New("invalid LEB128 encoded data")
	ErrGreaterThanMaxSize        = errors.New("greater than maximum size")
	ErrStreamReset               = errors.New("stream reset")
)
