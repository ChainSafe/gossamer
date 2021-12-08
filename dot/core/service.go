// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package core

import (
	"context"
	"fmt"
	"math/big"
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
	"github.com/ChainSafe/gossamer/pkg/scale"
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
	digestHandler    DigestHandler

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
	Runtime          runtime.Instance
	DigestHandler    DigestHandler

	CodeSubstitutes      map[common.Hash]string
	CodeSubstitutedState CodeSubstitutedState
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

	if cfg.Network == nil {
		return nil, ErrNilNetwork
	}

	if cfg.DigestHandler == nil {
		return nil, ErrNilDigestHandler
	}

	if cfg.CodeSubstitutedState == nil {
		return nil, errNilCodeSubstitutedState
	}

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
		digestHandler:        cfg.DigestHandler,
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
	if s.storageState == nil {
		return common.Hash{}, ErrNilStorageState
	}

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
		err := digest.Add(block.Header.Digest.Types[i].Value())
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
		return fmt.Errorf("unable to handle block due to nil parameter")
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
		if err == blocktree.ErrParentNotFound && block.Header.Number.Cmp(big.NewInt(0)) != 0 {
			return err
		} else if err == blocktree.ErrBlockExists || block.Header.Number.Cmp(big.NewInt(0)) == 0 {
			// this is fine
		} else {
			return err
		}
	}

	logger.Debugf("imported block %s and stored state trie with root %s",
		block.Header.Hash(), state.MustRoot())

	// handle consensus digests
	s.digestHandler.HandleDigests(&block.Header)

	rt, err := s.blockState.GetRuntime(&block.Header.ParentHash)
	if err != nil {
		return err
	}

	// check for runtime changes
	if err := s.blockState.HandleRuntimeChanges(state, rt, block.Header.Hash()); err != nil {
		logger.Criticalf("failed to update runtime code: %s", err)
		return err
	}

	// check if there was a runtime code substitution
	if err := s.handleCodeSubstitution(block.Header.Hash(), state); err != nil {
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

func (s *Service) handleCodeSubstitution(hash common.Hash, state *rtstorage.TrieState) error {
	value := s.codeSubstitute[hash]
	if value == "" {
		return nil
	}

	logger.Infof("🔄 detected runtime code substitution, upgrading for block %s...", hash)
	code := common.MustHexToBytes(value)
	if len(code) == 0 {
		return ErrEmptyRuntimeCode
	}

	rt, err := s.blockState.GetRuntime(&hash)
	if err != nil {
		return err
	}

	// this needs to create a new runtime instance, otherwise it will update
	// the blocks that reference the current runtime version to use the code substition
	cfg := &wasmer.Config{
		Imports: wasmer.ImportsNodeRuntime,
	}

	cfg.Storage = state
	cfg.Keystore = rt.Keystore()
	cfg.NodeStorage = rt.NodeStorage()
	cfg.Network = rt.NetworkService()

	if rt.Validator() {
		cfg.Role = 4
	}

	next, err := wasmer.NewInstance(code, cfg)
	if err != nil {
		return err
	}

	err = s.codeSubstitutedState.StoreCodeSubstitutedBlockHash(hash)
	if err != nil {
		return err
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
	rt, err := s.blockState.GetRuntime(nil)
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
			encExt, err := scale.Marshal(ext)
			if err != nil {
				return err
			}

			// decode extrinsic and make sure it's not an inherent.
			decExt := &types.ExtrinsicData{}
			if err = decExt.DecodeVersion(encExt); err != nil {
				continue
			}

			// Inherent are not signed.
			if !decExt.IsSigned() {
				continue
			}

			externalExt := types.Extrinsic(append([]byte{byte(types.TxnExternal)}, encExt...))
			txv, err := rt.ValidateTransaction(externalExt)
			if err != nil {
				logger.Debugf("failed to validate transaction for extrinsic %s: %s", ext, err)
				continue
			}

			vtx := transaction.NewValidTransaction(encExt, txv)
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
		// TODO: re-add this, need to update tests (#904)
		// val, err := s.rt.ValidateTransaction(tx.Extrinsic)
		// if err != nil {
		// 	// failed to validate tx, remove it from the pool or queue
		// 	s.transactionState.RemoveExtrinsic(tx.Extrinsic)
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
	rt, err := s.blockState.GetRuntime(nil)
	if err != nil {
		return nil, err
	}

	return rt.DecodeSessionKeys(enc)
}

// GetRuntimeVersion gets the current RuntimeVersion
func (s *Service) GetRuntimeVersion(bhash *common.Hash) (runtime.Version, error) {
	var stateRootHash *common.Hash

	// If block hash is not nil then fetch the state root corresponding to the block.
	if bhash != nil {
		var err error
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
	if err != nil {
		return nil, err
	}

	rt.SetContextStorage(ts)
	return rt.Version()
}

// HandleSubmittedExtrinsic is used to send a Transaction message containing a Extrinsic @ext
func (s *Service) HandleSubmittedExtrinsic(ext types.Extrinsic) error {
	if s.net == nil {
		return nil
	}

	if s.transactionState.IsInPool(ext) {
		return nil
	}

	ts, err := s.storageState.TrieState(nil)
	if err != nil {
		return err
	}

	rt, err := s.blockState.GetRuntime(nil)
	if err != nil {
		logger.Critical("failed to get runtime")
		return err
	}

	rt.SetContextStorage(ts)
	// the transaction source is External
	externalExt := types.Extrinsic(append([]byte{byte(types.TxnExternal)}, ext...))
	txv, err := rt.ValidateTransaction(externalExt)
	if err != nil {
		return err
	}

	// add transaction to pool
	vtx := transaction.NewValidTransaction(ext, txv)
	s.transactionState.AddToPool(vtx)

	// broadcast transaction
	msg := &network.TransactionMessage{Extrinsics: []types.Extrinsic{ext}}
	s.net.GossipMessage(msg)
	return nil
}

//GetMetadata calls runtime Metadata_metadata function
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
	if err != nil {
		return nil, err
	}

	rt.SetContextStorage(ts)
	return rt.Metadata()
}

// QueryStorage returns the key-value data by block based on `keys` params
// on every block starting `from` until `to` block, if `to` is not nil
func (s *Service) QueryStorage(from, to common.Hash, keys ...string) (map[common.Hash]QueryKeyValueChanges, error) {
	if to.IsEmpty() {
		to = s.blockState.BestBlockHash()
	}

	blocksToQuery, err := s.blockState.SubChain(from, to)
	if err != nil {
		return nil, err
	}

	queries := make(map[common.Hash]QueryKeyValueChanges)

	for _, hash := range blocksToQuery {
		changes, err := s.tryQueryStorage(hash, keys...)
		if err != nil {
			return nil, err
		}

		queries[hash] = changes
	}

	return queries, nil
}

// tryQueryStorage will try to get all the `keys` inside the block's current state
func (s *Service) tryQueryStorage(block common.Hash, keys ...string) (QueryKeyValueChanges, error) {
	stateRootHash, err := s.storageState.GetStateRootFromBlock(&block)
	if err != nil {
		return nil, err
	}

	changes := make(QueryKeyValueChanges)
	for _, k := range keys {
		keyBytes, err := common.HexToBytes(k)
		if err != nil {
			return nil, err
		}

		storedData, err := s.storageState.GetStorage(stateRootHash, keyBytes)
		if err != nil {
			return nil, err
		}

		if storedData == nil {
			continue
		}

		changes[k] = common.BytesToHex(storedData)
	}

	return changes, nil
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
