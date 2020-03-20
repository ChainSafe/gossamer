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
	"errors"
	"fmt"
	"math/big"
	"sync"

	"github.com/ChainSafe/gossamer/dot/core/types"
	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/lib/babe"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/services"

	log "github.com/ChainSafe/log15"
)

var _ services.Service = &Service{}

var maxResponseSize = 8 // maximum number of block datas to reply with in a BlockResponse message.

// Service is an overhead layer that allows communication between the runtime,
// BABE session, and network service. It deals with the validation of transactions
// and blocks by calling their respective validation functions in the runtime.
type Service struct {
	// State interfaces
	blockState       BlockState
	storageState     StorageState
	transactionQueue TransactionQueue

	// Current runtime and hash of the current runtime code
	rt       *runtime.Runtime
	codeHash common.Hash

	// Current BABE session
	bs          *babe.Session
	isAuthority bool

	// Keystore
	keys *keystore.Keystore

	// Channels for inter-process communication
	msgRec    <-chan network.Message // receive messages from network service
	msgSend   chan<- network.Message // send messages to network service
	blkRec    <-chan types.Block     // receive blocks from BABE session
	epochDone <-chan struct{}        // receive from this channel when BABE epoch changes
	babeKill  chan<- struct{}        // close this channel to kill current BABE session
	lock      sync.Mutex
	closed    bool

	// Block synchronization
	syncChan chan<- *big.Int
	syncLock *sync.Mutex
	syncer   *Syncer
}

// Config holds the configuration for the core Service.
type Config struct {
	BlockState       BlockState
	StorageState     StorageState
	TransactionQueue TransactionQueue
	Keystore         *keystore.Keystore
	Runtime          *runtime.Runtime
	IsAuthority      bool

	NewBlocks chan types.Block // only used for testing purposes
	MsgRec    <-chan network.Message
	MsgSend   chan<- network.Message
	SyncChan  chan *big.Int
}

// NewService returns a new core service that connects the runtime, BABE
// session, and network service.
func NewService(cfg *Config) (*Service, error) {
	if cfg.Keystore == nil {
		return nil, fmt.Errorf("no keystore provided")
	}

	keys := cfg.Keystore.Sr25519Keypairs()

	if cfg.NewBlocks == nil {
		cfg.NewBlocks = make(chan types.Block)
	}

	if cfg.BlockState == nil {
		return nil, fmt.Errorf("block state is nil")
	}

	if cfg.StorageState == nil {
		return nil, fmt.Errorf("storage state is nil")
	}

	codeHash, err := cfg.StorageState.LoadCodeHash()
	if err != nil {
		return nil, err
	}

	syncerLock := &sync.Mutex{}

	syncerCfg := &SyncerConfig{
		BlockState:    cfg.BlockState,
		BlockNumberIn: cfg.SyncChan,
		MsgOut:        cfg.MsgSend,
		Lock:          syncerLock,
	}

	syncer, err := NewSyncer(syncerCfg)
	if err != nil {
		return nil, err
	}

	var coreSrvc = &Service{}

	if cfg.IsAuthority {
		if cfg.Keystore.NumSr25519Keys() == 0 {
			return nil, fmt.Errorf("no keys provided for authority node")
		}

		epochDone := make(chan struct{})
		babeKill := make(chan struct{})

		coreSrvc = &Service{
			rt:               cfg.Runtime,
			codeHash:         codeHash,
			keys:             cfg.Keystore,
			blkRec:           cfg.NewBlocks, // becomes block receive channel in core service
			msgRec:           cfg.MsgRec,
			msgSend:          cfg.MsgSend,
			blockState:       cfg.BlockState,
			storageState:     cfg.StorageState,
			transactionQueue: cfg.TransactionQueue,
			epochDone:        epochDone,
			babeKill:         babeKill,
			isAuthority:      true,
			closed:           false,
			syncer:           syncer,
			syncLock:         syncerLock,
			syncChan:         cfg.SyncChan,
		}

		// TODO: update grandpaAuthorities runtime method, pass latest block number
		authData, err := coreSrvc.grandpaAuthorities()
		if err != nil {
			return nil, fmt.Errorf("could not retrieve authority data: %s", err)
		}

		// BABE session configuration
		bsConfig := &babe.SessionConfig{
			Keypair:          keys[0].(*sr25519.Keypair),
			Runtime:          cfg.Runtime,
			NewBlocks:        cfg.NewBlocks, // becomes block send channel in BABE session
			BlockState:       cfg.BlockState,
			StorageState:     cfg.StorageState,
			AuthData:         authData,
			Done:             epochDone,
			Kill:             babeKill,
			TransactionQueue: cfg.TransactionQueue,
			SyncLock:         syncerLock,
		}

		// create a new BABE session
		bs, err := babe.NewSession(bsConfig)
		if err != nil {
			coreSrvc.isAuthority = false
			log.Error("[core] could not start babe session", "error", err)
			return coreSrvc, nil
		}

		coreSrvc.bs = bs
	} else {
		coreSrvc = &Service{
			rt:               cfg.Runtime,
			codeHash:         codeHash,
			keys:             cfg.Keystore,
			blkRec:           cfg.NewBlocks, // becomes block receive channel in core service
			msgRec:           cfg.MsgRec,
			msgSend:          cfg.MsgSend,
			blockState:       cfg.BlockState,
			storageState:     cfg.StorageState,
			transactionQueue: cfg.TransactionQueue,
			isAuthority:      false,
			closed:           false,
			syncer:           syncer,
			syncLock:         syncerLock,
			syncChan:         cfg.SyncChan,
		}
	}

	return coreSrvc, nil
}

