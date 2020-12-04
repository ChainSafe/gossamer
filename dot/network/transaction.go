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
	"io"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/scale"

	"github.com/libp2p/go-libp2p-core/peer"
)

type transactionHandshake struct {
	Roles byte
}

// String formats a transactionHandshake as a string
func (hs *transactionHandshake) String() string {
	return fmt.Sprintf("transactionHandshake Roles=%d",
		hs.Roles)
}

// Encode encodes a transactionHandshake message using SCALE
func (hs *transactionHandshake) Encode() ([]byte, error) {
	return scale.Encode(hs)
}

// Decode the message into a transactionHandshake
func (hs *transactionHandshake) Decode(r io.Reader) error {
	sd := scale.Decoder{Reader: r}
	_, err := sd.Decode(hs)
	return err
}

// Type ...
func (hs *transactionHandshake) Type() byte {
	return 0
}

// IDString ...
func (hs *transactionHandshake) IDString() string {
	return ""
}

// IsHandshake returns true
func (hs *transactionHandshake) IsHandshake() bool {
	return true
}

func (s *Service) getTransactionHandshake() (Handshake, error) {
	return &transactionHandshake{
		Roles: s.cfg.Roles,
	}, nil
}

func decodeTransactionHandshake(r io.Reader) (Handshake, error) {
	roles, err := common.ReadByte(r)
	if err != nil {
		return nil, err
	}

	return &transactionHandshake{
		Roles: roles,
	}, nil
}

func validateTransactionHandshake(_ Handshake) error {
	return nil
}

func decodeTransactionMessage(r io.Reader) (Message, error) {
	msg := new(TransactionMessage)
	err := msg.Decode(r)
	return msg, err
}

func (s *Service) handleTransactionMessage(_ peer.ID, msg Message) error {
	txMsg, ok := msg.(*TransactionMessage)
	if !ok {
		return errors.New("invalid transaction type")
	}

	return s.transactionHandler.HandleTransactionMessage(txMsg)
}
