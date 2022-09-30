// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package core

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/blocktree"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/runtime/wasmer"
	"github.com/ChainSafe/gossamer/lib/services"
	"github.com/ChainSafe/gossamer/lib/transaction"
	cscale "github.com/centrifuge/go-substrate-rpc-client/v4/scale"
	ctypes "github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

var (
	_      services.Service = &Service{}
	logger                  = log.NewFromGlobal(log.AddContext("pkg", "core"))
)

// QueryKeyValueChanges represents the key-value data inside a block storage
type QueryKeyValueChanges map[string]string

// Service is an overhead layer that allows communication between the runtime,
// BABE session, and network service. It deals with the validation of transactions
// and blocks by calling their respective validation functions in the runtime.
type Service struct {
	ctx        context.Context
	cancel     context.CancelFunc
	blockAddCh chan *types.Block // for asynchronous block handling
	sync.Mutex                   // lock for channel

	// Service interfaces
	blockState       BlockState
	epochState       EpochState
	storageState     StorageState
	transactionState TransactionState
	net              Network

	// map of code substitutions keyed by block hash
	codeSubstitute       map[common.Hash]string
	codeSubstitutedState CodeSubstitutedState

	// Keystore
	keys *keystore.GlobalKeystore
}

// Config holds the configuration for the core Service.
type Config struct {
	LogLvl log.Level

	BlockState       BlockState
	EpochState       EpochState
	StorageState     StorageState
	TransactionState TransactionState
	Network          Network
	Keystore         *keystore.GlobalKeystore
	Runtime          RuntimeInstance

	CodeSubstitutes      map[common.Hash]string
	CodeSubstitutedState CodeSubstitutedState
}

// NewService returns a new core service that connects the runtime, BABE
// session, and network service.
func NewService(cfg *Config) (*Service, error) {
	logger.Patch(log.SetLevel(cfg.LogLvl))

	blockAddCh := make(chan *types.Block, 256)

	ctx, cancel := context.WithCancel(context.Background())
	srv := &Service{
		ctx:                  ctx,
		cancel:               cancel,
		keys:                 cfg.Keystore,
		blockState:           cfg.BlockState,
		epochState:           cfg.EpochState,
		storageState:         cfg.StorageState,
		transactionState:     cfg.TransactionState,
		net:                  cfg.Network,
		blockAddCh:           blockAddCh,
		codeSubstitute:       cfg.CodeSubstitutes,
		codeSubstitutedState: cfg.CodeSubstitutedState,
	}

	return srv, nil
}

// Start starts the core service
func (s *Service) Start() error {
	go s.handleBlocksAsync()
	return nil
}

// Stop stops the core service
func (s *Service) Stop() error {
	s.Lock()
	defer s.Unlock()

	s.cancel()
	close(s.blockAddCh)
	return nil
}

// StorageRoot returns the hash of the storage root
func (s *Service) StorageRoot() (common.Hash, error) {
	ts, err := s.storageState.TrieState(nil)
	if err != nil {
		return common.Hash{}, err
	}

	return ts.Root()
}

// HandleBlockImport handles a block that was imported via the network
func (s *Service) HandleBlockImport(block *types.Block, state *rtstorage.TrieState) error {
	return s.handleBlock(block, state)
}

// HandleBlockProduced handles a block that was produced by us
// It is handled the same as an imported block in terms of state updates; the only difference
// is we send a BlockAnnounceMessage to our peers.
func (s *Service) HandleBlockProduced(block *types.Block, state *rtstorage.TrieState) error {
	if err := s.handleBlock(block, state); err != nil {
		return err
	}

	digest := types.NewDigest()
	for i := range block.Header.Digest.Types {
		digestValue, err := block.Header.Digest.Types[i].Value()
		if err != nil {
			return fmt.Errorf("getting value of digest type at index %d: %w", i, err)
		}
		err = digest.Add(digestValue)
		if err != nil {
			return err
		}
	}

	msg := &network.BlockAnnounceMessage{
		ParentHash:     block.Header.ParentHash,
		Number:         block.Header.Number,
		StateRoot:      block.Header.StateRoot,
		ExtrinsicsRoot: block.Header.ExtrinsicsRoot,
		Digest:         digest,
		BestBlock:      true,
	}

	s.net.GossipMessage(msg)
	return nil
}

