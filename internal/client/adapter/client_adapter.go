// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package adapter

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/client/api"
	"github.com/ChainSafe/gossamer/internal/client/api/utils"
	client_consensus_common "github.com/ChainSafe/gossamer/internal/client/consensus/common"
	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/internal/primitives/blockchain"
	primitives_consensus_common "github.com/ChainSafe/gossamer/internal/primitives/consensus/common"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	statemachine "github.com/ChainSafe/gossamer/internal/primitives/state-machine"
	"github.com/ChainSafe/gossamer/internal/primitives/state-machine/overlayedchanges"
	"github.com/ChainSafe/gossamer/internal/primitives/storage"
	"github.com/ChainSafe/gossamer/lib/blocktree"
	"github.com/ChainSafe/gossamer/lib/common"
	rt "github.com/ChainSafe/gossamer/lib/runtime"
	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/ChainSafe/gossamer/pkg/trie"
	"github.com/ChainSafe/gossamer/pkg/trie/db"
	"github.com/ChainSafe/gossamer/pkg/trie/inmemory"
)

var ErrMissingOverlayedChanges = errors.New("missing overlayed changes")
var ErrMissingStorageVersion = errors.New("missing storage version")

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
	blockchain.HeaderMetadata[H, N]
	blockchain.BlockBackend[H, N, Header, Hasher, E]
	blockchain.Backend[H, N, Header, E]
	api.StorageProvider[H, Hasher]
	client_consensus_common.BlockImport[H, N, E, Header]

	CompareAndSetBlockData(bd *types.BlockData) error
	StateAt(hash H) (statemachine.Backend[H, Hasher], error)
}

type Backend[H runtime.Hash, Hasher runtime.Hasher[H]] interface {
	statemachine.Backend[H, Hasher]
}

type ClientAdapter[
	H runtime.Hash,
	Hasher runtime.Hasher[H],
	N runtime.Number,
	E runtime.Extrinsic,
	Header runtime.Header[N, H],
] struct {
	backend Backend[H, Hasher]
	client  Client[H, Hasher, N, E, Header]
	db      ClientAdapterDB
}

func NewClientAdapter[
	H runtime.Hash,
	Hasher runtime.Hasher[H],
	N runtime.Number,
	E runtime.Extrinsic,
	Header runtime.Header[N, H],
](
	client Client[H, Hasher, N, E, Header],
	db ClientAdapterDB,
	backend Backend[H, Hasher],
) *ClientAdapter[H, Hasher, N, E, Header] {
	return &ClientAdapter[H, Hasher, N, E, Header]{client: client, db: db, backend: backend}
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) AddBlock(
	block *types.Block,
	changes *overlayedchanges.OverlayedChanges[H, Hasher],
	storageVersion *storage.StateVersion,
) error {
	if changes == nil {
		return ErrMissingOverlayedChanges
	}
	if storageVersion == nil {
		return ErrMissingStorageVersion
	}
	// Convert old header into generic one
	encodedHeader, err := scale.Marshal(block.Header)
	if err != nil {
		return err
	}
	genericHeader := *new(Header)
	err = scale.Unmarshal(encodedHeader, &genericHeader)
	if err != nil {
		return err
	}

	// Convert old extrinsics into generic ones
	encodedExtrinsics, err := scale.Marshal(block.Body)
	if err != nil {
		return err
	}

	var extrinsics []E
	err = scale.Unmarshal(encodedExtrinsics, &extrinsics)
	if err != nil {
		return err
	}

	storageChanges, err := changes.DrainStorageChanges(ca.backend, *storageVersion)
	if err != nil {
		return err
	}

	blockImportParams := &client_consensus_common.BlockImportParams[H, N, E, Header]{
		Origin: primitives_consensus_common.NetworkInitialSyncBlockOrigin,
		Header: genericHeader,
		Body:   extrinsics,
		StateAction: client_consensus_common.StateActionApplyChanges{
			StorageChanges: client_consensus_common.Changes[H, Hasher](storageChanges),
		},
	}

	_, err = ca.client.ImportBlock(blockImportParams)
	if err != nil {
		return err
	}

	return err
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

// GetFinalisedHeader returns the finalised block header by round and setID
// TODO: remove from BlockState interface since it is only use by RPC and is not part of the standard
func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetFinalisedHeader(round, setID uint64) (*types.Header, error) {
	rawHash, err := ca.db.Get(state.FinalisedHashKey(round, setID))
	if err != nil {
		return nil, err
	}

	return ca.GetHeader(common.NewHash(rawHash))
}

// GetHashesByNumber returns all block hashes at the given height.
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

// GetHighestRoundAndSetID gets the highest round and setID that have been finalised
func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetHighestRoundAndSetID() (uint64, uint64, error) {
	b, err := ca.db.Get(state.HighestRoundAndSetIDKey)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get highest round and setID: %w", err)
	}

	round := binary.LittleEndian.Uint64(b[:8])
	setID := binary.LittleEndian.Uint64(b[8:16])
	return round, setID, nil
}

