// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/internal/client/api"
	client_common "github.com/ChainSafe/gossamer/internal/client/consensus/common"
	"github.com/ChainSafe/gossamer/internal/client/keystore"
	"github.com/ChainSafe/gossamer/internal/client/network"
	"github.com/ChainSafe/gossamer/internal/client/network/role"
	"github.com/ChainSafe/gossamer/internal/client/network/service"
	peerid "github.com/ChainSafe/gossamer/internal/client/network/types/peer-id"
	"github.com/ChainSafe/gossamer/internal/log"
	papi "github.com/ChainSafe/gossamer/internal/primitives/api"
	"github.com/ChainSafe/gossamer/internal/primitives/blockchain"
	"github.com/ChainSafe/gossamer/internal/primitives/consensus/common"
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

// Shared voter state for querying.
type SharedVoterState[AuthorityID comparable] struct {
	inner grandpa.VoterState[AuthorityID]
	sync.RWMutex
}

func (svs *SharedVoterState[AuthorityID]) reset(voterState grandpa.VoterState[AuthorityID]) {
	svs.Lock()
	defer svs.Unlock()
	svs.inner = voterState
}

type Config struct {
	// The expected duration for a message to be gossiped across the network.
	GossipDuration time.Duration
	// Justification generation period (in blocks). GRANDPA will try to generate justifications at least every
	// justification_period blocks. There are some other events which might cause justification generation.
	JustificationGenerationPeriod uint32
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
	papi.ProvideRuntimeAPI[primitives.GrandpaAPI[H, N]]
	// api.ExecutorProvider
	client_common.BlockImport[H, N, E, Header]
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
	isVoterCommand()
}

// Pause the voter for given reason.
type voterCommandPause string

func (vcp voterCommandPause) Error() string {
	return fmt.Sprintf("Pausing voter: %s", string(vcp))
}
func (vcp voterCommandPause) isVoterCommand() {}

// New authorities.
type voterCommandChangeAuthorities[H, N any] newAuthoritySet[H, N]

func (vcca voterCommandChangeAuthorities[H, N]) Error() string {
	return "Changing authorities"
}
func (voterCommandChangeAuthorities[H, N]) isVoterCommand() {}

// Link between the block importer and the background voter.
type LinkHalf[
	H runtime.Hash,
	N runtime.Number,
	Hasher runtime.Hasher[H],
	Header runtime.Header[N, H],
	E runtime.Extrinsic,
] struct {
	client              ClientForGrandpa[H, N, Hasher, Header, E]
	selectChain         common.SelectChain[H, N, Header]
	persistentData      persistentData[H, N]
	voterCommandsRx     chan voterCommand
	justificationSender GrandpaJustificationSender[H, N, Header]
	justificationStream GrandpaJustificationStream[H, N, Header] //nolint: unused
}

// Provider for the Grandpa authority set configured on the genesis block.
type GenesisAuthoritySetProvider interface {
	// Get the authority set at the genesis block.
	Get() (primitives.AuthorityList, error)
}

func globalCommunication[
	H runtime.Hash,
	N runtime.Number,
	Hasher runtime.Hasher[H],
	Header runtime.Header[N, H],
	E runtime.Extrinsic,
](
	setID primitives.SetID,
	voters grandpa.VoterSet[primitives.AuthorityID],
	client ClientForGrandpa[H, N, Hasher, Header, E],
	network *networkBridge[H, N, Hasher],
	keystore keystore.KeyStore,
	// TODO: metrics
) (chan grandpa.GlobalInItem[H, N, primitives.AuthoritySignature, primitives.AuthorityID], commitsOut[H, N, Hasher]) {
	isVoter := localAuthorityID(voters, keystore) != nil

	in, out := network.globalCommunication(SetID(setID), voters, isVoter)

	// block commit and catch up messages until relevant blocks are imported.
	globalIn := newUntilGlobalMessageBlocksImported(
		client.RegisterImportNotificationStream(),
		network,
		client,
		in,
		"global",
	)

	mappedIn := make(chan grandpa.GlobalInItem[H, N, primitives.AuthoritySignature, primitives.AuthorityID])
	go func() {
		defer close(mappedIn)
		for item := range globalIn.Chan() {
			mappedIn <- grandpa.GlobalInItem[H, N, primitives.AuthoritySignature, primitives.AuthorityID]{
				CommunicationIn: item.Blocked,
				Error:           item.Error,
			}
		}
	}()

	return mappedIn, out
}