func (s *Service) handleBlock(block *types.Block, state *rtstorage.TrieState) error {
	if block == nil || state == nil {
		return ErrNilBlockHandlerParameter
	}

	// store updates state trie nodes in database
	err := s.storageState.StoreTrie(state, &block.Header)
	if err != nil {
		logger.Warnf("failed to store state trie for imported block %s: %s",
			block.Header.Hash(), err)
		return err
	}

	// store block in database
	if err = s.blockState.AddBlock(block); err != nil {
		if errors.Is(err, blocktree.ErrParentNotFound) && block.Header.Number != 0 {
			return err
		} else if errors.Is(err, blocktree.ErrBlockExists) || block.Header.Number == 0 {
			// this is fine
		} else {
			return err
		}
	}

	logger.Debugf("imported block %s and stored state trie with root %s",
		block.Header.Hash(), state.MustRoot())

	rt, err := s.blockState.GetRuntime(&block.Header.ParentHash)
	if errors.Is(err, blocktree.ErrFailedToGetRuntime) {
		rt, err = s.storageState.GetRuntimeFromDB(block.Header.ParentHash)
		if err != nil {
			return fmt.Errorf("getting runtime from database: %w", err)
		}
		// ensure the runtime stops and releases resources since it was
		// instantiated from disk just for this function call.
		defer rt.Stop()
	} else if err != nil {
		return err
	}

	// check for runtime changes
	if err := s.blockState.HandleRuntimeChanges(state, rt, block.Header.Hash()); err != nil {
		logger.Criticalf("failed to update runtime code: %s", err)
		return err
	}

	// check if there was a runtime code substitution
	err = s.handleCodeSubstitution(block.Header.Hash(), state)
	if err != nil {
		logger.Criticalf("failed to substitute runtime code: %s", err)
		return err
	}

	go func() {
		s.Lock()
		defer s.Unlock()
		if s.ctx.Err() != nil {
			return
		}

		s.blockAddCh <- block
	}()

	return nil
}

func (s *Service) handleCodeSubstitution(hash common.Hash,
	state *rtstorage.TrieState) (err error) {
	value := s.codeSubstitute[hash]
	if value == "" {
		return nil
	}

	logger.Infof("🔄 detected runtime code substitution, upgrading for block %s...", hash)
	code := common.MustHexToBytes(value)
	if len(code) == 0 {
		return fmt.Errorf("%w: for hash %s", ErrEmptyRuntimeCode, hash)
	}

	rt, err := s.blockState.GetRuntime(&hash)
	if err != nil {
		return fmt.Errorf("getting runtime from block state: %w", err)
	}

	// this needs to create a new runtime instance, otherwise it will update
	// the blocks that reference the current runtime version to use the code substition
	cfg := wasmer.Config{
		Storage:     state,
		Keystore:    rt.Keystore(),
		NodeStorage: rt.NodeStorage(),
		Network:     rt.NetworkService(),
	}

	if rt.Validator() {
		cfg.Role = 4
	}

	next, err := wasmer.NewInstance(code, cfg)
	if err != nil {
		return fmt.Errorf("creating new runtime instance: %w", err)
	}

	err = s.codeSubstitutedState.StoreCodeSubstitutedBlockHash(hash)
	if err != nil {
		return fmt.Errorf("storing code substituted block hash: %w", err)
	}

	s.blockState.StoreRuntime(hash, next)
	return nil
}

