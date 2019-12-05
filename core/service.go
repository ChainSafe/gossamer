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

package core

import (
	"fmt"

	"github.com/ChainSafe/gossamer/common"
	"github.com/ChainSafe/gossamer/common/optional"
	"github.com/ChainSafe/gossamer/common/transaction"
	"github.com/ChainSafe/gossamer/consensus/babe"
	"github.com/ChainSafe/gossamer/core/types"
	"github.com/ChainSafe/gossamer/internal/services"
	"github.com/ChainSafe/gossamer/keystore"
	"github.com/ChainSafe/gossamer/p2p"
	"github.com/ChainSafe/gossamer/runtime"
	log "github.com/ChainSafe/log15"
)

var _ services.Service = &Service{}

// Service is an overhead layer that allows communication between the runtime,
// BABE session, and p2p service. It deals with the validation of transactions
// and blocks by calling their respective validation functions in the runtime.
type Service struct {
	rt      *runtime.Runtime
	bs      *babe.Session
	bsRec   <-chan types.Block // receive blocks from BABE session
	p2pRec  <-chan p2p.Message // receive messages from p2p service
	p2pSend chan<- p2p.Message // send messages to p2p service
}

type ServiceConfig struct {
	Keystore *keystore.Keystore
	Runtime  *runtime.Runtime
	BsChan   chan types.Block   // send and receive blocks from BABE session
	P2pRec   <-chan p2p.Message // receive messages from p2p service
	P2pSend  chan<- p2p.Message // send messages to p2p service
}

// NewService returns a new core service that connects the runtime, BABE
// session, and p2p service.
func NewService(cfg *ServiceConfig) (*Service, error) {
	bsConfig := &babe.SessionConfig{
		Keystore:  cfg.Keystore,
		Runtime:   cfg.Runtime,
		BlockSend: cfg.BsChan, // BsChan becomes blockSend in BABE session
	}

	// create a new BABE session
	bs, err := babe.NewSession(bsConfig)
	if err != nil {
		return nil, err
	}

	return &Service{
		rt:      cfg.Runtime,
		bs:      bs,
		bsRec:   cfg.BsChan, // BsChan becomes bsRec in core service
		p2pRec:  cfg.P2pRec,
		p2pSend: cfg.P2pSend,
	}, nil
}

// Start starts the core service
func (s *Service) Start() error {

	// start receiving blocks from BABE session
	go s.startReceivingBlocks()

	// start receiving messages from p2p service
	go s.startReceivingMessages()

	return nil
}

// Stop stops the core service
func (s *Service) Stop() error {

	// stop runtime
	if s.rt != nil {
		s.rt.Stop()
	}

	// close p2pSend channel
	if s.p2pSend != nil {
		close(s.p2pSend)
	}

	return nil
}

// StorageRoot returns the hash of the runtime storage root
func (s *Service) StorageRoot() (common.Hash, error) {
	return s.rt.StorageRoot()
}

// startReceivingBlocks starts receiving blocks from the BABE session
func (s *Service) startReceivingBlocks() {
	for {
		block, ok := <-s.bsRec
		if !ok {
			log.Error("Failed to receive block from BABE session")
			return // exit
		}
		err := s.handleBlock(block)
		if err != nil {
			log.Error("Failed to handle block from BABE session", "err", err)
		}
	}
}

// startReceivingMessages starts receiving messages from the p2p service
func (s *Service) startReceivingMessages() {
	for {
		msg, ok := <-s.p2pRec
		if !ok {
			log.Error("Failed to receive message from p2p service")
			return // exit
		}
		err := s.handleMessage(msg)
		if err != nil {
			log.Error("Failed to handle message from p2p service", "err", err)
		}
	}
}

// handleMessage handles blocks from the BABE session
func (s *Service) handleBlock(block types.Block) (err error) {
	msg := &p2p.BlockAnnounceMessage{
		ParentHash:     block.Header.ParentHash,
		Number:         block.Header.Number,
		StateRoot:      block.Header.StateRoot,
		ExtrinsicsRoot: block.Header.ExtrinsicsRoot,
		Digest:         block.Header.Digest,
	}

	// send block announce message to p2p service
	s.p2pSend <- msg

	return nil
}

// handleMessage handles messages from the p2p service
func (s *Service) handleMessage(msg p2p.Message) (err error) {
	msgType := msg.GetType()

	switch msgType {
	case p2p.BlockAnnounceMsgType:
		err = s.ProcessBlockAnnounceMessage(msg)
	case p2p.BlockResponseMsgType:
		err = s.ProcessBlockResponseMessage(msg)
	case p2p.TransactionMsgType:
		err = s.ProcessTransactionMessage(msg)
	default:
		err = fmt.Errorf("Received unsupported message type")
	}

	return err
}

// ProcessBlockAnnounceMessage creates a block request message from the block
// announce messages (block announce messages include the header but the full
// block is required to execute `core_execute_block`).
func (s *Service) ProcessBlockAnnounceMessage(msg p2p.Message) error {

	// TODO: check if we need to send block request message

	// TODO: update message properties and use generated id
	blockRequest := &p2p.BlockRequestMessage{
		ID:            1,
		RequestedData: 2,
		StartingBlock: []byte{},
		EndBlockHash:  optional.NewHash(true, common.Hash{}),
		Direction:     1,
		Max:           optional.NewUint32(false, 0),
	}

	// send block request message to p2p service
	s.p2pSend <- blockRequest

	return nil
}

// ProcessBlockResponseMessage attempts to validate and add the block to the
// chain by calling `core_execute_block`. Valid blocks are stored in the block
// database to become part of the canonical chain.
func (s *Service) ProcessBlockResponseMessage(msg p2p.Message) error {
	block := msg.(*p2p.BlockResponseMessage).Data

	err := s.validateBlock(block)
	if err != nil {
		log.Error("Failed to validate block", "err", err)
		return err
	}

	return nil
}

// ProcessTransactionMessage validates each transaction in the message and
// adds valid transactions to the transaction queue of the BABE session
func (s *Service) ProcessTransactionMessage(msg p2p.Message) error {

	// get transactions from message extrinsics
	txs := msg.(*p2p.TransactionMessage).Extrinsics

	for _, tx := range txs {
		tx := tx // pin

		// validate each transaction
		val, err := s.validateTransaction(tx)
		if err != nil {
			log.Error("Failed to validate transaction", "err", err)
			return err // exit
		}

		// create new valid transaction
		vtx := transaction.NewValidTransaction(&tx, val)

		// push to the transaction queue of BABE session
		s.bs.PushToTxQueue(vtx)
	}

	return nil
}
