// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"errors"
)

var (
	errCannotValidateHandshake       = errors.New("failed to validate handshake")
	errMessageTypeNotValid           = errors.New("message type is not valid")
	errInvalidHandshakeForPeer       = errors.New("peer previously sent invalid handshake")
	errHandshakeTimeout              = errors.New("handshake timeout reached")
	errBlockRequestFromNumberInvalid = errors.New("block request message From number is not valid")
	errInvalidStartingBlockType      = errors.New("invalid StartingBlock in messsage")
	errInboundHanshakeExists         = errors.New("an inbound handshake already exists for given peer")
	errInvalidRole                   = errors.New("invalid role")
	ErrFailedToReadEntireMessage     = errors.New("failed to read entire message")
	ErrNilStream                     = errors.New("nil stream")
	ErrInvalidLEB128EncodedData      = errors.New("invalid LEB128 encoded data")
	ErrGreaterThanMaxSize            = errors.New("greater than maximum size")
)