// GetJustification retrieves a Justification from the database
func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetJustification(bhash common.Hash) ([]byte, error) {
	return ca.db.Get(prefixKey(bhash, state.JustificationPrefix))
}

// GetFirstNonOriginSlotNumber returns the slot number of the first non origin block
func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetFirstNonOriginSlotNumber() (uint64, error) {
	rawVal, err := ca.db.Get(state.FirstSlotNumberKey)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return 0, nil
		}
		return 0, err
	}

	val := binary.LittleEndian.Uint64(rawVal)
	return val, nil
}

// GetReceipt retrieves a Receipt from the database
func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetReceipt(hash common.Hash) ([]byte, error) {
	return ca.db.Get(prefixKey(hash, state.ReceiptPrefix))
}

// GetMessageQueue retrieves a MessageQueue from the database
func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetMessageQueue(hash common.Hash) ([]byte, error) {
	return ca.db.Get(prefixKey(hash, state.MessageQueuePrefix))
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetTries() *state.Tries {
	panic("unimplemented")
}

// GetAllBlocksAtNumber returns all unfinalised blocks with the given number
func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetAllBlocksAtNumber(num uint) ([]common.Hash, error) {
	return ca.GetHashesByNumber(num)
}

// GetNonFinalisedBlocks get all the blocks in the blocktree
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
	return common.NewHashFromGeneric(ca.client.Info().FinalizedHash)
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

func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetRuntime(blockHash common.Hash) (instance rt.Instance, err error) {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) UnregisterRuntimeUpdatedChannel(id uint32) bool {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) HandleRuntimeChanges(
	newState rtstorage.TrieState,
	in rt.Instance,
	bHash common.Hash,
) error {
	panic("unimplemented")
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) CompareAndSetBlockData(bd *types.BlockData) error {
	return ca.client.CompareAndSetBlockData(bd)
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) IsDescendantOf(parent, child common.Hash) (bool, error) {
	hasher := *new(Hasher)
	parentHash := hasher.NewHash(parent.ToBytes())
	childHash := hasher.NewHash(child.ToBytes())

	return utils.IsDescendantOf(ca.client, nil)(parentHash, childHash)
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) LowestCommonAncestor(a, b common.Hash) (common.Hash, error) {
	hasher := *new(Hasher)
	hashA := hasher.NewHash(a.ToBytes())
	hashB := hasher.NewHash(b.ToBytes())
	ancestor, err := blockchain.LowestCommonAncestor(ca.client, hashA, hashB)
	if err != nil {
		return common.EmptyHash, err
	}

	return common.NewHashFromGeneric(ancestor.Hash), nil
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) NumberIsFinalised(blockNumber uint) (bool, error) {
	return ca.client.Info().FinalizedNumber >= N(blockNumber), nil
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) BlocktreeAsString() string {
	panic("unimplemented")
}

// Leaves returns hashes of all blocks that are leaves of the block tree.
// in other words, that have no children, are chain heads.
// Results must be ordered best (longest, highest) chain first.
func (ca *ClientAdapter[H, Hasher, N, E, Header]) Leaves() []common.Hash {
	leaves, _ := ca.client.Leaves()
	hashes := make([]common.Hash, len(leaves))

	for i, leaf := range leaves {
		hashes[i] = common.NewHashFromGeneric(leaf)
	}

	return hashes
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) Range(startHash, endHash common.Hash) (
	hashes []common.Hash, err error) {
	hasher := *new(Hasher)
	start := hasher.NewHash(startHash.ToBytes())
	end := hasher.NewHash(endHash.ToBytes())

	treeRoute, err := blockchain.NewTreeRoute(ca.client, start, end)
	if err != nil {
		return nil, err
	}

	route := treeRoute.Route

	hashes = make([]common.Hash, len(route))
	for i, hashNumber := range route {
		hashes[i] = common.NewHashFromGeneric(hashNumber.Hash)
	}

	return hashes, nil
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) RangeInMemory(start, end common.Hash) ([]common.Hash, error) {
	return ca.Range(start, end)
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

func (ca *ClientAdapter[H, Hasher, N, E, Header]) TrieState(bhash *common.Hash) (rtstorage.TrieState, error) {
	stateAt, err := ca.stateAt(bhash)
	if err != nil {
		return nil, err
	}

	ext := overlayedchanges.NewExt(overlayedchanges.NewOverlayedChanges[H, Hasher](), stateAt)
	return rtstorage.NewExtBackedTrieState(ext), nil
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) StoreTrie(rtstorage.TrieState, *types.Header) error {
	panic("unimplemented")
}

// GetStateRootFromBlock returns the state root of the block with the given hash.
// Uses the best block hash when called with `nil`.
func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetStateRootFromBlock(bhash *common.Hash) (*common.Hash, error) {
	hash := ca.hashToGeneric(bhash)
	header, err := ca.client.Header(hash)
	if err != nil {
		return nil, err
	}
	if header == nil {
		return nil, database.ErrNotFound
	}

	h := common.NewHashFromGeneric((*header).StateRoot())
	return &h, nil
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) GenerateTrieProof(stateRoot common.Hash, keys [][]byte) (
	[][]byte, error) {
	//hasher := *new(Hasher)
	//rootHash := hasher.NewHash(stateRoot.ToBytes())
	//
	//sKeys := make([]string, len(keys))
	//for i, k := range keys {
	//	sKeys[i] = string(k)
	//}
	//
	//return proof.NewMerkleProof[H, Hasher](/* FIXME */, trie.DefaultStateVersion, rootHash, sKeys)
	panic("unimplemented")
}

// GetStorage queries the state that corresponds to the given state root hash for the data at the given key.
func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetStorage(bhash *common.Hash, key []byte) ([]byte, error) {
	return ca.client.Storage(ca.hashToGeneric(bhash), key)
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) StorageRoot() (common.Hash, error) {
	header, err := ca.client.Header(ca.client.Info().BestHash)
	if err != nil {
		return common.Hash{}, err
	}

	return common.NewHashFromGeneric((*header).StateRoot()), nil
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) Entries(bhash *common.Hash) (map[string][]byte, error) {
	stateAt, err := ca.stateAt(bhash)
	if err != nil {
		return nil, err
	}

	entries := make(map[string][]byte)
	iter, err := stateAt.Pairs(statemachine.IterArgs{})
	if err != nil {
		return nil, err
	}

	for kv, err := range iter.All() {
		if err != nil {
			return nil, err
		}
		entries[string(kv.StorageKey)] = kv.StorageValue
	}

	return entries, nil
}

// GetKeysWithPrefix returns all keys with the given prefix from the state
// that corresponds to the given block hash.
func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetKeysWithPrefix(
	bhash *common.Hash,
	prefix []byte,
) ([][]byte, error) {
	stateAt, err := ca.stateAt(bhash)
	if err != nil {
		return nil, err
	}

	keys, err := stateAt.Keys(statemachine.IterArgs{Prefix: prefix})
	if err != nil {
		return nil, err
	}

	var res [][]byte
	for key, err := range keys.All() {
		if err != nil {
			return nil, err
		}

		res = append(res, bytes.Clone(key))
	}
	return res, nil
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetStorageChild(
	bhash *common.Hash,
	keyToChild []byte,
) (trie.Trie, error) {
	stateAt, err := ca.stateAt(bhash)
	if err != nil {
		return nil, err
	}

	memDB := db.NewEmptyMemoryDB()
	storageTrie := inmemory.NewTrie(nil, memDB)

	iterArgs := statemachine.IterArgs{
		ChildInfo: storage.NewDefaultChildInfo(keyToChild),
	}

	iter, err := stateAt.Pairs(iterArgs)
	if err != nil {
		return nil, err
	}

	for kv, err := range iter.All() {
		if err != nil {
			return nil, err
		}

		err = storageTrie.Put(kv.StorageKey, kv.StorageValue)
		if err != nil {
			return nil, err
		}
	}
	return storageTrie, nil
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) GetStorageFromChild(
	bhash *common.Hash,
	keyToChild []byte,
	key []byte,
) ([]byte, error) {
	hash := ca.hashToGeneric(bhash)
	info := storage.NewDefaultChildInfo(keyToChild)

	return ca.client.ChildStorage(hash, info, key)
}

// LoadCode returns the runtime blob for the given block hash.
func (ca *ClientAdapter[H, Hasher, N, E, Header]) LoadCode(bhash *common.Hash) ([]byte, error) {
	return ca.GetStorage(bhash, common.CodeKey)
}

// LoadCodeHash returns the hash of the runtime blob for the given block hash.
func (ca *ClientAdapter[H, Hasher, N, E, Header]) LoadCodeHash(bhash *common.Hash) (common.Hash, error) {
	code, err := ca.LoadCode(bhash)
	if err != nil {
		return common.Hash{}, err
	}

	hasher := *new(Hasher)
	codeHash := hasher.NewHash(code)

	return common.NewHashFromGeneric(codeHash), nil
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

func (ca *ClientAdapter[H, Hasher, N, E, Header]) stateAt(bhash *common.Hash) (statemachine.Backend[H, Hasher], error) {
	return ca.client.StateAt(ca.hashToGeneric(bhash))
}

func (ca *ClientAdapter[H, Hasher, N, E, Header]) hashToGeneric(bhash *common.Hash) H {
	if bhash == nil {
		return ca.client.Info().BestHash
	}

	hasher := *new(Hasher)
	return hasher.NewHash(bhash.ToBytes())
}

func prefixKey(hash common.Hash, prefix []byte) []byte {
	return append(prefix, hash.ToBytes()...)
}
