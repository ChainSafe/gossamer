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
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	wazero_runtime "github.com/ChainSafe/gossamer/lib/runtime/wazero"
	"github.com/ChainSafe/gossamer/lib/transaction"
	"github.com/ChainSafe/gossamer/pkg/trie"

	cscale "github.com/centrifuge/go-substrate-rpc-client/v4/scale"
	ctypes "github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

var (
	logger = log.NewFromGlobal(log.AddContext("pkg", "core"))
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
	lock       sync.Mutex        // lock for channel

	// Service interfaces
	blockState       BlockState
	storageState     StorageState
	transactionState TransactionState
	grandpaState     GrandpaState
	net              Network

	// map of code substitutions keyed by block hash
	codeSubstitute       map[common.Hash]string
	codeSubstitutedState CodeSubstitutedState

	// Keystore
	keys          *keystore.GlobalKeystore
	onBlockImport BlockImportDigestHandler
}

// Config holds the configuration for the core Service.
type Config struct {
	LogLvl log.Level

	BlockState       BlockState
	StorageState     StorageState
	TransactionState TransactionState
	GrandpaState     GrandpaState
	Network          Network
	Keystore         *keystore.GlobalKeystore
	Runtime          runtime.Instance

	CodeSubstitutes      map[common.Hash]string
	CodeSubstitutedState CodeSubstitutedState
	OnBlockImport        BlockImportDigestHandler
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
		storageState:         cfg.StorageState,
		transactionState:     cfg.TransactionState,
		grandpaState:         cfg.GrandpaState,
		net:                  cfg.Network,
		blockAddCh:           blockAddCh,
		codeSubstitute:       cfg.CodeSubstitutes,
		codeSubstitutedState: cfg.CodeSubstitutedState,
		onBlockImport:        cfg.OnBlockImport,
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
	s.lock.Lock()
	defer s.lock.Unlock()

	s.cancel()
	close(s.blockAddCh)
	return nil
}

func (s *Service) getCurrentStateTrieVersion() (trie.TrieLayout, error) {
	bestBlockHash := s.blockState.BestBlockHash()
	rt, err := s.blockState.GetRuntime(bestBlockHash)
	if err != nil {
		return trie.NoVersion, err
	}

	runtimeVersion, err := rt.Version()
	if err != nil {
		return trie.NoVersion, err
	}

	return trie.ParseVersion(runtimeVersion.StateVersion)
}

// StorageRoot returns the hash of the storage root
func (s *Service) StorageRoot() (common.Hash, error) {
	ts, err := s.storageState.TrieState(nil)
	if err != nil {
		return common.Hash{}, err
	}

	stateTrieVersion, err := s.getCurrentStateTrieVersion()
	if err != nil {
		return common.Hash{}, err
	}

	return stateTrieVersion.Hash(ts.Trie())
}

// HandleBlockImport handles a block that was imported via the network
func (s *Service) HandleBlockImport(block *types.Block, state *rtstorage.TrieState, announce bool) error {
	err := s.handleBlock(block, state)
	if err != nil {
		return fmt.Errorf("handling block: %w", err)
	}

	if !announce {
		return nil
	}

	bestBlockHash := s.blockState.BestBlockHash()
	isBestBlock := bestBlockHash == block.Header.Hash()

	blockAnnounce, err := createBlockAnnounce(block, isBestBlock)
	if err != nil {
		return fmt.Errorf("creating block announce: %w", err)
	}

	s.net.GossipMessage(blockAnnounce)
	return nil
}

// HandleBlockProduced handles a block that was produced by us
// It is handled the same as an imported block in terms of state updates; the only difference
// is we send a BlockAnnounceMessage to our peers.
func (s *Service) HandleBlockProduced(block *types.Block, state *rtstorage.TrieState) error {
	err := s.handleBlock(block, state)
	if err != nil {
		return fmt.Errorf("handling block: %w", err)
	}

	blockAnnounce, err := createBlockAnnounce(block, true)
	if err != nil {
		return fmt.Errorf("creating block announce: %w", err)
	}

	s.net.GossipMessage(blockAnnounce)
	return nil
}