// Start starts the core service
func (s *Service) Start() error {

	// start receiving blocks from BABE session
	go s.receiveBlocks()

	// start receiving messages from network service
	go s.receiveMessages()

	// start syncer
	s.syncer.Start()

	if s.isAuthority {
		// monitor babe session for epoch changes
		go s.handleBabeSession()

		err := s.bs.Start()
		if err != nil {
			log.Error("[core] could not start BABE", "error", err)
			return err
		}
	}

	return nil
}

// Stop stops the core service
func (s *Service) Stop() error {

	s.lock.Lock()
	defer s.lock.Unlock()

	// close channel to network service and BABE service
	if !s.closed {
		if s.msgSend != nil {
			close(s.msgSend)
		}
		if s.isAuthority {
			close(s.babeKill)
		}
		s.closed = true
	}

	return nil
}

func (s *Service) safeMsgSend(msg network.Message) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.closed {
		return errors.New("service has been stopped")
	}
	s.msgSend <- msg
	return nil
}

func (s *Service) safeBabeKill() error {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.closed {
		return errors.New("service has been stopped")
	}
	close(s.babeKill)
	return nil
}

// receiveBlocks starts receiving blocks from the BABE session
func (s *Service) receiveBlocks() {
	for {
		// receive block from BABE session
		block, ok := <-s.blkRec
		if ok {
			err := s.handleReceivedBlock(&block)
			if err != nil {
				log.Error("[core] failed to handle block from BABE session", "err", err)
			}
		}
	}
}

// receiveMessages starts receiving messages from the network service
func (s *Service) receiveMessages() {
	for {
		// receive message from network service
		msg, ok := <-s.msgRec
		if !ok {
			log.Error("[core] failed to receive message from network service")
			return // exit
		}

		err := s.handleReceivedMessage(msg)
		if err != nil {
			log.Error("[core] failed to handle message from network service", "err", err)
		}
	}
}