// Parameters used to run Grandpa.
type GrandpaParams[
	H runtime.Hash,
	N runtime.Number,
	Hasher runtime.Hasher[H],
	Header runtime.Header[N, H],
	E runtime.Extrinsic,
] struct {
	// Configuration for the GRANDPA service.
	Config Config
	// A link to the block import worker.
	LinkHalf LinkHalf[H, N, Hasher, Header, E]
	// The Network instance.
	// It is assumed that this network will feed us Grandpa notifications.
	Network Network
	// Event stream for syncing-related events.
	Sync Syncing[H, N]
	// Handle for interacting with `Notifications`.
	NotificationService service.NotificationService
	// A voting rule used to potentially restrict target votes.
	VotingRule VotingRule[H, N, Header]
	// The voter state is exposed at an RPC endpoint.
	SharedVoterState *SharedVoterState[primitives.AuthorityID]

	// TODO: telemetry, metrics
}

// Run a GRANDPA voter as a task. Provide configuration and a link to a
// block import worker that has already been instantiated with [client_common.BlockImport].
func RunGrandpaVoter[
	H runtime.Hash,
	N runtime.Number,
	Hasher runtime.Hasher[H],
	Header runtime.Header[N, H],
	E runtime.Extrinsic,
](
	grandpaParams GrandpaParams[H, N, Hasher, Header, E],
) (done chan struct{}, err error) {
	var (
		config              = grandpaParams.Config
		link                = grandpaParams.LinkHalf
		network             = grandpaParams.Network
		sync                = grandpaParams.Sync
		notificationService = grandpaParams.NotificationService
		votingRule          = grandpaParams.VotingRule
		sharedVoterState    = grandpaParams.SharedVoterState
	)

	var (
		client              = link.client
		selectChain         = link.selectChain
		persistentData      = link.persistentData
		voterCommandsRx     = link.voterCommandsRx
		justificationSender = link.justificationSender
	)

	networkBridge := newNetworkBridge[H, N, Hasher](
		network,
		sync,
		notificationService,
		config,
		persistentData.setState,
	)

	// TODO: telemetry

	voterWork := newVoterWork[H, N, Hasher, Header, E](
		client,
		config,
		networkBridge,
		selectChain,
		votingRule,
		persistentData,
		voterCommandsRx,
		sharedVoterState,
		justificationSender,
	)
	done = make(chan struct{})
	go func() {
		defer close(done)
		err := voterWork.run()
		if err != nil {
			logger.Errorf("GRANDPA voter error: %v", err)
			return
		}
		logger.Error("GRANDPA voter future has concluded naturally, this should be unreachable.")
	}()

	return done, nil
}

// Future that powers the voter.
type voterWork[
	H runtime.Hash,
	N runtime.Number,
	Hasher runtime.Hasher[H],
	Header runtime.Header[N, H],
	E runtime.Extrinsic,
] struct {
	// use string for AuthorityID and AuthoritySignature
	voter            *grandpa.Voter[H, N, primitives.AuthoritySignature, primitives.AuthorityID]
	voterErrChan     <-chan error
	sharedVoterState *SharedVoterState[primitives.AuthorityID]
	env              *environment[H, N, Hasher, Header, E]
	voterCommandsRx  <-chan voterCommand
	network          *networkBridge[H, N, Hasher]
	// TODO: telemtry, metrics
}