func createBlockAnnounce(block *types.Block, isBestBlock bool) (
	blockAnnounce *network.BlockAnnounceMessage, err error) {
	digest := types.NewDigest()
	for i := range block.Header.Digest {
		digestValue, err := block.Header.Digest[i].Value()
		if err != nil {
			return nil, fmt.Errorf("getting value of digest type at index %d: %w", i, err)
		}
		err = digest.Add(digestValue)
		if err != nil {
			return nil, fmt.Errorf("adding digest value for type at index %d: %w", i, err)
		}
	}

	return &network.BlockAnnounceMessage{
		ParentHash:     block.Header.ParentHash,
		Number:         block.Header.Number,
		StateRoot:      block.Header.StateRoot,
		ExtrinsicsRoot: block.Header.ExtrinsicsRoot,
		Digest:         digest,
		BestBlock:      isBestBlock,
	}, nil
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

	err = s.onBlockImport.HandleDigests(&block.Header)
	if err != nil {
		return fmt.Errorf("on block import handle: %w", err)
	}

	err = s.grandpaState.ApplyForcedChanges(&block.Header)
	if err != nil {
		return fmt.Errorf("applying forced changes: %w", err)
	}

	logger.Infof("imported block %s and stored state trie with root %s",
		block.Header.Hash(), state.MustRoot())

	parentRuntimeInstance, err := s.blockState.GetRuntime(block.Header.ParentHash)
	if err != nil {
		return err
	}

	// check for runtime changes
	err = s.blockState.HandleRuntimeChanges(state, parentRuntimeInstance, block.Header.Hash())
	if err != nil {
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
		s.lock.Lock()
		defer s.lock.Unlock()
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

	rt, err := s.blockState.GetRuntime(hash)
	if err != nil {
		return fmt.Errorf("getting runtime from block state: %w", err)
	}

	// this needs to create a new runtime instance, otherwise it will update
	// the blocks that reference the current runtime version to use the code substition
	cfg := wazero_runtime.Config{
		Storage:     state,
		Keystore:    rt.Keystore(),
		NodeStorage: rt.NodeStorage(),
		Network:     rt.NetworkService(),
	}

	if rt.Validator() {
		cfg.Role = 4
	}

	next, err := wazero_runtime.NewInstance(code, cfg)
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
		select {
		case block, ok := <-s.blockAddCh:
			if !ok {
				return
			}

			if block == nil {
				continue
			}

			bestBlockHash := s.blockState.BestBlockHash()
			if err := s.handleChainReorg(bestBlockHash, block.Header.Hash()); err != nil {
				// TODO remove once gossamer is in stable state
				panic(fmt.Errorf("failed to re-add transactions to chain upon re-org: %s", err))
			}

			if err := s.maintainTransactionPool(block, bestBlockHash); err != nil {
				// TODO remove once gossamer is in stable state
				panic(fmt.Errorf("failed to maintain txn pool after re-org: %s", err))
			}
		case <-s.ctx.Done():
			return
		}
	}
}

// handleChainReorg checks if there is a chain re-org (ie. new chain head is on a different chain than the
// previous chain head). If there is a re-org, it moves the transactions that were included on the previous
// chain back into the transaction pool.
func (s *Service) handleChainReorg(best, curr common.Hash) error {
	ancestor, err := s.blockState.LowestCommonAncestor(best, curr)
	if err != nil {
		return err
	}

	// if the highest common ancestor of the previous chain head and current chain head is the previous chain head,
	// then the current chain head is the descendant of the previous and thus are on the same chain
	if ancestor == best || ancestor == curr {
		return nil
	}

	subchain, err := s.blockState.RangeInMemory(ancestor, best)
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
	bestBlockHash := s.blockState.BestBlockHash()
	rt, err := s.blockState.GetRuntime(bestBlockHash)
	if err != nil {
		return err
	}

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
				return fmt.Errorf("decoding extrinsic: %s", err)
			}

			// Inherent are not signed.
			if !decExt.IsSigned() {
				continue
			}

			externalExt, err := s.buildExternalTransaction(rt, ext)
			if err != nil {
				return fmt.Errorf("building external transaction: %s", err)
			}

			transactionValidity, err := rt.ValidateTransaction(externalExt)
			if err != nil {
				logger.Debugf("failed to validate transaction for extrinsic %s: %s skipping in chain reorg", ext, err)
				s.transactionState.RemoveExtrinsic(ext)
				continue
			}
			vtx := transaction.NewValidTransaction(ext, transactionValidity)
			s.transactionState.AddToPool(vtx)
		}
	}

	return nil
}

