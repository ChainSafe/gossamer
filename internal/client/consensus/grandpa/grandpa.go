// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"time"

	"github.com/ChainSafe/gossamer/internal/client/api"
	"github.com/ChainSafe/gossamer/internal/client/consensus"
	"github.com/ChainSafe/gossamer/internal/client/keystore"
	"github.com/ChainSafe/gossamer/internal/client/network"
	"github.com/ChainSafe/gossamer/internal/client/network/role"
	"github.com/ChainSafe/gossamer/internal/log"
	papi "github.com/ChainSafe/gossamer/internal/primitives/api"
	"github.com/ChainSafe/gossamer/internal/primitives/blockchain"
	pgrandpa "github.com/ChainSafe/gossamer/internal/primitives/consensus/grandpa"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	grandpa "github.com/ChainSafe/gossamer/pkg/finality-grandpa"
)

var logger = log.NewFromGlobal(log.AddContext("consensus", "grandpa"))

// A global communication input stream for commits and catch up messages. Not exposed publicly, used internally to
// simplify types in the communication layer.
type communicationIn[H runtime.Hash, N runtime.Number] grandpa.CommunicationIn[
	H, N, pgrandpa.AuthoritySignature, pgrandpa.AuthorityID]

// Global communication sink for commits with the hash type not being derived from the block, useful for forcing the
// hash to some type (e.g. `H256`) when the compiler can't do the inference.
type communicationOut[H runtime.Hash, N runtime.Number] grandpa.CommunicationOut[ //nolint: unused
	H, N, pgrandpa.AuthoritySignature, pgrandpa.AuthorityID]

// newAuthoritySet A new authority set along with the canonical block it changed at.
type newAuthoritySet[H, N any] struct { //nolint: unused
	CanonNumber N
	CanonHash   H
	SetID       pgrandpa.SetID
	Authorities pgrandpa.AuthorityList
}
type Config struct {
	// The expected duration for a message to be gossiped across the network.
	GossipDuration time.Duration
	// Justification generation period (in blocks). GRANDPA will try to generate justifications at least every
	// justification_period blocks. There are some other events which might cause justification generation.
	JustificationGenerationPeriod uint32
	// Whether the GRANDPA observer protocol is live on the network and thereby a full-node not running as a validator
	// is running the GRANDPA observer protocol (we will only issue catch-up requests to authorities when the observer
	// protocol is enabled).
	ObserverEnabled bool
	// The role of the local node (i.e. authority, full-node or light).
	LocalRole role.Role
	// Some local identifier of the voter.
	Name *string
	// The keystore that manages the keys of this node.
	KeyStore keystore.KeyStore // can be nil for optionality
	// Chain specific GRANDPA protocol name.
	ProtocolName network.ProtocolName
	// TODO: telemetry
}

func (c Config) name() string {
	if c.Name == nil {
		return "<unknown>"
	}
	return *c.Name
}

// / A trait that includes all the client functionalities grandpa requires.
// / Ideally this would be a trait alias, we're not there yet.
// / tracking issue <https://github.com/rust-lang/rust/issues/41517>
// pub trait ClientForGrandpa<Block, BE>:
//
//	LockImportRun<Block, BE>
//	+ Finalizer<Block, BE>
//	+ AuxStore
//	+ HeaderMetadata<Block, Error = sp_blockchain::Error>
//	+ HeaderBackend<Block>
//	+ BlockchainEvents<Block>
//	+ ProvideRuntimeApi<Block>
//	+ ExecutorProvider<Block>
//	+ BlockImport<Block, Transaction = TransactionFor<BE, Block>, Error = sp_consensus::Error>
//	+ StorageProvider<Block, BE>
//
// where
//
//	BE: Backend<Block>,
//	Block: BlockT,
//
// {}
type ClientForGrandpa[
	H runtime.Hash,
	N runtime.Number,
	Hasher runtime.Hasher[H],
	Header runtime.Header[N, H],
	E runtime.Extrinsic,
] interface {
	api.LockImportRun[H, N, Hasher, Header, E]
	api.Finalizer[H, N, Hasher, Header, E]
	api.AuxStore
	blockchain.HeaderMetadata[H, N]
	blockchain.HeaderBackend[H, N, Header]
	api.BlockchainEvents[H, N, Header]
	papi.ProvideRuntimeAPI
	api.ExecutorProvider
	consensus.BlockImport[H, N]
	api.StorageProvider[H, N, Hasher]
}
