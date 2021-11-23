// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"errors"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

var (
	_ NotificationsMessage = &TransactionMessage{}
	_ NotificationsMessage = &transactionHandshake{}
)

// TransactionMessage is a network message that is sent to notify of new transactions entering the network
type TransactionMessage struct {
	Extrinsics []types.Extrinsic
}

// SubProtocol returns the transactions sub-protocol
func (*TransactionMessage) SubProtocol() string {
	return transactionsID
}

// Type returns TransactionMsgType
func (*TransactionMessage) Type() byte {
	return TransactionMsgType
}

// String returns the TransactionMessage extrinsics
func (tm *TransactionMessage) String() string {
	return fmt.Sprintf("TransactionMessage extrinsics count=%d", len(tm.Extrinsics))
}

// Encode will encode TransactionMessage using scale.Encode
func (tm *TransactionMessage) Encode() ([]byte, error) {
	return scale.Marshal(tm.Extrinsics)
}

// Decode the message into a TransactionMessage
func (tm *TransactionMessage) Decode(in []byte) error {
	return scale.Unmarshal(in, &tm.Extrinsics)
}

// Hash returns the hash of the TransactionMessage
func (tm *TransactionMessage) Hash() common.Hash {
	encMsg, _ := tm.Encode()
	hash, _ := common.Blake2bHash(encMsg)
	return hash
}

// IsHandshake returns false
func (*TransactionMessage) IsHandshake() bool {
	return false
}

type transactionHandshake struct{}

// SubProtocol returns the transactions sub-protocol
func (*transactionHandshake) SubProtocol() string {
	return transactionsID
}

// String formats a transactionHandshake as a string
func (*transactionHandshake) String() string {
	return "transactionHandshake"
}

// Encode encodes a transactionHandshake message using SCALE
func (*transactionHandshake) Encode() ([]byte, error) {
	return []byte{}, nil
}

// Decode the message into a transactionHandshake
func (*transactionHandshake) Decode(_ []byte) error {
	return nil
}

// Type ...
func (*transactionHandshake) Type() byte {
	return 1
}

// Hash ...
func (*transactionHandshake) Hash() common.Hash {
	return common.Hash{}
}

// IsHandshake returns true
func (*transactionHandshake) IsHandshake() bool {
	return true
}

func (*Service) getTransactionHandshake() (Handshake, error) {
	return &transactionHandshake{}, nil
}

func decodeTransactionHandshake(_ []byte) (Handshake, error) {
	return &transactionHandshake{}, nil
}

func (s *Service) createBatchMessageHandler(txnBatchCh chan *BatchMessage) NotificationsMessageBatchHandler {
	go func() {
		protocolID := s.host.protocolID + transactionsID
		ticker := time.NewTicker(s.cfg.SlotDuration)
		defer ticker.Stop()

		for {
			select {
			case <-s.ctx.Done():
				return
			case <-ticker.C:
				timeOut := time.NewTimer(s.cfg.SlotDuration / 3)
				var completed bool
				for !completed {
					select {
					case <-timeOut.C:
						completed = true
					case txnMsg := <-txnBatchCh:
						propagate, err := s.handleTransactionMessage(txnMsg.peer, txnMsg.msg)
						if err != nil {
							s.host.closeProtocolStream(protocolID, txnMsg.peer)
							continue
						}

						if s.noGossip || !propagate {
							continue
						}

						if !s.gossip.hasSeen(txnMsg.msg) {
							s.broadcastExcluding(s.notificationsProtocols[TransactionMsgType], txnMsg.peer, txnMsg.msg)
						}
					}
				}
			}
		}
	}()

	return func(peer peer.ID, msg NotificationsMessage) {
		data := &BatchMessage{
			msg:  msg,
			peer: peer,
		}

		timeOut := time.NewTimer(time.Millisecond * 200)

		select {
		case txnBatchCh <- data:
			if !timeOut.Stop() {
				<-timeOut.C
			}
		case <-timeOut.C:
			logger.Debugf("transaction message not included into batch", "peer", peer.String(), "msg", msg.String())
		}
	}
}

func validateTransactionHandshake(_ peer.ID, _ Handshake) error {
	return nil
}

func decodeTransactionMessage(in []byte) (NotificationsMessage, error) {
	msg := new(TransactionMessage)
	err := msg.Decode(in)
	return msg, err
}

func (s *Service) handleTransactionMessage(peerID peer.ID, msg NotificationsMessage) (bool, error) {
	txMsg, ok := msg.(*TransactionMessage)
	if !ok {
		return false, errors.New("invalid transaction type")
	}

	return s.transactionHandler.HandleTransactionMessage(peerID, txMsg)
}
