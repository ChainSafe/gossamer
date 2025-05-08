// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package adapter

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/primitives/blockchain"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/lib/blocktree"
	"github.com/ChainSafe/gossamer/lib/common"
	rt "github.com/ChainSafe/gossamer/lib/runtime"
	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/pkg/trie"
)

type ClientAdapterDB interface {
	Get(key []byte) (value []byte, err error)
	Has(key []byte) (has bool, err error)
}

type Client[
	H runtime.Hash,
	Hasher runtime.Hasher[H],
	N runtime.Number,
	E runtime.Extrinsic,
	Header runtime.Header[N, H],
] interface {
	blockchain.HeaderBackend[H, N, Header]
	blockchain.BlockBackend[H, N, Header, Hasher, E]
	blockchain.Backend[H, N, Header, E]

	CompareAndSetBlockData(bd *types.BlockData) error
}

type ClientAdapter[
	H runtime.Hash,
	Hasher runtime.Hasher[H],
	N runtime.Number,
	E runtime.Extrinsic,
	Header runtime.Header[N, H],
] struct {
	client Client[H, Hasher, N, E, Header]
	db     ClientAdapterDB
}

func NewClientAdapter[
	H runtime.Hash,
	Hasher runtime.Hasher[H],
	N runtime.Number,
	E runtime.Extrinsic,
	Header runtime.Header[N, H],
](client Client[H, Hasher, N, E, Header], db ClientAdapterDB) *ClientAdapter[H, Hasher, N, E, Header] {
	return &ClientAdapter[H, Hasher, N, E, Header]{client: client, db: db}
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) AddBlock(*types.Block) error {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) AddBlockWithArrivalTime(block *types.Block,
	arrivalTime time.Time) error {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) BestBlock() (*types.Block, error) {
	signedBlock, err := ca.client.Block(ca.client.Info().BestHash)
	if err != nil {
		return nil, err
	}

	if signedBlock == nil {
		return nil, nil
	}

	return types.NewBlockFromGeneric(signedBlock.Block)
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) BestBlockHash() common.Hash {
	return common.NewHashFromGeneric(ca.client.Info().BestHash)
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) BestBlockHeader() (*types.Header, error) {
	bestBlock, err := ca.BestBlock()
	if err != nil {
		return nil, err
	}

	if bestBlock == nil {
		return nil, nil
	}

	return &bestBlock.Header, nil
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) BestBlockNumber() (number uint, err error) {
	return uint(ca.client.Info().BestNumber), nil
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) GenesisHash() common.Hash {
	return common.NewHashFromGeneric(ca.client.Info().GenesisHash)
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetBlockBody(hash common.Hash) (*types.Body, error) {
	block, err := ca.GetBlockByHash(hash)
	if err != nil {
		return nil, err
	}

	if block == nil {
		return nil, nil
	}

	return &block.Body, nil
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetBlockStateRoot(hash common.Hash) (common.Hash, error) {
	block, err := ca.GetBlockByHash(hash)
	if err != nil {
		return common.EmptyHash, err
	}

	if block == nil {
		return common.EmptyHash, nil
	}

	return block.Header.StateRoot, nil
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetBlockByHash(bhash common.Hash) (*types.Block, error) {
	hasher := *new(Hasher)
	hash := hasher.NewHash(bhash.ToBytes())
	block, err := ca.client.Block(hash)
	if err != nil {
		return nil, err
	}

	if block == nil {
		return nil, nil
	}

	return types.NewBlockFromGeneric(block.Block)
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetBlockByNumber(blockNumber uint) (*types.Block, error) {
	hash, err := ca.client.BlockHash(N(blockNumber))
	if err != nil {
		return nil, err
	}

	if hash == nil {
		return nil, nil
	}

	signedBlock, err := ca.client.Block(*hash)
	if err != nil {
		return nil, err
	}

	if signedBlock == nil {
		return nil, nil
	}

	return types.NewBlockFromGeneric(signedBlock.Block)
}

// TODO: remove from BlockState interface since it is only use by RPC and is not part of the standard
func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetFinalisedHeader(round, setID uint64) (*types.Header, error) {
	panic("unimplemented")
}

// TODO: check if this is the right implementation
func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetHashesByNumber(blockNumber uint) ([]common.Hash, error) {
	hash, err := ca.client.BlockHash(N(blockNumber))
	if err != nil {
		return nil, err
	}

	if hash == nil {
		return nil, nil
	}

	children, err := ca.client.Children(*hash)
	if err != nil {
		return nil, err
	}

	if children == nil {
		return nil, nil
	}

	hashes := make([]common.Hash, 0, len(children))

	for _, child := range children {
		block, err := ca.client.Block(child)
		if err != nil {
			return nil, err
		}

		if block.Block.Header().Number() == N(blockNumber) {
			hashes = append(hashes, common.NewHashFromGeneric(block.Block.Header().Hash()))
		}
	}

	return hashes, nil
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetHashByNumber(blockNumber uint) (common.Hash, error) {
	hash, err := ca.client.BlockHash(N(blockNumber))
	if err != nil {
		return common.Hash{}, err
	}

	return common.NewHashFromGeneric(*hash), nil
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetHeader(bhash common.Hash) (*types.Header, error) {
	block, err := ca.GetBlockByHash(bhash)

	if err != nil {
		return nil, err
	}

	if block == nil {
		return nil, nil
	}

	return &block.Header, nil
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetHeaderByNumber(num uint) (*types.Header, error) {
	block, err := ca.GetBlockByNumber(num)
	if err != nil {
		return nil, err
	}

	if block == nil {
		return nil, nil
	}

	return &block.Header, nil
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetHighestFinalisedHeader() (*types.Header, error) {
	header, err := ca.client.Header(ca.client.Info().FinalizedHash)
	if err != nil {
		return nil, err
	}

	if header == nil {
		return nil, nil
	}

	return types.NewHeaderFromGeneric(*header)
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetHighestFinalisedHash() (common.Hash, error) {
	return common.NewHashFromGeneric(ca.client.Info().FinalizedHash), nil
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetHighestRoundAndSetID() (uint64, uint64, error) {
	b, err := ca.db.Get(state.HighestRoundAndSetIDKey)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get highest round and setID: %w", err)
	}

	round := binary.LittleEndian.Uint64(b[:8])
	setID := binary.LittleEndian.Uint64(b[8:16])
	return round, setID, nil
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetJustification(bhash common.Hash) ([]byte, error) {
	data, err := ca.db.Get(prefixKey(bhash, state.JustificationPrefix))
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetFirstNonOriginSlotNumber() (uint64, error) {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetReceipt(hash common.Hash) ([]byte, error) {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetMessageQueue(hash common.Hash) ([]byte, error) {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetTries() *state.Tries {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetBlockHashesBySlot(slotNum uint64) ([]common.Hash, error) {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetAllBlocksAtNumber(num uint) ([]common.Hash, error) {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetNonFinalisedBlocks() []common.Hash {
	lastFinalized := ca.client.Info().FinalizedHash

	unfinalizedHashes, err := ca.client.Children(lastFinalized)
	if err != nil {
		return nil
	}

	hashes := make([]common.Hash, len(unfinalizedHashes))
	for i, hash := range unfinalizedHashes {
		hashes[i] = common.NewHashFromGeneric(hash)
	}

	return hashes
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetSlotForBlock(hash common.Hash) (uint64, error) {
	header, err := ca.GetHeader(hash)
	if err != nil {
		return 0, err
	}

	return types.GetSlotFromHeader(header)
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) HasFinalisedBlock(round, setID uint64) (bool, error) {
	return ca.db.Has(state.FinalisedHashKey(round, setID))
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) HasHeader(hash common.Hash) (bool, error) {
	nonFinalised := ca.GetNonFinalisedBlocks()
	for _, h := range nonFinalised {
		if h == hash {
			return true, nil
		}
	}

	return ca.HasHeaderInDatabase(hash)
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) HasJustification(bhash common.Hash) (bool, error) {
	hasher := *new(Hasher)
	hash := hasher.NewHash(bhash.ToBytes())
	justifications, err := ca.client.Justifications(hash)
	if err != nil {
		return false, err
	}

	return len(justifications) > 0, nil
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) HasHeaderInDatabase(hash common.Hash) (bool, error) {
	header, err := ca.GetHeader(hash)
	if err != nil {
		return false, err
	}

	return header != nil, nil
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetLastFinalized() common.Hash {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) SetFirstNonOriginSlotNumber(slotNumber uint64) error {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) SetFinalisedHash(
	hash common.Hash, round uint64, setID uint64, finalizeAncestors bool) error {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) SetFinalizedHeader(header *types.Header) error {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) SetHeader(header *types.Header) error {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) SetJustification(hash common.Hash, data []byte) error {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) SetHighestRoundAndSetID(round, setID uint64) error {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetRoundAndSetID() (uint64, uint64) {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetRuntime(blockHash common.Hash) (instance rt.Instance, err error) {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) UnregisterRuntimeUpdatedChannel(id uint32) bool {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) HandleRuntimeChanges(newState *rtstorage.TrieState,
	in rt.Instance, bHash common.Hash) error {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) CompareAndSetBlockData(bd *types.BlockData) error {
	return ca.client.CompareAndSetBlockData(bd)
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) IsDescendantOf(parent, child common.Hash) (bool, error) {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) LowestCommonAncestor(a, b common.Hash) (common.Hash, error) {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) NumberIsFinalised(blockNumber uint) (bool, error) {
	return ca.client.Info().FinalizedNumber >= N(blockNumber), nil
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) BlocktreeAsString() string {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) Leaves() []common.Hash {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) Range(startHash, endHash common.Hash) (
	hashes []common.Hash, err error) {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) RangeInMemory(start, end common.Hash) ([]common.Hash, error) {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) StoreRuntime(blockHash common.Hash, runtime rt.Instance) {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) FreeImportedBlockNotifierChannel(ch chan *types.Block) {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetImportedBlockNotifierChannel() chan *types.Block {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) FreeFinalisedNotifierChannel(ch chan *types.FinalisationInfo) {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetFinalisedNotifierChannel() chan *types.FinalisationInfo {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) RegisterRuntimeUpdatedChannel(ch chan<- rt.Version) (uint32, error) {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) Rewind(toBlock uint) error {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) IsPaused() bool {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) Pause() error {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) TrieState(root *common.Hash) (*rtstorage.TrieState, error) {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) StoreTrie(*rtstorage.TrieState, *types.Header) error {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetStateRootFromBlock(bhash *common.Hash) (*common.Hash, error) {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) GenerateTrieProof(stateRoot common.Hash, keys [][]byte) (
	[][]byte, error) {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetStorage(root *common.Hash, key []byte) ([]byte, error) {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetStorageByBlockHash(bhash *common.Hash, key []byte) (
	[]byte, error) {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) StorageRoot() (common.Hash, error) {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) Entries(root *common.Hash) (map[string][]byte, error) {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetKeysWithPrefix(root *common.Hash, prefix []byte) (
	[][]byte, error) {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetStorageChild(root *common.Hash, keyToChild []byte) (
	trie.Trie, error) {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetStorageFromChild(root *common.Hash, keyToChild, key []byte) (
	[]byte, error) {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) LoadCode(hash *common.Hash) ([]byte, error) {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) LoadCodeHash(hash *common.Hash) (common.Hash, error) {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) RegisterStorageObserver(o state.Observer) {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) UnregisterStorageObserver(o state.Observer) {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) SetBlockTree(blocktree *blocktree.BlockTree) {
	panic("unimplemented")
}

func prefixKey(hash common.Hash, prefix []byte) []byte {
	return append(prefix, hash.ToBytes()...)
}
