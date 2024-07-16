// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"fmt"

	"github.com/ChainSafe/gossamer/dot/network/messages"
	"github.com/ChainSafe/gossamer/lib/common"
)

type MessageType byte

// Message types for notifications protocol messages. Used internally to map message to protocol.
const (
	blockAnnounceMsgType MessageType = iota + 3
	transactionMsgType
	ConsensusMsgType
)

// NotificationsMessage must be implemented by all messages sent over a notifications protocol
type NotificationsMessage interface {
	messages.P2PMessage
	Type() MessageType
	Hash() (common.Hash, error)
}

var _ NotificationsMessage = &ConsensusMessage{}

// ConsensusMessage is mostly opaque to us
type ConsensusMessage struct {
	Data []byte
}

// Type returns ConsensusMsgType
func (*ConsensusMessage) Type() MessageType {
	return ConsensusMsgType
}

// String is the string
func (cm *ConsensusMessage) String() string {
	return fmt.Sprintf("ConsensusMessage Data=%x", cm.Data)
}

// Encode encodes a block response message using SCALE
func (cm *ConsensusMessage) Encode() ([]byte, error) {
	return cm.Data, nil
}

// Decode the message into a ConsensusMessage
func (cm *ConsensusMessage) Decode(in []byte) error {
	cm.Data = in
	return nil
}

// Hash returns the Hash of ConsensusMessage
func (cm *ConsensusMessage) Hash() (common.Hash, error) {
	// scale encode each extrinsic
	encMsg, err := cm.Encode()
	if err != nil {
		return common.Hash{}, fmt.Errorf("cannot encode message: %w", err)
	}
	return common.Blake2bHash(encMsg)
}