// handleBlocksAsync handles a block asynchronously; the handling performed by this function
// does not need to be completed before the next block can be imported.
func (s *Service) handleBlocksAsync() {
	for {
		prev := s.blockState.BestBlockHash()

		select {
		case block, ok := <-s.blockAddCh:
			if !ok {
				return
			}

			if block == nil {
				continue
			}

			if err := s.handleChainReorg(prev, block.Header.Hash()); err != nil {
				logger.Warnf("failed to re-add transactions to chain upon re-org: %s", err)
			}

			s.maintainTransactionPool(block)
		case <-s.ctx.Done():
			return
		}
	}
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

	// subchain contains the ancestor as well so we need to remove it.
	if len(subchain) > 0 {
		subchain = subchain[1:]
	} else {
		return nil
	}

	// Check transaction validation on the best block.
	rt := s.blockState.GetBestBlockRuntime()

	if rt == nil {
		return ErrNilRuntime
	}

	// for each block in the previous chain, re-add its extrinsics back into the pool
	for _, hash := range subchain {
		body, err := s.blockState.GetBlockBody(hash)
		if err != nil || body == nil {
			continue
		}

		for _, ext := range *body {
			logger.Tracef("validating transaction on re-org chain for extrinsic %s", ext)
			decExt := &ctypes.Extrinsic{}
			decoder := cscale.NewDecoder(bytes.NewReader(ext))
			if err = decoder.Decode(&decExt); err != nil {
				continue
			}

			// Inherent are not signed.
			if !decExt.IsSigned() {
				continue
			}

			externalExt := make(types.Extrinsic, 0, 1+len(ext))
			externalExt = append(externalExt, byte(types.TxnExternal))
			externalExt = append(externalExt, ext...)
			txv, err := rt.ValidateTransaction(externalExt)
			if err != nil {
				logger.Debugf("failed to validate transaction for extrinsic %s: %s", ext, err)
				continue
			}
			vtx := transaction.NewValidTransaction(ext, txv)
			s.transactionState.AddToPool(vtx)
		}
	}

	return nil
}

// maintainTransactionPool removes any transactions that were included in
// the new block, revalidates the transactions in the pool, and moves
// them to the queue if valid.
// See https://github.com/paritytech/substrate/blob/74804b5649eccfb83c90aec87bdca58e5d5c8789/client/transaction-pool/src/lib.rs#L545
func (s *Service) maintainTransactionPool(block *types.Block) {
	// remove extrinsics included in a block
	for _, ext := range block.Body {
		s.transactionState.RemoveExtrinsic(ext)
	}

	// re-validate transactions in the pool and move them to the queue
	txs := s.transactionState.PendingInPool()
	for _, tx := range txs {
		// get the best block corresponding runtime
		rt := s.blockState.GetBestBlockRuntime()

		txnValidity, err := rt.ValidateTransaction(tx.Extrinsic)
		if err != nil {
			s.transactionState.RemoveExtrinsic(tx.Extrinsic)
			continue
		}

		tx = transaction.NewValidTransaction(tx.Extrinsic, txnValidity)

		// Err is only thrown if tx is already in pool, in which case it still gets removed
		h, _ := s.transactionState.Push(tx)

		s.transactionState.RemoveExtrinsicFromPool(tx.Extrinsic)
		logger.Tracef("moved transaction %s to queue", h)
	}
}

// InsertKey inserts keypair into the account keystore
func (s *Service) InsertKey(kp crypto.Keypair, keystoreType string) error {
	ks, err := s.keys.GetKeystore([]byte(keystoreType))
	if err != nil {
		return err
	}

	return ks.Insert(kp)
}

// HasKey returns true if given hex encoded public key string is found in keystore, false otherwise, error if there
// are issues decoding string
func (s *Service) HasKey(pubKeyStr, keystoreType string) (bool, error) {
	ks, err := s.keys.GetKeystore([]byte(keystoreType))
	if err != nil {
		return false, err
	}

	return keystore.HasKey(pubKeyStr, keystoreType, ks)
}

// DecodeSessionKeys executes the runtime DecodeSessionKeys and return the scale encoded keys
func (s *Service) DecodeSessionKeys(enc []byte) ([]byte, error) {
	rt := s.blockState.GetBestBlockRuntime()

	return rt.DecodeSessionKeys(enc)
}

