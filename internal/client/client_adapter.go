// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package client

import (
	"time"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/lib/common"
	rt "github.com/ChainSafe/gossamer/lib/runtime"
	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/pkg/trie"
)

type ClientAdapter[
	H runtime.Hash,
	Hasher runtime.Hasher[H],
	N runtime.Number,
	E runtime.Extrinsic,
	Header runtime.Header[N, H],
] struct {
	Client *Client[H, Hasher, N, E, Header]
}

func NewClientAdapter[
	H runtime.Hash,
	Hasher runtime.Hasher[H],
	N runtime.Number,
	E runtime.Extrinsic,
	Header runtime.Header[N, H],
](client *Client[H, Hasher, N, E, Header]) *ClientAdapter[H, Hasher, N, E, Header] {
	return &ClientAdapter[H, Hasher, N, E, Header]{Client: client}
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) AddBlock(*types.Block) error {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) AddBlockWithArrivalTime(block *types.Block,
	arrivalTime time.Time) error {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) BestBlock() (*types.Block, error) {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) BestBlockHash() common.Hash {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) BestBlockHeader() (*types.Header, error) {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) BestBlockNumber() (number uint, err error) {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) GenesisHash() common.Hash {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetBlockBody(hash common.Hash) (*types.Body, error) {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetBlockStateRoot(bhash common.Hash) (common.Hash, error) {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetBlockByHash(common.Hash) (*types.Block, error) {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetBlockByNumber(blockNumber uint) (*types.Block, error) {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetFinalisedHeader(round, setID uint64) (*types.Header, error) {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetFinalisedHash(round, setID uint64) (common.Hash, error) {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetHashesByNumber(blockNumber uint) ([]common.Hash, error) {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetHashByNumber(blockNumber uint) (common.Hash, error) {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetHeader(bhash common.Hash) (*types.Header, error) {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetHeaderByNumber(num uint) (*types.Header, error) {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetHighestFinalisedHeader() (*types.Header, error) {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetHighestFinalisedHash() (common.Hash, error) {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetHighestRoundAndSetID() (uint64, uint64, error) {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetJustification(common.Hash) ([]byte, error) {
	panic("unimplemented")
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
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetSlotForBlock(common.Hash) (uint64, error) {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) HasFinalisedBlock(round, setID uint64) (bool, error) {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) HasHeader(hash common.Hash) (bool, error) {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) HasJustification(hash common.Hash) (bool, error) {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) HasHeaderInDatabase(hash common.Hash) (bool, error) {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetLastFinalized() common.Hash {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) SetFirstNonOriginSlotNumber(slotNumber uint64) error {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) SetFinalisedHash(hash common.Hash, round uint64, setID uint64) error {
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
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) IsDescendantOf(parent, child common.Hash) (bool, error) {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) LowestCommonAncestor(a, b common.Hash) (common.Hash, error) {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) NumberIsFinalised(blockNumber uint) (bool, error) {
	panic("unimplemented")
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
