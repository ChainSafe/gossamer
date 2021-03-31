// Copyright 2020 ChainSafe Systems (ON) Corp.
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

package grandpa

import (
	"fmt"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/scale"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
)

var (
	grandpaID protocol.ID = "/paritytech/grandpa/1"
	messageID             = network.ConsensusMsgType
)

// Handshake is an alias for network.Handshake
type Handshake = network.Handshake

// Message is an alias for network.Message
type Message = network.Message

// NotificationsMessage is an alias for network.NotificationsMessage
type NotificationsMessage = network.NotificationsMessage

// ConsensusMessage is an alias for network.ConsensusMessage
type ConsensusMessage = network.ConsensusMessage

// GrandpaHandshake is exchanged by nodes that are beginning the grandpa protocol
type GrandpaHandshake struct { //nolint
	Roles byte
}

// SubProtocol returns the grandpa sub-protocol
func (hs *GrandpaHandshake) SubProtocol() string {
	return string(grandpaID)
}

// String formats a BlockAnnounceHandshake as a string
func (hs *GrandpaHandshake) String() string {
	return fmt.Sprintf("GrandpaHandshake Roles=%d", hs.Roles)
}

// Encode encodes a GrandpaHandshake message using SCALE
func (hs *GrandpaHandshake) Encode() ([]byte, error) {
	return scale.Encode(hs)
}

// Decode the message into a GrandpaHandshake
func (hs *GrandpaHandshake) Decode(in []byte) error {
	msg, err := scale.Decode(in, hs)
	if err != nil {
		return err
	}

	hs.Roles = msg.(*GrandpaHandshake).Roles
	return nil
}

// Type ...
func (hs *GrandpaHandshake) Type() byte {
	return 0
}

// Hash ...
func (hs *GrandpaHandshake) Hash() common.Hash {
	return common.Hash{}
}

// IsHandshake returns true
func (hs *GrandpaHandshake) IsHandshake() bool {
	return true
}

func (s *Service) registerProtocol() error {
	return s.network.RegisterNotificationsProtocol(grandpaID,
		messageID,
		s.getHandshake,
		s.decodeHandshake,
		s.validateHandshake,
		s.decodeMessage,
		s.handleNetworkMessage,
		true,
	)
}

func (s *Service) getHandshake() (Handshake, error) {
	return &GrandpaHandshake{
		Roles: 1, // TODO: don't hard-code this
	}, nil
}

func (s *Service) decodeHandshake(in []byte) (Handshake, error) {
	hs := new(GrandpaHandshake)
	err := hs.Decode(in)
	return hs, err
}

func (s *Service) validateHandshake(_ peer.ID, _ Handshake) error {
	return nil
}

func (s *Service) decodeMessage(in []byte) (NotificationsMessage, error) {
	msg := new(network.ConsensusMessage)
	err := msg.Decode(in)
	return msg, err
}

func (s *Service) handleNetworkMessage(_ peer.ID, msg NotificationsMessage) error {
	cm, ok := msg.(*network.ConsensusMessage)
	if !ok {
		return ErrInvalidMessageType
	}

	resp, err := s.messageHandler.handleMessage(cm)
	if err != nil {
		return err
	}

	if resp != nil {
		s.network.SendMessage(resp)
	}

	return nil
}
