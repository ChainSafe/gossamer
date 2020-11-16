// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package network

import (
	"errors"
	"io"

	"github.com/libp2p/go-libp2p-core/peer"
)

var errCannotValidateHandshake = errors.New("failed to validate handshake")

// Handshake is the interface all handshakes for notifications protocols must implement
type Handshake interface {
	Message
}

type (
	HandshakeGetter    = func() (Handshake, error)
	HandshakeDecoder   = func(io.Reader) (Handshake, error)
	HandshakeValidator = func(Handshake) error
	MessageDecoder     = func(io.Reader) (Message, error)

	// NotificationsMessageHandler is called when a (non-handshake) message is received over a notifications stream.
	NotificationsMessageHandler = func(peer peer.ID, msg Message) error
)