func newVoterWork[
	H runtime.Hash,
	N runtime.Number,
	Hasher runtime.Hasher[H],
	Header runtime.Header[N, H],
	E runtime.Extrinsic,
](
	client ClientForGrandpa[H, N, Hasher, Header, E],
	config Config,
	network *networkBridge[H, N, Hasher],
	selectChain common.SelectChain[H, N, Header],
	votingRule VotingRule[H, N, Header],
	persistentData persistentData[H, N],
	voterCommandsRx <-chan voterCommand,
	sharedVoterState *SharedVoterState[primitives.AuthorityID],
	justificationSender GrandpaJustificationSender[H, N, Header],
	// TODO: telemetry
) voterWork[H, N, Hasher, Header, E] {
	// TODO: register to prometheus registry

	voters := persistentData.authoritySet.CurrentAuthorities()
	env := environment[H, N, Hasher, Header, E]{
		Client:              client,
		SelectChain:         selectChain,
		VotingRule:          votingRule,
		Voters:              voters,
		Config:              config,
		Network:             network,
		SetID:               SetID(persistentData.authoritySet.SetID()),
		AuthoritySet:        persistentData.authoritySet,
		VoterSetState:       persistentData.setState,
		JustificationSender: &justificationSender,
	}

	work := voterWork[H, N, Hasher, Header, E]{
		// voter is set to a temporary value and replaced below when calling rebuildVoter().
		voter:            nil,
		sharedVoterState: sharedVoterState,
		env:              &env,
		voterCommandsRx:  voterCommandsRx,
		network:          network,
	}
	work.rebuildVoter()
	return work
}

// Rebuilds the voter field using the current authority set
// state. This method should be called when we know that the authority set
// has changed (e.g. as signalled by a voter command).
func (vw *voterWork[H, N, Hasher, Header, E]) rebuildVoter() {
	logger.Debugf("%s: Starting new voter with set ID %v", vw.env.Config.name(), vw.env.SetID)

	maybeAuthorityID := localAuthorityID(vw.env.Voters, vw.env.Config.KeyStore)
	var authorityID string
	if maybeAuthorityID != nil {
		authorityID = string(maybeAuthorityID.Bytes())
	} else {
		authorityID = "<unknown>"
	}

	// TODO: telemetry afg.starting_new_voter

	chainInfo := vw.env.Client.Info()

	// TODO: telemetry afg.authority_set
	_ = authorityID
	_ = chainInfo

	vw.env.VoterSetState.innerMtx.RLock()
	defer vw.env.VoterSetState.innerMtx.RUnlock()

	switch vss := vw.env.VoterSetState.inner.(type) {
	case voterSetStateLive[H, N]:
		var lastFinalized = grandpa.HashNumber[H, N]{
			Hash:   chainInfo.FinalizedHash,
			Number: chainInfo.FinalizedNumber,
		}
		globalIn, globalOut := globalCommunication(
			primitives.SetID(vw.env.SetID),
			vw.env.Voters,
			vw.env.Client,
			vw.env.Network,
			vw.env.Config.KeyStore,
		)

		lastCompletedRound := vss.completedRounds().last()

		votes := make([]grandpa.SignedMessage[H, N, primitives.AuthoritySignature, primitives.AuthorityID],
			len(lastCompletedRound.Votes))
		for i, vote := range lastCompletedRound.Votes {
			votes[i] = grandpa.SignedMessage[H, N, primitives.AuthoritySignature, primitives.AuthorityID]{
				Signature: vote.Signature,
				Message:   vote.Message,
				ID:        vote.ID,
			}
		}
		voter := grandpa.NewVoter[H, N, primitives.AuthoritySignature, primitives.AuthorityID](
			vw.env,
			vw.env.Voters,
			globalIn,
			globalOut.preSend,
			uint64(lastCompletedRound.Number),
			votes,
			lastCompletedRound.Base,
			lastFinalized,
		)

		// Repoint shared_voter_state so that the RPC endpoint can query the state
		vw.sharedVoterState.reset(voter.VoterState())

		vw.voter = voter
		errChan := make(chan error)
		go func() {
			err := voter.Start()
			errChan <- err
			close(errChan)
		}()
		vw.voterErrChan = errChan
	case voterSetStatePaused[H, N]:
	default:
		panic("unreachable")
	}
}