// handleReceivedBlock handles blocks from the BABE session
func (s *Service) handleReceivedBlock(block *types.Block) (err error) {
	if s.blockState == nil {
		return fmt.Errorf("blockState is nil")
	}

	err = s.blockState.AddBlock(block)
	if err != nil {
		return err
	}

	msg := &network.BlockAnnounceMessage{
		ParentHash:     block.Header.ParentHash,
		Number:         block.Header.Number,
		StateRoot:      block.Header.StateRoot,
		ExtrinsicsRoot: block.Header.ExtrinsicsRoot,
		Digest:         block.Header.Digest,
	}

	err = s.safeMsgSend(msg)
	if err != nil {
		return err
	}

	err = s.checkForRuntimeChanges()
	if err != nil {
		return err
	}

	return nil
}

// handleReceivedMessage handles messages from the network service
func (s *Service) handleReceivedMessage(msg network.Message) (err error) {
	msgType := msg.GetType()

	switch msgType {
	case network.BlockRequestMsgType: // 1
		err = s.ProcessBlockRequestMessage(msg)
	case network.BlockResponseMsgType: // 2
		err = s.ProcessBlockResponseMessage(msg)
	case network.BlockAnnounceMsgType: // 3
		err = s.ProcessBlockAnnounceMessage(msg)
	case network.TransactionMsgType: // 4
		err = s.ProcessTransactionMessage(msg)
	default:
		err = fmt.Errorf("received unsupported message type %d", msgType)
	}

	return err
}

func (s *Service) handleBabeSession() {
	for {
		<-s.epochDone
		log.Debug("[core] BABE epoch complete, initializing new session")

		// commit the storage trie to the DB
		err := s.storageState.StoreInDB()
		if err != nil {
			log.Error("[core]", "error", err)
		}

		newBlocks := make(chan types.Block)
		s.blkRec = newBlocks

		epochDone := make(chan struct{})
		s.epochDone = epochDone

		babeKill := make(chan struct{})
		s.babeKill = babeKill

		keys := s.keys.Sr25519Keypairs()

		latestSlot, err := s.blockState.GetSlotForBlock(s.blockState.HighestBlockHash())
		if err != nil {
			log.Error("[core]", "error", err)
		}

		// BABE session configuration
		bsConfig := &babe.SessionConfig{
			Keypair:          keys[0].(*sr25519.Keypair),
			Runtime:          s.rt,
			NewBlocks:        newBlocks, // becomes block send channel in BABE session
			BlockState:       s.blockState,
			StorageState:     s.storageState,
			TransactionQueue: s.transactionQueue,
			AuthData:         s.bs.AuthorityData(), // AuthorityData will be updated when the NextEpochDescriptor arrives.
			Done:             epochDone,
			Kill:             babeKill,
			StartSlot:        latestSlot + 1,
			SyncLock:         s.syncLock,
		}

		// create a new BABE session
		bs, err := babe.NewSession(bsConfig)
		if err != nil {
			log.Error("[core] could not initialize BABE", "error", err)
			return
		}

		err = bs.Start()
		if err != nil {
			log.Error("[core] could not start BABE", "error", err)
		}

		s.bs = bs
		log.Trace("[core] BABE session initialized and started")
	}
}

// handleConsensusDigest handles authority and randomness changes over transitions from one epoch to the next
//nolint
func (s *Service) handleConsensusDigest(header *types.Header) (err error) {
	var item types.DigestItem
	for _, digest := range header.Digest {
		item, err = types.DecodeDigestItem(digest)
		if err != nil {
			return err
		}

		if item.Type() == types.ConsensusDigestType {
			break
		}
	}

	// TODO: if this block is the first in the epoch and it doesn't have a consensus digest, this is an error
	if item == nil {
		return nil
	}

	consensusDigest := item.(*types.ConsensusDigest)

	epochData := new(babe.NextEpochDescriptor)
	err = epochData.Decode(consensusDigest.Data)
	if err != nil {
		return err
	}

	if s.isAuthority {
		// TODO: if this block isn't the first in the epoch, and it has a consensus digest, this is an error
		err = s.bs.SetEpochData(epochData)
		if err != nil {
			return err
		}
	}

	return nil
}
