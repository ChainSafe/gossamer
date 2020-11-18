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
	"context"
	"math/big"
	"os"
	"sync"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/runtime/wasmer"
	"github.com/ChainSafe/gossamer/lib/services"
	"github.com/ChainSafe/gossamer/lib/transaction"

	log "github.com/ChainSafe/log15"
)

var _ services.Service = &Service{}

// Service is an overhead layer that allows communication between the runtime,
// BABE session, and network service. It deals with the validation of transactions
// and blocks by calling their respective validation functions in the runtime.
type Service struct {
	logger log.Logger
	ctx    context.Context
	cancel context.CancelFunc

	// State interfaces
	blockState       BlockState
	storageState     StorageState
	transactionState TransactionState

	// Current runtime and hash of the current runtime code
	rt       runtime.LegacyInstance
	codeHash common.Hash

	// Block production variables
	blockProducer   BlockProducer
	isBlockProducer bool

	// Finality gadget variables
	finalityGadget          FinalityGadget
	isFinalityAuthority     bool
	consensusMessageHandler ConsensusMessageHandler

	// Block verification
	verifier Verifier

	// Keystore
	keys *keystore.GlobalKeystore

	// Channels and interfaces for inter-process communication
	blkRec <-chan types.Block // receive blocks from BABE session
	net    Network

	blockAddCh   chan *types.Block // receive blocks added to blocktree
	blockAddChID byte

	// State variables
	lock *sync.Mutex // channel lock
}

// Config holds the configuration for the core Service.
type Config struct {
	LogLvl                  log.Lvl
	BlockState              BlockState
	StorageState            StorageState
	TransactionState        TransactionState
	Network                 Network
	Keystore                *keystore.GlobalKeystore
	Runtime                 runtime.LegacyInstance
	BlockProducer           BlockProducer
	IsBlockProducer         bool
	FinalityGadget          FinalityGadget
	IsFinalityAuthority     bool
	ConsensusMessageHandler ConsensusMessageHandler
	Verifier                Verifier

	NewBlocks     chan types.Block // only used for testing purposes
	BabeThreshold *big.Int         // used by Verifier, for development purposes
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

	if cfg.IsFinalityAuthority && cfg.FinalityGadget == nil {
		return nil, ErrNilFinalityGadget
	}

	if cfg.ConsensusMessageHandler == nil {
		return nil, ErrNilConsensusMessageHandler
	}

	logger := log.New("pkg", "core")
	h := log.StreamHandler(os.Stdout, log.TerminalFormat())
	h = log.CallerFileHandler(h)
	logger.SetHandler(log.LvlFilterHandler(cfg.LogLvl, h))

	sr, err := cfg.BlockState.BestBlockStateRoot()
	if err != nil {
		return nil, err
	}

	codeHash, err := cfg.StorageState.LoadCodeHash(&sr)
	if err != nil {
		return nil, err
	}

	blockAddCh := make(chan *types.Block, 16)
	id, err := cfg.BlockState.RegisterImportedChannel(blockAddCh)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	srv := &Service{
		logger:                  logger,
		ctx:                     ctx,
		cancel:                  cancel,
		rt:                      cfg.Runtime,
		codeHash:                codeHash,
		keys:                    cfg.Keystore,
		blkRec:                  cfg.NewBlocks,
		blockState:              cfg.BlockState,
		storageState:            cfg.StorageState,
		transactionState:        cfg.TransactionState,
		net:                     cfg.Network,
		isBlockProducer:         cfg.IsBlockProducer,
		blockProducer:           cfg.BlockProducer,
		finalityGadget:          cfg.FinalityGadget,
		consensusMessageHandler: cfg.ConsensusMessageHandler,
		verifier:                cfg.Verifier,
		isFinalityAuthority:     cfg.IsFinalityAuthority,
		lock:                    &sync.Mutex{},
		blockAddCh:              blockAddCh,
		blockAddChID:            id,
	}

	if cfg.NewBlocks != nil {
		srv.blkRec = cfg.NewBlocks
	} else if cfg.IsBlockProducer {
		srv.blkRec = cfg.BlockProducer.GetBlockChannel()
	}

	return srv, nil
}

// Start starts the core service
func (s *Service) Start() error {
	// we can ignore the `cancel` function returned by `context.WithCancel` since Stop() cancels the parent context,
	// so all the child contexts should also be canceled. potentially update if there is a better way to do this

	// start receiving blocks from BABE session
	ctx, _ := context.WithCancel(s.ctx) //nolint
	go s.receiveBlocks(ctx)

	// start receiving messages from network service
	ctx, _ = context.WithCancel(s.ctx) //nolint

	// start handling imported blocks
	ctx, _ = context.WithCancel(s.ctx) //nolint
	go s.handleBlocks(ctx)

	if s.isFinalityAuthority && s.finalityGadget != nil {
		s.logger.Debug("routing finality gadget messages")
		ctx, _ = context.WithCancel(s.ctx) //nolint
		go s.sendVoteMessages(ctx)

		ctx, _ = context.WithCancel(s.ctx) //nolint
		go s.sendFinalizationMessages(ctx)
	}

	return nil
}