// maintainTransactionPool removes any transactions that were included in
// the new block, revalidates the transactions in the pool, and moves
// them to the queue if valid.
// See https://github.com/paritytech/substrate/blob/74804b5649eccfb83c90aec87bdca58e5d5c8789/client/transaction-pool/src/lib.rs#L545
func (s *Service) maintainTransactionPool(block *types.Block, bestBlockHash common.Hash) error {
	// remove extrinsics included in a block
	for _, ext := range block.Body {
		s.transactionState.RemoveExtrinsic(ext)
	}

	stateRoot, err := s.storageState.GetStateRootFromBlock(&bestBlockHash)
	if err != nil {
		logger.Errorf("could not get state root from block %s: %w", bestBlockHash, err)
		return err
	}

	ts, err := s.storageState.TrieState(stateRoot)
	if err != nil {
		logger.Errorf(err.Error())
		return err
	}

	// re-validate transactions in the pool and move them to the queue
	txs := s.transactionState.PendingInPool()
	for _, tx := range txs {
		bestBlockHash := s.blockState.BestBlockHash()
		rt, err := s.blockState.GetRuntime(bestBlockHash)
		if err != nil {
			return fmt.Errorf("failed to get runtime to re-validate transactions in pool: %s", err)
		}

		rt.SetContextStorage(ts)
		externalExt, err := s.buildExternalTransaction(rt, tx.Extrinsic)
		if err != nil {
			return fmt.Errorf("building external transaction: %s", err)
		}

		txnValidity, err := rt.ValidateTransaction(externalExt)
		if err != nil {
			logger.Debugf("failed to validate transaction for extrinsic %s: %s", tx.Extrinsic, err)
			s.transactionState.RemoveExtrinsic(tx.Extrinsic)
			continue
		}

		tx = transaction.NewValidTransaction(tx.Extrinsic, txnValidity)

		// Err is only thrown if tx is already in pool, in which case it still gets removed
		h, _ := s.transactionState.Push(tx)

		s.transactionState.RemoveExtrinsicFromPool(tx.Extrinsic)
		logger.Tracef("moved transaction %s to queue", h)
	}
	return nil
}

// InsertKey inserts keypair into the account keystore
func (s *Service) InsertKey(kp KeyPair, keystoreType string) error {
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
func (s *Service) DecodeSessionKeys(encodedSessionKeys []byte) ([]byte, error) {
	bestBlockHash := s.blockState.BestBlockHash()
	rt, err := s.blockState.GetRuntime(bestBlockHash)
	if err != nil {
		return nil, err
	}

	return rt.DecodeSessionKeys(encodedSessionKeys)
}

// GetRuntimeVersion gets the current RuntimeVersion
func (s *Service) GetRuntimeVersion(bhash *common.Hash) (
	version runtime.Version, err error) {
	rt, err := prepareRuntime(bhash, s.storageState, s.blockState)
	if err != nil {
		return version, fmt.Errorf("setting up runtime: %w", err)
	}
	return rt.Version()
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

	rt, err := s.blockState.GetRuntime(bestBlockHash)
	if err != nil {
		logger.Critical("failed to get runtime")
		return err
	}

	rt.SetContextStorage(ts)

	externalExt, err := s.buildExternalTransaction(rt, ext)
	if err != nil {
		return fmt.Errorf("building external transaction: %w", err)
	}

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
func (s *Service) GetMetadata(bhash *common.Hash) (metadata []byte, err error) {
	rt, err := prepareRuntime(bhash, s.storageState, s.blockState)
	if err != nil {
		return nil, fmt.Errorf("setting up runtime: %w", err)
	}
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

// buildExternalTransaction builds an external transaction based on the current transaction queue API version
// See https://github.com/paritytech/substrate/blob/polkadot-v0.9.25/primitives/transaction-pool/src/runtime_api.rs#L25-L55
func (s *Service) buildExternalTransaction(rt runtime.Instance, ext types.Extrinsic) (types.Extrinsic, error) {
	runtimeVersion, err := rt.Version()
	if err != nil {
		return nil, err
	}
	txQueueVersion, err := runtimeVersion.TaggedTransactionQueueVersion()
	if err != nil {
		return nil, err
	}
	var extrinsicParts [][]byte
	switch txQueueVersion {
	case 3:
		extrinsicParts = [][]byte{{byte(types.TxnExternal)}, ext, s.blockState.BestBlockHash().ToBytes()}
	case 2:
		extrinsicParts = [][]byte{{byte(types.TxnExternal)}, ext}
	default:
		return nil, fmt.Errorf("%w: %d", errInvalidTransactionQueueVersion, txQueueVersion)
	}
	return types.Extrinsic(bytes.Join(extrinsicParts, nil)), nil
}

func prepareRuntime(blockHash *common.Hash, storageState StorageState,
	blockState BlockState) (instance runtime.Instance, err error) {
	var stateRootHash *common.Hash
	if blockHash != nil {
		stateRootHash, err = storageState.GetStateRootFromBlock(blockHash)
		if err != nil {
			return nil, fmt.Errorf("getting state root from block hash: %w", err)
		}
	}

	trieState, err := storageState.TrieState(stateRootHash)
	if err != nil {
		return nil, fmt.Errorf("getting trie state: %w", err)
	}

	var blockHashValue common.Hash
	if blockHash != nil {
		blockHashValue = *blockHash
	} else {
		blockHashValue = blockState.BestBlockHash()
	}
	instance, err = blockState.GetRuntime(blockHashValue)
	if err != nil {
		return nil, fmt.Errorf("getting runtime: %w", err)
	}

	instance.SetContextStorage(trieState)
	return instance, nil
}
