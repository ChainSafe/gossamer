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
	"bytes"
	"errors"
	"math/big"
	"sync"
	"sync/atomic"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/babe"
	"github.com/ChainSafe/gossamer/lib/blocktree"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/services"

	database "github.com/ChainSafe/chaindb"
	log "github.com/ChainSafe/log15"
)

var _ services.Service = &Service{}

var maxResponseSize int64 = 8 // maximum number of block datas to reply with in a BlockResponse message.

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

	// Block production variables
	blockProducer   BlockProducer
	isBlockProducer bool

	// Keystore
	keys *keystore.Keystore

	// Channels for inter-process communication
	msgRec  <-chan network.Message // receive messages from network service
	msgSend chan<- network.Message // send messages to network service
	blkRec  <-chan types.Block     // receive blocks from BABE session

	// State variables
	lock    *sync.Mutex
	started uint32

	// Block synchronization
	blockNumOut chan<- *big.Int                      // send block numbers from peers to Syncer
	respOut     chan<- *network.BlockResponseMessage // send incoming BlockResponseMessags to Syncer
	syncLock    *sync.Mutex
	syncer      *Syncer
}

// Config holds the configuration for the core Service.
type Config struct {
	BlockState       BlockState
	StorageState     StorageState
	TransactionQueue TransactionQueue
	Keystore         *keystore.Keystore
	Runtime          *runtime.Runtime
	BlockProducer    BlockProducer
	IsBlockProducer  bool

	NewBlocks chan types.Block // only used for testing purposes
	Verifier  Verifier         // only used for testing purposes

	MsgRec   <-chan network.Message
	MsgSend  chan<- network.Message
	SyncChan chan *big.Int
}

// NewService returns a new core service that connects the runtime, BABE
// session, and network service.
func NewService(cfg *Config) (*Service, error) {
	if cfg.Keystore == nil {
		return nil, ErrNilKeystore
	}

	if cfg.BlockState == nil {
		return nil, ErrNilBlockState
	}

	if cfg.StorageState == nil {
		return nil, ErrNilStorageState
	}

	if cfg.Runtime == nil {
		return nil, ErrNilRuntime
	}

	if cfg.IsBlockProducer && cfg.BlockProducer == nil {
		return nil, ErrNilBlockProducer
	}

	codeHash, err := cfg.StorageState.LoadCodeHash()
	if err != nil {
		return nil, err
	}

	syncerLock := &sync.Mutex{}
	respChan := make(chan *network.BlockResponseMessage, 128)
	chanLock := &sync.Mutex{}

	var srv = &Service{}

	srv = &Service{
		rt:               cfg.Runtime,
		codeHash:         codeHash,
		keys:             cfg.Keystore,
		msgRec:           cfg.MsgRec,
		msgSend:          cfg.MsgSend,
		blockState:       cfg.BlockState,
		storageState:     cfg.StorageState,
		transactionQueue: cfg.TransactionQueue,
		isBlockProducer:  cfg.IsBlockProducer,
		lock:             chanLock,
		syncLock:         syncerLock,
		blockNumOut:      cfg.SyncChan,
		respOut:          respChan,
	}

	if cfg.NewBlocks != nil {
		srv.blkRec = cfg.NewBlocks
	} else if cfg.IsBlockProducer {
		srv.blkRec = cfg.BlockProducer.GetBlockChannel()
	}

	// TODO: change to atomic.Value
	canLock := atomic.CompareAndSwapUint32(&srv.started, 0, 1)
	if !canLock {
		return nil, errors.New("failed to change Service status from stopped to started")
	}

	// load BABE verification data from runtime
	// TODO: authority data may change, use NextEpochDescriptor if available
	babeCfg, err := srv.rt.BabeConfiguration()
	if err != nil {
		return nil, err
	}

	ad, err := types.BABEAuthorityDataRawToAuthorityData(babeCfg.GenesisAuthorities)
	if err != nil {
		return nil, err
	}

	currentDescriptor := &babe.NextEpochDescriptor{
		Authorities: ad,
		Randomness:  babeCfg.Randomness,
	}

	if cfg.Verifier == nil {
		// TODO: load current epoch from database chain head
		cfg.Verifier, err = babe.NewVerificationManager(cfg.BlockState, 0, currentDescriptor)
		if err != nil {
			return nil, err
		}
	}

	// only one process is starting *core.Service, don't need to use atomic here
	srv.started = 1

	syncerCfg := &SyncerConfig{
		BlockState:       cfg.BlockState,
		BlockNumIn:       cfg.SyncChan,
		RespIn:           respChan,
		MsgOut:           cfg.MsgSend,
		ChanLock:         chanLock,
		TransactionQueue: cfg.TransactionQueue,
		Verifier:         cfg.Verifier,
		Runtime:          cfg.Runtime,
	}

	syncer, err := NewSyncer(syncerCfg)
	if err != nil {
		return nil, err
	}

	srv.syncer = syncer

	// core service
	return srv, nil
}

// Start starts the core service
func (s *Service) Start() error {

	// start receiving blocks from BABE session
	go s.receiveBlocks()

	// start receiving messages from network service
	go s.receiveMessages()

	// start syncer
	err := s.syncer.Start()
	if err != nil {
		log.Error("[core] could not start syncer", "error", err)
		return err
	}

	return nil
}

// Stop stops the core service
func (s *Service) Stop() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	// close channel to network service
	// TODO: change to atomic.Value
	if atomic.LoadUint32(&s.started) == uint32(1) {
		if s.msgSend != nil {
			close(s.msgSend)
		}

		defer func() {
			if ok := atomic.CompareAndSwapUint32(&s.started, 1, 0); !ok {
				log.Error("[core] failed to change Service status from started to stopped")
			}
		}()

	}

	err := s.syncer.Stop()
	if err != nil {
		return err
	}

	return nil
}