// Stop stops the core service
func (s *Service) Stop() error {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.cancel()

	s.blockState.UnregisterImportedChannel(s.blockAddChID)
	close(s.blockAddCh)

	return nil
}

// StorageRoot returns the hash of the storage root
func (s *Service) StorageRoot() (common.Hash, error) {
	if s.storageState == nil {
		return common.Hash{}, ErrNilStorageState
	}

	ts, err := s.storageState.TrieState(nil)
	if err != nil {
		return common.Hash{}, err
	}

	return ts.Root()
}

func (s *Service) handleBlocks(ctx context.Context) {
	for {
		prev := s.blockState.BestBlockHash()

		select {
		case block := <-s.blockAddCh:
			if block == nil {
				continue
			}

			if err := s.storageState.StoreInDB(block.Header.StateRoot); err != nil {
				s.logger.Warn("failed to store storage trie in database", "error", err)
			}

			if err := s.handleChainReorg(prev, block.Header.Hash()); err != nil {
				s.logger.Warn("failed to re-add transactions to chain upon re-org", "error", err)
			}

			if err := s.maintainTransactionPool(block); err != nil {
				s.logger.Warn("failed to maintain transaction pool", "error", err)
			}

			if err := s.handleRuntimeChanges(block.Header); err != nil {
				s.logger.Warn("failed to handle runtime change for block", "block", block.Header.Hash(), "error", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

// receiveBlocks starts receiving blocks from the BABE session
func (s *Service) receiveBlocks(ctx context.Context) {
	for {
		select {
		case block := <-s.blkRec:
			if block.Header == nil {
				continue
			}

			err := s.handleReceivedBlock(&block)
			if err != nil {
				s.logger.Warn("failed to handle block from BABE session", "err", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

// HandleMessage handles network messages that are passed to it
func (s *Service) HandleMessage(message network.Message) {
	if message == nil {
		return
	}

	if s.ctx.Err() != nil {
		return
	}

	err := s.handleReceivedMessage(message)
	if err != nil {
		s.logger.Trace("failed to handle message from network service", "err", err)
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

	s.logger.Debug("added block from BABE", "header", block.Header, "body", block.Body)

	msg := &network.BlockAnnounceMessage{
		ParentHash:     block.Header.ParentHash,
		Number:         block.Header.Number,
		StateRoot:      block.Header.StateRoot,
		ExtrinsicsRoot: block.Header.ExtrinsicsRoot,
		Digest:         block.Header.Digest,
	}

	s.net.SendMessage(msg)
	return nil
}

// handleReceivedMessage handles messages from the network service
func (s *Service) handleReceivedMessage(msg network.Message) (err error) {
	msgType := msg.Type()

	switch msgType {
	case network.TransactionMsgType: // 4
		msg, ok := msg.(*network.TransactionMessage)
		if !ok {
			return ErrMessageCast("TransactionMessage")
		}

		err = s.ProcessTransactionMessage(msg)
	case network.ConsensusMsgType: // 5
		msg, ok := msg.(*network.ConsensusMessage)
		if !ok {
			return ErrMessageCast("ConsensusMessage")
		}

		err = s.processConsensusMessage(msg)
	default:
		err = ErrUnsupportedMsgType(msgType)
	}

	return err
}

// handleRuntimeChanges checks if changes to the runtime code have occurred; if so, load the new runtime
// It also updates the BABE service and block verifier with the new runtime
func (s *Service) handleRuntimeChanges(header *types.Header) error {
	sr, err := s.blockState.BestBlockStateRoot()
	if err != nil {
		return err
	}

	currentCodeHash, err := s.storageState.LoadCodeHash(&sr)
	if err != nil {
		return err
	}

	if !bytes.Equal(currentCodeHash[:], s.codeHash[:]) {
		code, err := s.storageState.LoadCode(&sr)
		if err != nil {
			return err
		}

		s.rt.Stop()

		ts, err := s.storageState.TrieState(&sr)
		if err != nil {
			return err
		}

		cfg := &wasmer.Config{
			Imports: wasmer.ImportsLegacyNodeRuntime,
		}
		cfg.Storage = ts
		cfg.Keystore = s.keys.Acco.(*keystore.GenericKeystore)
		cfg.LogLvl = -1
		cfg.NodeStorage = s.rt.NodeStorage()
		cfg.Network = s.rt.NetworkService()

		s.rt, err = wasmer.NewLegacyInstance(code, cfg)
		if err != nil {
			return err
		}

		if s.isBlockProducer {
			err = s.blockProducer.SetRuntime(s.rt)
			if err != nil {
				return err
			}
		}

		err = s.verifier.SetRuntimeChangeAtBlock(header, s.rt)
		if err != nil {
			return err
		}
	}

	return nil
}

// handleChainReorg checks if there is a chain re-org (ie. new chain head is on a different chain than the
// previous chain head). If there is a re-org, it moves the transactions that were included on the previous
// chain back into the transaction pool.
func (s *Service) handleChainReorg(prev, curr common.Hash) error {
	ancestor, err := s.blockState.HighestCommonAncestor(prev, curr)
	if err != nil {
		return err
	}

	// if the highest common ancestor of the previous chain head and current chain head is the previous chain head,
	// then the current chain head is the descendant of the previous and thus are on the same chain
	if ancestor == prev {
		return nil
	}

	subchain, err := s.blockState.SubChain(ancestor, prev)
	if err != nil {
		return err
	}

	// for each block in the previous chain, re-add its extrinsics back into the pool
	for _, hash := range subchain {
		body, err := s.blockState.GetBlockBody(hash)
		if err != nil {
			continue
		}

		exts, err := body.AsExtrinsics()
		if err != nil {
			continue
		}

		for _, ext := range exts {
			s.logger.Trace("validating transaction on re-org chain", "extrinsic", ext)

			txv, err := s.rt.ValidateTransaction(ext)
			if err != nil {
				s.logger.Trace("failed to validate transaction", "extrinsic", ext)
				continue
			}

			vtx := transaction.NewValidTransaction(ext, txv)
			s.transactionState.AddToPool(vtx)
		}
	}

	return nil
}

// maintainTransactionPool removes any transactions that were included in the new block, revalidates the transactions in the pool,
// and moves them to the queue if valid.
// See https://github.com/paritytech/substrate/blob/74804b5649eccfb83c90aec87bdca58e5d5c8789/client/transaction-pool/src/lib.rs#L545
func (s *Service) maintainTransactionPool(block *types.Block) error {
	exts, err := block.Body.AsExtrinsics()
	if err != nil {
		return err
	}

	// remove extrinsics included in a block
	for _, ext := range exts {
		s.transactionState.RemoveExtrinsic(ext)
	}

	// re-validate transactions in the pool and move them to the queue
	txs := s.transactionState.PendingInPool()
	for _, tx := range txs {
		// TODO: re-add this on update to v0.8

		// val, err := s.rt.ValidateTransaction(tx.Extrinsic)
		// if err != nil {
		// 	// failed to validate tx, remove it from the pool or queue
		// 	s.transactionState.RemoveExtrinsic(ext)
		// 	continue
		// }

		// tx = transaction.NewValidTransaction(tx.Extrinsic, val)

		h, err := s.transactionState.Push(tx)
		if err != nil && err == transaction.ErrTransactionExists {
			// transaction is already in queue, remove it from the pool
			s.transactionState.RemoveExtrinsicFromPool(tx.Extrinsic)
			continue
		}

		s.transactionState.RemoveExtrinsicFromPool(tx.Extrinsic)
		s.logger.Trace("moved transaction to queue", "hash", h)
	}

	return nil
}

// InsertKey inserts keypair into the account keystore
// TODO: define which keystores need to be updated and create separate insert funcs for each
func (s *Service) InsertKey(kp crypto.Keypair) {
	s.keys.Acco.Insert(kp)
}

// HasKey returns true if given hex encoded public key string is found in keystore, false otherwise, error if there
//  are issues decoding string
func (s *Service) HasKey(pubKeyStr string, keyType string) (bool, error) {
	return keystore.HasKey(pubKeyStr, keyType, s.keys.Acco)
}

// GetRuntimeVersion gets the current RuntimeVersion
func (s *Service) GetRuntimeVersion() (*runtime.VersionAPI, error) {
	ts, err := s.storageState.TrieState(nil)
	if err != nil {
		return nil, err
	}

	s.rt.SetContext(ts)
	return s.rt.Version()
}

// IsBlockProducer returns true if node is a block producer
func (s *Service) IsBlockProducer() bool {
	return s.isBlockProducer
}

// HandleSubmittedExtrinsic is used to send a Transaction message containing a Extrinsic @ext
func (s *Service) HandleSubmittedExtrinsic(ext types.Extrinsic) error {
	msg := &network.TransactionMessage{Extrinsics: []types.Extrinsic{ext}}
	s.net.SendMessage(msg)
	return nil
}

//GetMetadata calls runtime Metadata_metadata function
func (s *Service) GetMetadata() ([]byte, error) {
	return s.rt.Metadata()
}
