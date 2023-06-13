// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"fmt"
	"strings"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

const grandpaID1 = "grandpa/1"

// NotificationsMessage is an alias for network.NotificationsMessage
type NotificationsMessage = network.NotificationsMessage

// ConsensusMessage is an alias for network.ConsensusMessage
type ConsensusMessage = network.ConsensusMessage

// GrandpaHandshake is exchanged by nodes that are beginning the grandpa protocol
type GrandpaHandshake struct {
	Role common.NetworkRole
}

// String formats a BlockAnnounceHandshake as a string
func (hs *GrandpaHandshake) String() string {
	return fmt.Sprintf("GrandpaHandshake NetworkRole=%d", hs.Role)
}

// Encode encodes a GrandpaHandshake message using SCALE
func (hs *GrandpaHandshake) Encode() ([]byte, error) {
	return scale.Marshal(*hs)
}

// Decode the message into a GrandpaHandshake
func (hs *GrandpaHandshake) Decode(in []byte) error {
	return scale.Unmarshal(in, hs)
}

// IsValid return if it is a valid handshake.
func (hs *GrandpaHandshake) IsValid() bool {
	switch hs.Role {
	case common.AuthorityRole, common.FullNodeRole:
		return true
	default:
		return false
	}
}

func (s *Service) registerProtocol() error {
	genesisHash := s.blockState.GenesisHash().String()
	genesisHash = strings.TrimPrefix(genesisHash, "0x")
	grandpaProtocolID := fmt.Sprintf("/%s/%s", genesisHash, grandpaID1)

	return s.network.RegisterNotificationsProtocol(
		protocol.ID(grandpaProtocolID),
		network.ConsensusMsgType,
		s.getHandshake,
		s.decodeHandshake,
		s.validateHandshake,
		s.decodeMessage,
		s.handleNetworkMessage,
		nil,
		network.MaxGrandpaNotificationSize,
	)
}

func (s *Service) getHandshake() (network.Handshake, error) {
	var role common.NetworkRole

	if s.authority {
		role = common.AuthorityRole
	} else {
		role = common.FullNodeRole
	}

	return &GrandpaHandshake{
		Role: role,
	}, nil
}

func (*Service) decodeHandshake(in []byte) (network.Handshake, error) {
	hs := new(GrandpaHandshake)
	err := hs.Decode(in)
	return hs, err
}

func (*Service) validateHandshake(_ peer.ID, _ network.Handshake) error {
	return nil
}

func (*Service) decodeMessage(in []byte) (NotificationsMessage, error) {
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
			s.network.GossipMessage(resp)
		}
	case nil:
	default:
		logger.Warnf(
			"unexpected type %T returned from message handler: %v",
			resp, resp)
	}

	switch m.(type) {
	case *NeighbourPacketV1, *CatchUpResponse:
		return false, nil
	}

	return true, nil
}

// decodeMessage decodes a network-level consensus message into a GRANDPA VoteMessage or CommitMessage
func decodeMessage(cm *network.ConsensusMessage) (m GrandpaMessage, err error) {
	msg := newGrandpaMessage()
	err = scale.Unmarshal(cm.Data, &msg)
	if err != nil {
		return nil, err
	}

	msgValue, err := msg.Value()
	if err != nil {
		return nil, fmt.Errorf("getting message value: %w", err)
	}
	switch val := msgValue.(type) {
	case VoteMessage:
		m = &val
	case CommitMessage:
		m = &val
	case VersionedNeighbourPacket:
		neighbourMessageVal, err := val.Value()
		if err != nil {
			return nil, fmt.Errorf("getting versioned neighbour packet value: %w", err)
		}
		switch NeighbourMessage := neighbourMessageVal.(type) {
		case NeighbourPacketV1:
			m = &NeighbourMessage
		default:
			return nil, fmt.Errorf("%w", ErrInvalidMessageType)
		}
	case CatchUpRequest:
		m = &val
	case CatchUpResponse:
		m = &val
	default:
		return nil, fmt.Errorf("%w", ErrInvalidMessageType)
	}

	return m, nil
}
