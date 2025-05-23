// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"errors"
	"fmt"
	"time"

	"github.com/ChainSafe/gossamer/internal/client/api"
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

// Errors that can occur while voting in GRANDPA.
var (
	// ErrClient means we could not complete a round on disk.
	ErrClient = errors.New("could not complete a round on disk")

	// ErrSafety means an invariant has been violated (e.g. not finalizing pending change blocks in-order)
	ErrSafety = errors.New("safety invariant has been violated")

	// ErrRuntimeApi means a runtime api request failed.
	ErrRuntimeApi = errors.New("runtime API request failed")
)

// Something which can determine if a block is known.
type BlockStatus[H runtime.Hash, N runtime.Number] interface {
	// Return a number or nil depending on whether the block is definitely known and has been imported. If an
	// unexpected error occurs, return that.
	Number(hash H) (*N, error)
}

// ClientForGrandpa is an interface that includes all the client functionalities grandpa requires.
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
	papi.ProvideRuntimeApi[primitives.GrandpaAPI[H, N]]
	// api.ExecutorProvider
	// common.BlockImport[H, N]
	// api.StorageProvider[H, N, Hasher]
}

// Something that one can ask to do a block sync request.
type BlockSyncRequester[H runtime.Hash, N runtime.Number] interface {
	// Notifies the sync service to try and sync the given block from the given peers.
	//
	// If the given vector of peers is empty then the underlying implementation should make a best effort to fetch
	// the block from any peers it is connected to (NOTE: this assumption will change in the future substrate #3629).
	SetSyncForkRequest(peers []peerid.PeerID, hash H, number N)
}

// A new authority set along with the canonical block it changed at.
type newAuthoritySet[H, N any] struct {
	CanonNumber N
	CanonHash   H
	SetID       primitives.SetID
	Authorities primitives.AuthorityList
}

// Commands issued to the voter.
type voterCommand interface {
	Error() string
}

// Pause the voter for given reason.
type voterCommandPause string //nolint: unused

func (vcp voterCommandPause) Error() string { //nolint: unused
	return fmt.Sprintf("Pausing voter: %s", string(vcp))
}

// New authorities.
type voterCommandChangeAuthorities[H, N any] newAuthoritySet[H, N]

func (vcca voterCommandChangeAuthorities[H, N]) Error() string {
	return "Changing authorities"
}

// Checks if this node has any available keys in the keystore for any authority id in the givenvoter set.  Returns the
// authority id for which keys are available, or nil if no keys are available.
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
