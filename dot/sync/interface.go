// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"sync"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/transaction"
	"github.com/libp2p/go-libp2p-core/peer"
)

// RuntimeInstance is the interface for runtime Instance
type RuntimeInstance interface {
	UpdateRuntimeCode([]byte) error
	Stop()
	NodeStorage() runtime.NodeStorage
	NetworkService() runtime.BasicNetwork
	Keystore() *keystore.GlobalKeystore
	Validator() bool
	Exec(function string, data []byte) ([]byte, error)
	SetContextStorage(s runtime.Storage)
	GetCodeHash() common.Hash
	Version() runtime.Version
	Metadata() ([]byte, error)
	BabeConfiguration() (*types.BabeConfiguration, error)
	GrandpaAuthorities() ([]types.Authority, error)
	ValidateTransaction(e types.Extrinsic) (*transaction.Validity, error)
	InitializeBlock(header *types.Header) error
	InherentExtrinsics(data []byte) ([]byte, error)
	ApplyExtrinsic(data types.Extrinsic) ([]byte, error)
	FinalizeBlock() (*types.Header, error)
	ExecuteBlock(block *types.Block) ([]byte, error)
	DecodeSessionKeys(enc []byte) ([]byte, error)
	PaymentQueryInfo(ext []byte) (*types.TransactionPaymentQueryInfo, error)
	CheckInherents()
	RandomSeed()
	OffchainWorker()
	GenerateSessionKeys()
}

// BlockState is the interface for the block state
type BlockState interface {
	BestBlockHash() common.Hash
	BestBlockHeader() (*types.Header, error)
	BestBlockNumber() (number uint, err error)
	AddBlock(*types.Block) error
	CompareAndSetBlockData(bd *types.BlockData) error
	GetBlockByNumber(blockNumber uint) (*types.Block, error)
	HasBlockBody(hash common.Hash) (bool, error)
	GetBlockBody(common.Hash) (*types.Body, error)
	SetHeader(*types.Header) error
	GetHeader(common.Hash) (*types.Header, error)
	HasHeader(hash common.Hash) (bool, error)
	SubChain(start, end common.Hash) ([]common.Hash, error)
	GetReceipt(common.Hash) ([]byte, error)
	GetMessageQueue(common.Hash) ([]byte, error)
	GetJustification(common.Hash) ([]byte, error)
	SetJustification(hash common.Hash, data []byte) error
	SetFinalisedHash(hash common.Hash, round, setID uint64) error
	AddBlockToBlockTree(block *types.Block) error
	GetHashByNumber(blockNumber uint) (common.Hash, error)
	GetBlockByHash(common.Hash) (*types.Block, error)
	GetBestBlockRuntime() (instance runtime.Instance)
	GetRuntime(hash *common.Hash) (runtime.Instance, error)
	StoreRuntime(common.Hash, runtime.Instance)
	GetHighestFinalisedHeader() (*types.Header, error)
	GetFinalisedNotifierChannel() chan *types.FinalisationInfo
	GetHeaderByNumber(num uint) (*types.Header, error)
	GetAllBlocksAtNumber(num uint) ([]common.Hash, error)
	IsDescendantOf(parent, child common.Hash) (bool, error)
}

// StorageState is the interface for the storage state
type StorageState interface {
	TrieState(root *common.Hash) (*rtstorage.TrieState, error)
	LoadCodeHash(*common.Hash) (common.Hash, error)
	GetStateRootFromBlock(bhash *common.Hash) (*common.Hash, error)
	sync.Locker
}

// CodeSubstitutedState interface to handle storage of code substitute state
type CodeSubstitutedState interface {
	LoadCodeSubstitutedBlockHash() common.Hash
	StoreCodeSubstitutedBlockHash(hash common.Hash) error
}

// TransactionState is the interface for transaction queue methods
type TransactionState interface {
	RemoveExtrinsic(ext types.Extrinsic)
}

// BabeVerifier deals with BABE block verification
type BabeVerifier interface {
	VerifyBlock(header *types.Header) error
}

// FinalityGadget implements justification verification functionality
type FinalityGadget interface {
	VerifyBlockJustification(common.Hash, []byte) ([]byte, error)
}

// BlockImportHandler is the interface for the handler of newly imported blocks
type BlockImportHandler interface {
	HandleBlockImport(block *types.Block, state *rtstorage.TrieState) error
}

// Network is the interface for the network
type Network interface {
	// DoBlockRequest sends a request to the given peer.
	// If a response is received within a certain time period,
	// it is returned, otherwise an error is returned.
	DoBlockRequest(to peer.ID, req *network.BlockRequestMessage) (*network.BlockResponseMessage, error)

	// Peers returns a list of currently connected peers
	Peers() []common.PeerInfo

	// ReportPeer reports peer based on the peer behaviour.
	ReportPeer(change peerset.ReputationChange, p peer.ID)
}
