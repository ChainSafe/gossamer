// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"errors"
)

var (
	errCannotValidateHandshake = errors.New("failed to validate handshake")
	errMessageTypeNotValid     = errors.New("message type is not valid")
	errMessageIsNotHandshake   = errors.New("failed to convert message to Handshake")
	errMissingHandshakeMutex   = errors.New("outboundHandshakeMutex does not exist")
	errInvalidHandshakeForPeer = errors.New("peer previously sent invalid handshake")
	errHandshakeTimeout        = errors.New("handshake timeout reached")
)
