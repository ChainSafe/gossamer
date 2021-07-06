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
	"bytes"
	"fmt"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/scale"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
)

var (
	grandpaID                protocol.ID = "/paritytech/grandpa/1"
	messageID                            = network.ConsensusMsgType
	neighbourMessageInterval             = time.Minute * 5
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
	var roles byte

	if s.authority {
		roles = 4
	} else {
		roles = 1
	}

	return &GrandpaHandshake{
		Roles: roles,
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

func (s *Service) handleNetworkMessage(from peer.ID, msg NotificationsMessage) (bool, error) {
	if msg == nil {
		return false, nil
	}

	cm, ok := msg.(*network.ConsensusMessage)
	if !ok {
		return false, ErrInvalidMessageType
	}

	if len(cm.Data) < 2 {
		return false, nil
	}

	m, err := decodeMessage(cm)
	if err != nil {
		return false, err
	}

	resp, err := s.messageHandler.handleMessage(from, m)
	if err != nil {
		return false, err
	}

	switch r := resp.(type) {
	case *ConsensusMessage:
		if r != nil {
			s.network.SendMessage(resp)
		}
	case nil:
	default:
		logger.Warn("unexpected type returned from message handler", "response", resp)
	}

	if m.Type() == neighbourType || m.Type() == catchUpResponseType {
		return false, nil
	}

	return true, nil
}

// sendMessage sends a vote message to be gossiped to the network
func (s *Service) sendMessage(msg GrandpaMessage) error {
	cm, err := msg.ToConsensusMessage()
	if err != nil {
		return err
	}

	s.network.SendMessage(cm)
	logger.Trace("sent message", "msg", msg)
	return nil
}

func (s *Service) sendNeighbourMessage() {
	t := time.NewTicker(neighbourMessageInterval)
	defer t.Stop()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-t.C:
			if s.neighbourMessage == nil {
				continue
			}
		case info, ok := <-s.finalisedCh:
			if !ok {
				// channel was closed
				return
			}

			s.neighbourMessage = &NeighbourMessage{
				Version: 1,
				Round:   info.Round,
				SetID:   info.SetID,
				Number:  uint32(info.Header.Number.Int64()),
			}
		}

		cm, err := s.neighbourMessage.ToConsensusMessage()
		if err != nil {
			logger.Warn("failed to convert NeighbourMessage to network message", "error", err)
			continue
		}

		s.network.SendMessage(cm)
	}
}

// decodeMessage decodes a network-level consensus message into a GRANDPA VoteMessage or CommitMessage
func decodeMessage(msg *ConsensusMessage) (m GrandpaMessage, err error) {
	var (
		mi interface{}
		ok bool
	)

	switch msg.Data[0] {
	case voteType:
		r := &bytes.Buffer{}
		_, _ = r.Write(msg.Data[1:])
		vm := &VoteMessage{}
		err = vm.Decode(r)
		m = vm
		logger.Trace("got VoteMessage!!!", "msg", m)
	case commitType:
		r := &bytes.Buffer{}
		_, _ = r.Write(msg.Data[1:])
		cm := &CommitMessage{}
		err = cm.Decode(r)
		m = cm
		logger.Trace("got CommitMessage!!!", "msg", m)
	case neighbourType:
		mi, err = scale.Decode(msg.Data[1:], &NeighbourMessage{})
		if m, ok = mi.(*NeighbourMessage); !ok {
			return nil, ErrInvalidMessageType
		}
	case catchUpRequestType:
		mi, err = scale.Decode(msg.Data[1:], &catchUpRequest{})
		if m, ok = mi.(*catchUpRequest); !ok {
			return nil, ErrInvalidMessageType
		}
	case catchUpResponseType:
		mi, err = scale.Decode(msg.Data[1:], &catchUpResponse{})
		if m, ok = mi.(*catchUpResponse); !ok {
			return nil, ErrInvalidMessageType
		}
	default:
		return nil, ErrInvalidMessageType
	}

	if err != nil {
		return nil, err
	}

	return m, nil
}