// StorageRoot returns the hash of the storage root
func (s *Service) StorageRoot() (common.Hash, error) {
	if s.storageState == nil {
		return common.Hash{}, ErrNilStorageState
	}
	return s.storageState.StorageRoot()
}

func (s *Service) safeMsgSend(msg network.Message) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if atomic.LoadUint32(&s.started) == uint32(0) {
		return ErrServiceStopped
	}

	s.msgSend <- msg
	return nil
}

// receiveBlocks starts receiving blocks from the BABE session
func (s *Service) receiveBlocks() {
	// receive block from BABE session
	for block := range s.blkRec {
		if block.Header != nil {
			err := s.handleReceivedBlock(&block)
			if err != nil {
				log.Error("[core] failed to handle block from BABE session", "err", err)
			}
		} else {
			log.Trace("[core] receiveBlocks got nil Header")
		}
	}
}

// receiveMessages starts receiving messages from the network service
func (s *Service) receiveMessages() {
	// receive message from network service
	for msg := range s.msgRec {
		if msg == nil {
			log.Error("[core] failed to receive message from network service")
			continue
		}

		err := s.handleReceivedMessage(msg)
		if err == blocktree.ErrDescendantNotFound || err == blocktree.ErrStartNodeNotFound || err == database.ErrKeyNotFound {
			log.Trace("[core] failed to handle message from network service", "err", err)
		} else if err != nil {
			log.Error("[core] failed to handle message from network service", "err", err)
		}
	}
}

// handleReceivedBlock handles blocks from the BABE session
func (s *Service) handleReceivedBlock(block *types.Block) (err error) {
	if s.blockState == nil {
		return ErrNilBlockState
	}

	err = s.blockState.AddBlock(block)
	if err != nil {
		return err
	}

	log.Debug("[core] added block from BABE", "header", block.Header, "body", block.Body)

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

	return s.checkForRuntimeChanges()
}

// handleReceivedMessage handles messages from the network service
func (s *Service) handleReceivedMessage(msg network.Message) (err error) {
	msgType := msg.GetType()

	switch msgType {
	case network.BlockRequestMsgType: // 1
		msg, ok := msg.(*network.BlockRequestMessage)
		if !ok {
			return ErrMessageCast("BlockRequestMessage")
		}

		err = s.ProcessBlockRequestMessage(msg)
	case network.BlockResponseMsgType: // 2
		msg, ok := msg.(*network.BlockResponseMessage)
		if !ok {
			return ErrMessageCast("BlockResponseMessage")
		}

		err = s.ProcessBlockResponseMessage(msg)
	case network.BlockAnnounceMsgType: // 3
		msg, ok := msg.(*network.BlockAnnounceMessage)
		if !ok {
			return ErrMessageCast("BlockAnnounceMessage")
		}

		err = s.ProcessBlockAnnounceMessage(msg)
	case network.TransactionMsgType: // 4
		msg, ok := msg.(*network.TransactionMessage)
		if !ok {
			return ErrMessageCast("TransactionMessage")
		}

		err = s.ProcessTransactionMessage(msg)
	default:
		err = ErrUnsupportedMsgType(msgType)
	}

	return err
}

// checkForRuntimeChanges checks if changes to the runtime code have occurred; if so, load the new runtime
func (s *Service) checkForRuntimeChanges() error {
	currentCodeHash, err := s.storageState.LoadCodeHash()
	if err != nil {
		return err
	}

	if !bytes.Equal(currentCodeHash[:], s.codeHash[:]) {
		code, err := s.storageState.LoadCode()
		if err != nil {
			return err
		}

		s.rt.Stop()

		s.rt, err = runtime.NewRuntime(code, s.storageState, s.keys, runtime.RegisterImports_NodeRuntime)
		if err != nil {
			return err
		}

		if s.isBlockProducer {
			err = s.blockProducer.SetRuntime(s.rt)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// InsertKey inserts keypair into keystore
func (s *Service) InsertKey(kp crypto.Keypair) {
	s.keys.Insert(kp)
}

// HasKey returns true if given hex encoded public key string is found in keystore, false otherwise, error if there
//  are issues decoding string
func (s *Service) HasKey(pubKeyStr string, keyType string) (bool, error) {
	return keystore.HasKey(pubKeyStr, keyType, s.keys)
}

// GetRuntimeVersion gets the current RuntimeVersion
func (s *Service) GetRuntimeVersion() (*runtime.VersionAPI, error) {
	//TODO ed, change this so that it can lookup runtime by block hash
	version := &runtime.VersionAPI{
		RuntimeVersion: &runtime.Version{},
		API:            nil,
	}

	ret, err := s.rt.Exec(runtime.CoreVersion, []byte{})
	if err != nil {
		return nil, err
	}
	err = version.Decode(ret)
	if err != nil {
		return nil, err
	}

	return version, nil
}

// IsBlockProducer returns true if node is a block producer
func (s *Service) IsBlockProducer() bool {
	return s.isBlockProducer
}

// HandleSubmittedExtrinsic is used to send a Transaction message containing a Extrinsic @ext
func (s *Service) HandleSubmittedExtrinsic(ext types.Extrinsic) error {
	msg := &network.TransactionMessage{Extrinsics: []types.Extrinsic{ext}}
	return s.safeMsgSend(msg)
}

//GetMetadata calls runtime Metadata_metadata function
func (s *Service) GetMetadata() ([]byte, error) {
	return s.rt.Exec(runtime.Metadata_metadata, []byte{})
}