// GetRuntimeVersion gets the current RuntimeVersion
func (s *Service) GetRuntimeVersion(bhash *common.Hash) (
	version runtime.Version, err error) {
	var stateRootHash *common.Hash

	// If block hash is not nil then fetch the state root corresponding to the block.
	if bhash != nil {
		var err error
		stateRootHash, err = s.storageState.GetStateRootFromBlock(bhash)
		if err != nil {
			return version, err
		}
	}

	ts, err := s.storageState.TrieState(stateRootHash)
	if err != nil {
		return version, err
	}

	rt, err := s.blockState.GetRuntime(bhash)
	if errors.Is(err, blocktree.ErrFailedToGetRuntime) {
		rt, err = s.storageState.GetRuntimeFromDB(*bhash)
		if err != nil {
			return version, fmt.Errorf("getting runtime from database: %w", err)
		}
		version = rt.Version()

		rt.Stop()
		return version, nil
	} else if err != nil {
		return version, err
	}

	rt.SetContextStorage(ts)
	return rt.Version(), nil
}

// HandleSubmittedExtrinsic is used to send a Transaction message containing a Extrinsic @ext
func (s *Service) HandleSubmittedExtrinsic(ext types.Extrinsic) error {
	if s.net == nil {
		return nil
	}

	if s.transactionState.Exists(ext) {
		return nil
	}

	bestBlockHash := s.blockState.BestBlockHash()

	stateRoot, err := s.storageState.GetStateRootFromBlock(&bestBlockHash)
	if err != nil {
		return fmt.Errorf("could not get state root from block %s: %w", bestBlockHash, err)
	}

	ts, err := s.storageState.TrieState(stateRoot)
	if err != nil {
		return err
	}

	rt := s.blockState.GetBestBlockRuntime()

	rt.SetContextStorage(ts)
	// the transaction source is External
	externalExt := types.Extrinsic(append([]byte{byte(types.TxnExternal)}, ext...))
	transactionValidity, err := rt.ValidateTransaction(externalExt)
	if err != nil {
		return err
	}

	// add transaction to pool
	vtx := transaction.NewValidTransaction(ext, transactionValidity)
	s.transactionState.AddToPool(vtx)

	// broadcast transaction
	msg := &network.TransactionMessage{Extrinsics: []types.Extrinsic{ext}}
	s.net.GossipMessage(msg)
	return nil
}

// GetMetadata calls runtime Metadata_metadata function
func (s *Service) GetMetadata(bhash *common.Hash) ([]byte, error) {
	var (
		stateRootHash *common.Hash
		err           error
	)

	// If block hash is not nil then fetch the state root corresponding to the block.
	if bhash != nil {
		stateRootHash, err = s.storageState.GetStateRootFromBlock(bhash)
		if err != nil {
			return nil, err
		}
	}
	ts, err := s.storageState.TrieState(stateRootHash)
	if err != nil {
		return nil, err
	}

	rt, err := s.blockState.GetRuntime(bhash)
	if errors.Is(err, blocktree.ErrFailedToGetRuntime) {
		rt, err = s.storageState.GetRuntimeFromDB(*bhash)
		if err != nil {
			return nil, fmt.Errorf("getting runtime from database: %w", err)
		}
		// ensure the runtime stops and releases resources since it was
		// instantiated from disk just for this function call.
		defer rt.Stop()
		metadata, err := rt.Metadata()
		if err != nil {
			return nil, fmt.Errorf("getting runtime metadata: %w", err)
		}
		return metadata, nil
	} else if err != nil {
		return nil, err
	}

	rt.SetContextStorage(ts)
	return rt.Metadata()
}

// GetReadProofAt will return an array with the proofs for the keys passed as params
// based on the block hash passed as param as well, if block hash is nil then the current state will take place
func (s *Service) GetReadProofAt(block common.Hash, keys [][]byte) (
	hash common.Hash, proofForKeys [][]byte, err error) {
	if block.IsEmpty() {
		block = s.blockState.BestBlockHash()
	}

	stateRoot, err := s.blockState.GetBlockStateRoot(block)
	if err != nil {
		return hash, nil, err
	}

	proofForKeys, err = s.storageState.GenerateTrieProof(stateRoot, keys)
	if err != nil {
		return hash, nil, err
	}

	return block, proofForKeys, nil
}
