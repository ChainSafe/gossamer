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
	"fmt"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/scale"

	"github.com/libp2p/go-libp2p-core/peer"
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
func (tm *TransactionMessage) SubProtocol() string {
	return transactionsID
}

// Type returns TransactionMsgType
func (tm *TransactionMessage) Type() byte {
	return TransactionMsgType
}

// String returns the TransactionMessage extrinsics
func (tm *TransactionMessage) String() string {
	return fmt.Sprintf("TransactionMessage extrinsics=%x", tm.Extrinsics)
}

// Encode will encode TransactionMessage using scale.Encode
func (tm *TransactionMessage) Encode() ([]byte, error) {
	// scale encode each extrinsic
	var encodedExtrinsics = make([]byte, 0)
	for _, extrinsic := range tm.Extrinsics {
		encExt, err := scale.Encode([]byte(extrinsic))
		if err != nil {
			return nil, err
		}
		encodedExtrinsics = append(encodedExtrinsics, encExt...)
	}

	// scale encode the set of all extrinsics
	return scale.Encode(encodedExtrinsics)
}

// Decode the message into a TransactionMessage
func (tm *TransactionMessage) Decode(in []byte) error {
	decodedMessage, err := scale.Decode(in, []byte{})
	if err != nil {
		return err
	}
	messageSize := len(decodedMessage.([]byte))
	bytesProcessed := 0
	// loop through the message decoding extrinsics until they have all been decoded
	for bytesProcessed < messageSize {
		decodedExtrinsic, err := scale.Decode(decodedMessage.([]byte)[bytesProcessed:], []byte{})
		if err != nil {
			return err
		}
		bytesProcessed = bytesProcessed + len(decodedExtrinsic.([]byte)) + 1 // add 1 to processed since the first decode byte is consumed during decoding
		tm.Extrinsics = append(tm.Extrinsics, decodedExtrinsic.([]byte))
	}

	return nil
}

// Hash returns the hash of the TransactionMessage
func (tm *TransactionMessage) Hash() common.Hash {
	encMsg, _ := tm.Encode()
	hash, _ := common.Blake2bHash(encMsg)
	return hash
}

// IsHandshake returns false
func (tm *TransactionMessage) IsHandshake() bool {
	return false
}

type transactionHandshake struct{}

// SubProtocol returns the transactions sub-protocol
func (hs *transactionHandshake) SubProtocol() string {
	return transactionsID
}

// String formats a transactionHandshake as a string
func (hs *transactionHandshake) String() string {
	return "transactionHandshake"
}

// Encode encodes a transactionHandshake message using SCALE
func (hs *transactionHandshake) Encode() ([]byte, error) {
	return []byte{}, nil
}

// Decode the message into a transactionHandshake
func (hs *transactionHandshake) Decode(in []byte) error {
	return nil
}

// Type ...
func (hs *transactionHandshake) Type() byte {
	return 1
}

// Hash ...
func (hs *transactionHandshake) Hash() common.Hash {
	return common.Hash{}
}

// IsHandshake returns true
func (hs *transactionHandshake) IsHandshake() bool {
	return true
}

func (s *Service) getTransactionHandshake() (Handshake, error) {
	return &transactionHandshake{}, nil
}

func decodeTransactionHandshake(in []byte) (Handshake, error) {
	return &transactionHandshake{}, nil
}

func validateTransactionHandshake(_ peer.ID, _ Handshake) error {
	return nil
}

func decodeTransactionMessage(in []byte) (NotificationsMessage, error) {
	msg := new(TransactionMessage)
	err := msg.Decode(in)
	return msg, err
}

func (s *Service) handleTransactionMessage(_ peer.ID, msg NotificationsMessage) error {
	txMsg, ok := msg.(*TransactionMessage)
	if !ok {
		return errors.New("invalid transaction type")
	}

	return s.transactionHandler.HandleTransactionMessage(txMsg)
}