func (vw *voterWork[H, N, Hasher, Header, E]) handleVoterCommand(command voterCommand) error {
	switch command := command.(type) {
	case voterCommandChangeAuthorities[H, N]:
		new := command
		// TODO: telemetry afg.voter_command_change_authorities

		err := vw.env.updateVoterSetState(func(voterSetState voterSetState[H, N]) (voterSetState[H, N], error) {
			// start the new authority set using the block where the
			// set changed (not where the signal happened!) as the base.
			authoritySet, unlock := vw.env.AuthoritySet.inner.DataMut()
			defer unlock()
			setState := newVoterSetStateLive(
				primitives.SetID(vw.env.SetID),
				*authoritySet,
				grandpa.HashNumber[H, N]{
					Hash:   command.CanonHash,
					Number: command.CanonNumber,
				},
			)
			err := writeVoterSetState(vw.env.Client, &setState)
			if err != nil {
				return nil, err
			}
			return setState, nil
		})
		if err != nil {
			return err
		}

		authorities := make([]grandpa.IDWeight[primitives.AuthorityID], len(new.Authorities))
		for i, authority := range new.Authorities {
			authorities[i] = grandpa.IDWeight[primitives.AuthorityID]{
				ID:     authority.AuthorityID,
				Weight: uint64(authority.AuthorityWeight),
			}
		}
		voters := grandpa.NewVoterSet(authorities)
		if voters == nil {
			panic("new authorities come from pending change; pending change comes from AuthoritySet;" +
				"AuthoritySet validates authorities is non-empty and weights are non-zero")
		}

		vw.env = &environment[H, N, Hasher, Header, E]{
			Voters:              *voters,
			SetID:               SetID(new.SetID),
			VoterSetState:       vw.env.VoterSetState,
			Client:              vw.env.Client,
			SelectChain:         vw.env.SelectChain,
			Config:              vw.env.Config,
			AuthoritySet:        vw.env.AuthoritySet,
			Network:             vw.env.Network,
			VotingRule:          vw.env.VotingRule,
			JustificationSender: vw.env.JustificationSender,
		}

		vw.rebuildVoter()
		return nil
	case voterCommandPause:
		logger.Infof("Pausing old validator set: %s", string(command))

		// not racing because old voter is shut down.
		err := vw.env.updateVoterSetState(func(voterSetState voterSetState[H, N]) (voterSetState[H, N], error) {
			completedRounds := voterSetState.completedRounds()
			setState := voterSetStatePaused[H, N]{CompletedRounds: completedRounds}
			err := writeVoterSetState(vw.env.Client, setState)
			if err != nil {
				return nil, err
			}
			return setState, nil
		})
		if err != nil {
			return err
		}

		vw.rebuildVoter()
		return nil
	default:
		panic("unreachable")
	}
}

func (vw *voterWork[H, N, Hasher, Header, E]) poll() error {
	select {
	case err := <-vw.voterErrChan:
		if err == nil {
			// voters don't conclude naturally
			return fmt.Errorf("consensus-grandpa inner voter has concluded: %w", ErrSafety)
		}
		vc, isVoterCommand := err.(voterCommand)
		if !isVoterCommand {
			// return inner observer error
			return err
		}
		// some command issued internally
		return vw.handleVoterCommand(vc)
	default:
	}

	select {
	case vc, ok := <-vw.voterCommandsRx:
		if !ok {
			// the voterCommandsRx stream should never conclude since it's never closed.
			return fmt.Errorf("`%w: voter_commands_rx` was closed", ErrSafety)
		}
		// some command issued externally
		return vw.handleVoterCommand(vc)
	default:
	}
	return nil
}

func (vw *voterWork[H, N, Hasher, Header, E]) run() error {
	for {
		err := vw.poll()
		if err != nil {
			return err
		}
	}
}

// Checks if this node has any available keys in the keystore for any authority id in the givenvoter set.  Returns the
// authority id for which keys are available, or nil if no keys are available.
func localAuthorityID(voters grandpa.VoterSet[primitives.AuthorityID], ks keystore.KeyStore) *primitives.AuthorityID {
	if ks == nil {
		return nil
	}

	for _, voter := range voters.Voters() {
		if ks.HasKeys([]keystore.PublicKey{{
			Key:       voter.ID.Bytes(),
			KeyTypeID: crypto.GRANDPA,
		}}) {
			return &voter.ID
		}
	}
	return nil
}
