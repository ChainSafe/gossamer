// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import "fmt"

var (
	errCannotValidateHandshake       = fmt.Errorf("failed to validate handshake")
	errMessageTypeNotValid           = fmt.Errorf("message type is not valid")
	errInvalidHandshakeForPeer       = fmt.Errorf("peer previously sent invalid handshake")
	errHandshakeTimeout              = fmt.Errorf("handshake timeout reached")
	errBlockRequestFromNumberInvalid = fmt.Errorf("block request message From number is not valid")
	errInvalidStartingBlockType      = fmt.Errorf("invalid StartingBlock in messsage")
	errInboundHanshakeExists         = fmt.Errorf("an inbound handshake already exists for given peer")
	errInvalidRole                   = fmt.Errorf("invalid role")
)
