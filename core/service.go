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
	"math/big"

	"github.com/ChainSafe/gossamer/common"
	"github.com/ChainSafe/gossamer/common/optional"
	"github.com/ChainSafe/gossamer/common/transaction"
	"github.com/ChainSafe/gossamer/consensus/babe"
	"github.com/ChainSafe/gossamer/core/types"
	"github.com/ChainSafe/gossamer/crypto"
	"github.com/ChainSafe/gossamer/crypto/sr25519"
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
	blockState   BlockState
	storageState StorageState
	rt           *runtime.Runtime
	bs           *babe.Session
	keys         []crypto.Keypair
	blkRec       <-chan types.Block // receive blocks from BABE session
	msgRec       <-chan p2p.Message // receive messages from p2p service
	epochDone    <-chan struct{}    // receive from this channel when BABE epoch changes
	msgSend      chan<- p2p.Message // send messages to p2p service
}

type Config struct {
	BlockState   BlockState
	StorageState StorageState
	Keystore     *keystore.Keystore
	Runtime      *runtime.Runtime
	MsgRec       <-chan p2p.Message
	MsgSend      chan<- p2p.Message
	NewBlocks    chan types.Block // only used for testing purposes
}

// NewService returns a new core service that connects the runtime, BABE
// session, and p2p service.
func NewService(cfg *Config) (*Service, error) {
	if cfg.Keystore == nil {
		return nil, fmt.Errorf("no keystore provided")
	}

	keys := cfg.Keystore.Sr25519Keypairs()

	// no validator keypair found, generate a new one
	if len(keys) == 0 {
		kp, err := sr25519.GenerateKeypair()
		if err != nil {
			return nil, err
		}
		cfg.Keystore.Insert(kp)
		keys = cfg.Keystore.Sr25519Keypairs()
	}

	if cfg.NewBlocks == nil {
		cfg.NewBlocks = make(chan types.Block)
	}

	epochDone := make(chan struct{})

	// BABE session configuration
	bsConfig := &babe.SessionConfig{
		Keypair:        keys[0].(*sr25519.Keypair),
		Runtime:        cfg.Runtime,
		NewBlocks:      cfg.NewBlocks, // becomes block send channel in BABE session
		BlockState:     cfg.BlockState,
		AuthorityIndex: 0, // TODO: where do we get the BABE authority data?
		AuthData:       []*babe.AuthorityData{babe.NewAuthorityData(keys[0].Public().(*sr25519.PublicKey), 1)},
		EpochThreshold: big.NewInt(0),
		Done:           epochDone,
	}

	// create a new BABE session
	bs, err := babe.NewSession(bsConfig)
	if err != nil {
		return nil, err
	}

	// core service
	return &Service{
		rt:           cfg.Runtime,
		bs:           bs,
		keys:         keys,
		blkRec:       cfg.NewBlocks, // becomes block receive channel in core service
		msgRec:       cfg.MsgRec,
		msgSend:      cfg.MsgSend,
		blockState:   cfg.BlockState,
		storageState: cfg.StorageState,
		epochDone:    epochDone,
	}, nil
}

// Start starts the core service
func (s *Service) Start() error {

	// TODO: generate host status message and send to p2p service on startup
	// msgSend <- hostMessage

	// start receiving blocks from BABE session
	go s.receiveBlocks()

	// start receiving messages from p2p service
	go s.receiveMessages()

	// monitor babe session for epoch changes
	go s.handleBabeSession()

	err := s.bs.Start()
	if err != nil {
		log.Error("core could not start BABE", "error", err)
	}

	return err
}

// Stop stops the core service
func (s *Service) Stop() error {

	// stop runtime
	if s.rt != nil {
		s.rt.Stop()
	}

	// close message channel to p2p service
	if s.msgSend != nil {
		close(s.msgSend)
	}

	return nil
}

// StorageRoot returns the hash of the runtime storage root
func (s *Service) StorageRoot() (common.Hash, error) {
	return s.storageState.StorageRoot()
}

func (s *Service) handleBabeSession() {
	for {
		<-s.epochDone
		log.Trace("core: BABE epoch complete, initializing new session")

		newBlocks := make(chan types.Block)
		s.blkRec = newBlocks

		epochDone := make(chan struct{})
		s.epochDone = epochDone

		// BABE session configuration
		bsConfig := &babe.SessionConfig{
			Keypair:        s.keys[0].(*sr25519.Keypair),
			Runtime:        s.rt,
			NewBlocks:      newBlocks, // becomes block send channel in BABE session
			BlockState:     s.blockState,
			AuthorityIndex: 0, // TODO: where do we get the BABE authority data?
			AuthData:       []*babe.AuthorityData{babe.NewAuthorityData(s.keys[0].Public().(*sr25519.PublicKey), 1)},
			EpochThreshold: big.NewInt(0),
			Done:           epochDone,
		}

		// create a new BABE session
		bs, err := babe.NewSession(bsConfig)
		if err != nil {
			log.Error("core could not initialize BABE", "error", err)
			return
		}

		err = bs.Start()
		if err != nil {
			log.Error("core could not start BABE", "error", err)
		}

		s.bs = bs
		log.Trace("core: BABE session initialized and started")
	}
}

// receiveBlocks starts receiving blocks from the BABE session
func (s *Service) receiveBlocks() {
	for {
		// receive block from BABE session
		block, ok := <-s.blkRec
		if !ok {
			// epoch complete
			log.Debug("core: BABE session complete")
		} else {
			err := s.handleReceivedBlock(block)
			if err != nil {
				log.Error("Failed to handle block from BABE session", "err", err)
			}
		}
	}
}

// receiveMessages starts receiving messages from the p2p service
func (s *Service) receiveMessages() {
	for {
		// receive message from p2p service
		msg, ok := <-s.msgRec
		if !ok {
			log.Error("Failed to receive message from p2p service")
			return // exit
		}
		err := s.handleReceivedMessage(msg)
		if err != nil {
			log.Error("Failed to handle message from p2p service", "err", err)
		}
	}
}

// handleReceivedBlock handles blocks from the BABE session
func (s *Service) handleReceivedBlock(block types.Block) (err error) {
	msg := &p2p.BlockAnnounceMessage{
		ParentHash:     block.Header.ParentHash,
		Number:         block.Header.Number,
		StateRoot:      block.Header.StateRoot,
		ExtrinsicsRoot: block.Header.ExtrinsicsRoot,
		Digest:         block.Header.Digest,
	}

	// send block announce message to p2p service
	s.msgSend <- msg

	// TODO: check if host status message needs to be updated based on new block
	// information, if so, generate host status message and send to p2p service

	// TODO: send updated host status message to p2p service
	// s.msgSend <- msg

	return nil
}

// handleReceivedMessage handles messages from the p2p service
func (s *Service) handleReceivedMessage(msg p2p.Message) (err error) {
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

	// TODO: check if we should send block request message

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
	s.msgSend <- blockRequest

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
		vtx := transaction.NewValidTransaction(tx, val)

		// push to the transaction queue of BABE session
		s.bs.PushToTxQueue(vtx)
	}

	return nil
}
