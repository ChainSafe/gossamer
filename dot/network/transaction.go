// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"errors"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

var (
	_ NotificationsMessage = &TransactionMessage{}
	_ Handshake            = (*transactionHandshake)(nil)
)

// txnBatchChTimeout is the timeout for adding a transaction to the batch processing channel
const txnBatchChTimeout = time.Millisecond * 200

// TransactionMessage is a network message that is sent to notify of new transactions entering the network
type TransactionMessage struct {
	Extrinsics []types.Extrinsic
}

// Type returns transactionMsgType
func (*TransactionMessage) Type() MessageType {
	return transactionMsgType
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
func (tm *TransactionMessage) Hash() (common.Hash, error) {
	encMsg, err := tm.Encode()
	if err != nil {
		return common.Hash{}, fmt.Errorf("could not encode message: %w", err)
	}
	return common.Blake2bHash(encMsg)
}

type transactionHandshake struct{}

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

// IsValid returns true
func (*transactionHandshake) IsValid() bool {
	return true
}

func (*Service) getTransactionHandshake() (Handshake, error) {
	return &transactionHandshake{}, nil
}

func decodeTransactionHandshake(_ []byte) (Handshake, error) {
	return &transactionHandshake{}, nil
}

func (s *Service) startTxnBatchProcessing(txnBatchCh chan *batchMessage, slotDuration time.Duration) {
	protocolID := s.host.protocolID + transactionsID
	ticker := time.NewTicker(slotDuration)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			timer := time.NewTimer(slotDuration / 3)
			var timedOut bool
			for !timedOut {
				select {
				case <-timer.C:
					timedOut = true
				case txnMsg := <-txnBatchCh:
					propagate, err := s.handleTransactionMessage(txnMsg.peer, txnMsg.msg)
					if err != nil {
						logger.Warnf("could not handle transaction message: %s", err)
						s.host.closeProtocolStream(protocolID, txnMsg.peer)
						continue
					}

					if s.noGossip || !propagate {
						continue
					}

					// TODO: Check if s.gossip.hasSeen should be moved before handleTransactionMessage. #2445
					// That we could avoid handling the transactions again, which we would have already seen.

					hasSeen, err := s.gossip.hasSeen(txnMsg.msg)
					if err != nil {
						s.host.closeProtocolStream(protocolID, txnMsg.peer)
						logger.Debugf("could not check if message was seen before: %s", err)
						continue
					}
					if !hasSeen {
						s.broadcastExcluding(s.notificationsProtocols[transactionMsgType], txnMsg.peer, txnMsg.msg)
					}
				}
			}
		}
	}
}

func (s *Service) createBatchMessageHandler(txnBatchCh chan *batchMessage) NotificationsMessageBatchHandler {
	go s.startTxnBatchProcessing(txnBatchCh, s.cfg.SlotDuration)

	return func(peer peer.ID, msg NotificationsMessage) {
		data := &batchMessage{
			msg:  msg,
			peer: peer,
		}

		timer := time.NewTimer(txnBatchChTimeout)

		select {
		case txnBatchCh <- data:
			timer.Stop()
		case <-timer.C:
			logger.Debugf("transaction message %s for peer %s not included into batch", msg, peer)
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
