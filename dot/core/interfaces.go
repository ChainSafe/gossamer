// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package core

import (
	"encoding/json"
	"sync"

	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/transaction"
)

// RuntimeInstance for runtime methods
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
	Version() (version runtime.Version)
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
	PaymentQueryInfo(ext []byte) (*types.RuntimeDispatchInfo, error)
	CheckInherents()
	RandomSeed()
	OffchainWorker()
	GenerateSessionKeys()
}

// BlockState interface for block state methods
type BlockState interface {
	BestBlockHash() common.Hash
	BestBlockHeader() (*types.Header, error)
	AddBlock(*types.Block) error
	GetBlockStateRoot(bhash common.Hash) (common.Hash, error)
	SubChain(start, end common.Hash) ([]common.Hash, error)
	GetBlockBody(hash common.Hash) (*types.Body, error)
	HandleRuntimeChanges(newState *rtstorage.TrieState, in state.Runtime, bHash common.Hash) error
	GetRuntime(blockHash common.Hash) (instance state.Runtime, err error)
	StoreRuntime(blockHash common.Hash, runtime state.Runtime)
	LowestCommonAncestor(a, b common.Hash) (common.Hash, error)
}

// StorageState interface for storage state methods
type StorageState interface {
	TrieState(root *common.Hash) (*rtstorage.TrieState, error)
	StoreTrie(*rtstorage.TrieState, *types.Header) error
	GetStateRootFromBlock(bhash *common.Hash) (*common.Hash, error)
	GenerateTrieProof(stateRoot common.Hash, keys [][]byte) ([][]byte, error)
	sync.Locker
}

// TransactionState is the interface for transaction state methods
type TransactionState interface {
	Push(vt *transaction.ValidTransaction) (common.Hash, error)
	AddToPool(vt *transaction.ValidTransaction) common.Hash
	RemoveExtrinsic(ext types.Extrinsic)
	RemoveExtrinsicFromPool(ext types.Extrinsic)
	PendingInPool() []*transaction.ValidTransaction
	Exists(ext types.Extrinsic) bool
}

// Network is the interface for the network service
type Network interface {
	GossipMessage(network.NotificationsMessage)
	IsSynced() bool
	ReportPeer(change peerset.ReputationChange, p peer.ID)
}

// CodeSubstitutedState interface to handle storage of code substitute state
type CodeSubstitutedState interface {
	StoreCodeSubstitutedBlockHash(hash common.Hash) error
}

// Telemetry is the telemetry client to send telemetry messages.
type Telemetry interface {
	SendMessage(msg json.Marshaler)
}

// KeyPair is a key pair to sign messages and from which
// the public key and key type can be obtained.
type KeyPair interface {
	Type() crypto.KeyType
	Sign(msg []byte) ([]byte, error)
	Public() crypto.PublicKey
}
