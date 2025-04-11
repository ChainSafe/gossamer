// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"errors"
	"fmt"
	"time"

	"github.com/ChainSafe/gossamer/internal/client/api"
	"github.com/ChainSafe/gossamer/internal/client/consensus/common"
	"github.com/ChainSafe/gossamer/internal/client/keystore"
	"github.com/ChainSafe/gossamer/internal/client/network"
	"github.com/ChainSafe/gossamer/internal/client/network/role"
	peerid "github.com/ChainSafe/gossamer/internal/client/network/types/peer-id"
	"github.com/ChainSafe/gossamer/internal/log"
	papi "github.com/ChainSafe/gossamer/internal/primitives/api"
	"github.com/ChainSafe/gossamer/internal/primitives/blockchain"
	primitives "github.com/ChainSafe/gossamer/internal/primitives/consensus/grandpa"
	"github.com/ChainSafe/gossamer/internal/primitives/core/crypto"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	grandpa "github.com/ChainSafe/gossamer/pkg/finality-grandpa"
)

var logger = log.NewFromGlobal(log.AddContext("consensus", "grandpa"))

// A global communication input stream for commits and catch up messages. Not exposed publicly, used internally to
// simplify types in the communication layer.
type communicationIn[H runtime.Hash, N runtime.Number] grandpa.CommunicationIn[
	H, N, primitives.AuthoritySignature, primitives.AuthorityID]

// Global communication sink for commits with the hash type not being derived from the block, useful for forcing the
// hash to some type (e.g. `H256`) when the compiler can't do the inference.
type communicationOut[H runtime.Hash, N runtime.Number] grandpa.CommunicationOut[ //nolint: unused
	H, N, primitives.AuthoritySignature, primitives.AuthorityID]

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

/// Errors that can occur while voting in GRANDPA.
// #[derive(Debug, thiserror::Error)]
// pub enum Error {
// 	/// An error within grandpa.
// 	#[error("grandpa error: {0}")]
// 	Grandpa(#[from] GrandpaError),

// 	/// A network error.
// 	#[error("network error: {0}")]
// 	Network(String),

// 	/// A blockchain error.
// 	#[error("blockchain error: {0}")]
// 	Blockchain(String),

// /// Could not complete a round on disk.
// #[error("could not complete a round on disk: {0}")]
// Client(#[from] ClientError),
var ErrClient = errors.New("could not complete a round on disk")

// 	/// Could not sign outgoing message
// 	#[error("could not sign outgoing message: {0}")]
// 	Signing(String),

// /// An invariant has been violated (e.g. not finalizing pending change blocks in-order)
// #[error("safety invariant has been violated: {0}")]
// Safety(String),
var ErrSafety = errors.New("safety invariant has been violated")

// 	/// A timer failed to fire.
// 	#[error("a timer failed to fire: {0}")]
// 	Timer(io::Error),

// /// A runtime api request failed.
// #[error("runtime API request failed: {0}")]
// RuntimeApi(sp_api::ApiError),
var ErrRuntimeApi = errors.New("runtime API request failed")

// }

// / Something which can determine if a block is known.
// pub(crate) trait BlockStatus<Block: BlockT> {
type BlockStatus[H runtime.Hash, N runtime.Number] interface {
	/// Return `Ok(Some(number))` or `Ok(None)` depending on whether the block
	/// is definitely known and has been imported.
	/// If an unexpected error occurs, return that.
	// fn block_number(&self, hash: Block::Hash) -> Result<Option<NumberFor<Block>>, Error>;
	Number(hash H) (*N, error)
}

// impl<Block: BlockT, Client> BlockStatus<Block> for Arc<Client>
// where
//
//	Client: HeaderBackend<Block>,
//	NumberFor<Block>: BlockNumberOps,
//
//	{
//		fn block_number(&self, hash: Block::Hash) -> Result<Option<NumberFor<Block>>, Error> {
//			self.block_number_from_id(&BlockId::Hash(hash))
//				.map_err(|e| Error::Blockchain(e.to_string()))
//		}
//	}
type BlockStatusForClient[H runtime.Hash, N runtime.Number, Header runtime.Header[N, H]] struct {
	blockchain.HeaderBackend[H, N, Header]
}

func (bsfc BlockStatusForClient[H, N, Header]) BlockNumber(hash H) (*N, error) {
	return bsfc.HeaderBackend.Number(hash)
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
	papi.ProvideRuntimeAPI[primitives.GrandpaAPI[H, N]]
	api.ExecutorProvider
	common.BlockImport[H, N]
	api.StorageProvider[H, N, Hasher]
}

// / Something that one can ask to do a block sync request.
// pub(crate) trait BlockSyncRequester<Block: BlockT> {
type BlockSyncRequester[H runtime.Hash, N runtime.Number] interface {
	/// Notifies the sync service to try and sync the given block from the given
	/// peers.
	///
	/// If the given vector of peers is empty then the underlying implementation
	/// should make a best effort to fetch the block from any peers it is
	/// connected to (NOTE: this assumption will change in the future #3629).
	SetSyncForkRequest(peers []peerid.PeerID, hash H, number N)
}

// / A new authority set along with the canonical block it changed at.
type newAuthoritySet[H, N any] struct {
	CanonNumber N
	CanonHash   H
	SetID       primitives.SetID
	Authorities primitives.AuthorityList
}

// / Commands issued to the voter.
type voterCommand interface {
	Error() string
}

// / Pause the voter for given reason.
type voterCommandPause string

func (vcp voterCommandPause) Error() string {
	return fmt.Sprintf("Pausing voter: %s", string(vcp))
}

// / New authorities.
type voterCommandChangeAuthorities[H, N any] newAuthoritySet[H, N]

func (vcca voterCommandChangeAuthorities[H, N]) Error() string {
	return fmt.Sprintf("Changing authorities")
}

// / Checks if this node has any available keys in the keystore for any authority id in the given
// / voter set.  Returns the authority id for which keys are available, or `None` if no keys are
// / available.
// fn local_authority_id(
//
//	voters: &VoterSet<AuthorityId>,
//	keystore: Option<&KeystorePtr>,
//
// ) -> Option<AuthorityId> {
func localAuthorityID(voters grandpa.VoterSet[primitives.AuthorityID], ks *keystore.KeyStore) *primitives.AuthorityID {
	if ks == nil {
		return nil
	}

	for _, voter := range voters.Voters() {
		if (*ks).HasKeys([]keystore.PublicKey{{
			Key:       voter.ID.Bytes(),
			KeyTypeID: crypto.GRANDPA,
		}}) {
			return &voter.ID
		}
	}
	return nil
}

// 	keystore.and_then(|keystore| {
// 		voters
// 			.iter()
// 			.find(|(p, _)| keystore.has_keys(&[(p.to_raw_vec(), AuthorityId::ID)]))
// 			.map(|(p, _)| p.clone())
// 	})
// }
