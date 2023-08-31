// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/telemetry"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/blocktree"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"golang.org/x/exp/maps"
)

const (
	defaultGrandpaInterval = time.Second
)

var (
	logger = log.NewFromGlobal(log.AddContext("pkg", "grandpa"))

	ErrUnsupportedSubround = errors.New("unsupported subround")
	roundGauge             = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "gossamer_grandpa",
		Name:      "round",
		Help:      "current grandpa round",
	})
)

// Service represents the current state of the grandpa protocol
type Service struct {
	// preliminaries
	ctx            context.Context
	cancel         context.CancelFunc
	blockState     BlockState
	grandpaState   GrandpaState
	keypair        *ed25519.Keypair // TODO: change to grandpa keystore (#1870)
	mapLock        sync.Mutex
	chanLock       sync.Mutex
	roundLock      sync.Mutex
	authority      bool          // run the service as an authority (ie participate in voting)
	paused         atomic.Value  // the service will be paused if it is waiting for catch up responses
	resumed        chan struct{} // this channel will be closed when the service resumes
	messageHandler *MessageHandler
	network        Network
	interval       time.Duration

	// current state information
	state *State // current state
	// map[ed25519.PublicKeyBytes]*SignedVote - pre-votes for the current round
	prevotes *sync.Map
	// map[ed25519.PublicKeyBytes]*SignedVote - pre-commits for the current round
	precommits      *sync.Map
	pvEquivocations map[ed25519.PublicKeyBytes][]*SignedVote // equivocatory votes for current pre-vote stage
	pcEquivocations map[ed25519.PublicKeyBytes][]*SignedVote // equivocatory votes for current pre-commit stage
	tracker         *tracker                                 // tracker of vote messages we may need in the future
	head            *types.Header                            // most recently finalised block

	// historical information
	preVotedBlock      map[uint64]*Vote // map of round number -> pre-voted block
	bestFinalCandidate map[uint64]*Vote // map of round number -> best final candidate

	// channels for communication with other services
	finalisedCh chan *types.FinalisationInfo

	telemetry Telemetry
}

// Config represents a GRANDPA service configuration
type Config struct {
	LogLvl       log.Level
	BlockState   BlockState
	GrandpaState GrandpaState
	Network      Network
	Voters       []Voter
	Keypair      *ed25519.Keypair
	Authority    bool
	Interval     time.Duration
	Telemetry    Telemetry
}

// NewService returns a new GRANDPA Service instance.
func NewService(cfg *Config) (*Service, error) {
	logger.Patch(log.SetLevel(cfg.LogLvl))

	var pub string
	if cfg.Authority {
		pub = cfg.Keypair.Public().Hex()
	}

	logger.Debugf(
		"creating service with authority=%t, pub=%s and voter set %s",
		cfg.Authority, pub, Voters(cfg.Voters))

	// get latest finalised header
	head, err := cfg.BlockState.GetFinalisedHeader(0, 0)
	if err != nil {
		return nil, err
	}

	setID, err := cfg.GrandpaState.GetCurrentSetID()
	if err != nil {
		return nil, err
	}

	finalisedCh := cfg.BlockState.GetFinalisedNotifierChannel()

	round, err := cfg.GrandpaState.GetLatestRound()
	if err != nil {
		return nil, err
	}

	if cfg.Interval == 0 {
		cfg.Interval = defaultGrandpaInterval
	}

	ctx, cancel := context.WithCancel(context.Background())
	s := &Service{
		ctx:                ctx,
		cancel:             cancel,
		state:              NewState(cfg.Voters, setID, round),
		blockState:         cfg.BlockState,
		grandpaState:       cfg.GrandpaState,
		keypair:            cfg.Keypair,
		authority:          cfg.Authority,
		prevotes:           new(sync.Map),
		precommits:         new(sync.Map),
		pvEquivocations:    make(map[ed25519.PublicKeyBytes][]*SignedVote),
		pcEquivocations:    make(map[ed25519.PublicKeyBytes][]*SignedVote),
		preVotedBlock:      make(map[uint64]*Vote),
		bestFinalCandidate: make(map[uint64]*Vote),
		head:               head,
		resumed:            make(chan struct{}),
		network:            cfg.Network,
		finalisedCh:        finalisedCh,
		interval:           cfg.Interval,
		telemetry:          cfg.Telemetry,
	}

	if err := s.registerProtocol(); err != nil {
		return nil, err
	}

	s.messageHandler = NewMessageHandler(s, s.blockState, cfg.Telemetry)
	s.tracker = newTracker(s.blockState, s.messageHandler)
	s.paused.Store(false)
	return s, nil
}

// Start begins the GRANDPA finality service
func (s *Service) Start() error {
	// if we're not an authority, we don't need to worry about the voting process.
	// the grandpa service is only used to verify incoming block justifications
	if !s.authority {
		return nil
	}

	s.tracker.start()

	go func() {
		err := s.initiate()
		if err != nil {
			panic(fmt.Sprintf("running grandpa service: %s", err))
		}
	}()

	return nil
}

// Stop stops the GRANDPA finality service
func (s *Service) Stop() error {
	s.chanLock.Lock()
	defer s.chanLock.Unlock()

	s.cancel()
	s.blockState.FreeFinalisedNotifierChannel(s.finalisedCh)

	if !s.authority {
		return nil
	}

	s.tracker.stop()
	return nil
}

// authorities returns the current grandpa authorities
func (s *Service) authorityKeySet() (authorityKeys map[string]struct{}) {
	authorityKeys = make(map[string]struct{}, len(s.state.voters))
	for idx := range s.state.voters {
		voter := s.state.voters[idx]
		authority := &types.Authority{
			Key:    &voter.Key,
			Weight: voter.ID,
		}
		encodedKey := authority.Key.Encode()
		authorityKeys[string(encodedKey)] = struct{}{}
	}

	return authorityKeys
}

// updateAuthorities updates the grandpa voter set, increments the setID, and resets the round numbers
func (s *Service) updateAuthorities() error {
	currSetID, err := s.grandpaState.GetCurrentSetID()
	if err != nil {
		return fmt.Errorf("cannot get current set id: %w", err)
	}

	// set ID hasn't changed, do nothing
	if currSetID == s.state.setID {
		return nil
	}

	nextAuthorities, err := s.grandpaState.GetAuthorities(currSetID)
	if err != nil {
		return fmt.Errorf("cannot get authorities for set id %d: %w", currSetID, err)
	}

	s.state.voters = nextAuthorities
	s.state.setID = currSetID
	// round resets to 1 after a set ID change,
	// setting to 0 before incrementing indicates
	// the setID has been increased
	s.state.round = 0
	roundGauge.Set(float64(s.state.round))

	s.sendTelemetryAuthoritySet()

	return nil
}

func (s *Service) publicKeyBytes() ed25519.PublicKeyBytes {
	return s.keypair.Public().(*ed25519.PublicKey).AsBytes()
}

func (s *Service) sendTelemetryAuthoritySet() {
	authorityID := s.keypair.Public().Hex()
	authorities := make([]string, len(s.state.voters))
	for i, voter := range s.state.voters {
		authorities[i] = fmt.Sprint(voter.ID)
	}

	authoritiesBytes, err := json.Marshal(authorities)
	if err != nil {
		logger.Warnf("could not marshal authorities: %s", err)
		return
	}

	s.telemetry.SendMessage(
		telemetry.NewAfgAuthoritySet(
			authorityID,
			fmt.Sprint(s.state.setID),
			string(authoritiesBytes),
		),
	)
}

func (s *Service) initiateRound() error {
	// if there is an authority change, execute it
	err := s.updateAuthorities()
	if err != nil {
		return fmt.Errorf("cannot update authorities while initiating the round: %w", err)
	}

	round, setID, err := s.blockState.GetHighestRoundAndSetID()
	if err != nil {
		return fmt.Errorf("cannot get highest round and set id: %w", err)
	}

	if round > s.state.round && setID == s.state.setID {
		logger.Debugf(
			"found block finalised in higher round, updating our round to be %d...",
			round)
		s.state.round = round
		roundGauge.Set(float64(s.state.round))
		err = s.grandpaState.SetLatestRound(round)
		if err != nil {
			return err
		}
	}

	if setID > s.state.setID {
		logger.Debugf("found block finalised in higher setID, updating our setID to be %d...", setID)
		s.state.setID = setID
		s.state.round = round
	}

	s.head, err = s.blockState.GetFinalisedHeader(round, setID)
	if err != nil {
		logger.Criticalf("failed to get finalised header for round %d: %s", round, err)
		return err
	}

	// there was a setID change, or the node was started from genesis
	if s.state.round == 0 {
		s.chanLock.Lock()
		s.mapLock.Lock()
		s.preVotedBlock[0] = NewVoteFromHeader(s.head)
		s.bestFinalCandidate[0] = NewVoteFromHeader(s.head)
		s.mapLock.Unlock()
		s.chanLock.Unlock()
	}

	// make sure no votes can be validated while we are incrementing rounds
	s.roundLock.Lock()
	defer s.roundLock.Unlock()

	s.state.round++
	logger.Debugf("incrementing grandpa round, next round will be %d", s.state.round)
	s.prevotes = new(sync.Map)
	s.precommits = new(sync.Map)
	s.pvEquivocations = make(map[ed25519.PublicKeyBytes][]*SignedVote)
	s.pcEquivocations = make(map[ed25519.PublicKeyBytes][]*SignedVote)

	return nil
}

// initiate initates the grandpa service to begin voting in sequential rounds
func (s *Service) initiate() error {
	finalisationHandler := newFinalisationHandler(s)
	errorCh, err := finalisationHandler.Start()
	if err != nil {
		return fmt.Errorf("starting finalisation handler: %w", err)
	}

	for {
		select {
		case <-s.ctx.Done():
			return finalisationHandler.Stop()
		case err := <-errorCh:
			return err
		}
	}
}

func (s *Service) handleIsPrimary() (bool, error) {
	// derive primary
	primary := s.derivePrimary()

	// if primary, broadcast the best final candidate from the previous round
	// otherwise, do nothing
	if !bytes.Equal(primary.Key.Encode(), s.keypair.Public().Encode()) {
		return false, nil
	}

	if s.head.Number > 0 {
		s.primaryBroadcastCommitMessage()
	}

	best, err := s.blockState.BestBlockHeader()
	if err != nil {
		return false, err
	}

	pv := &Vote{
		Hash:   best.Hash(),
		Number: uint32(best.Number),
	}

	// send primary prevote message to network
	spv, primProposal, err := s.createSignedVoteAndVoteMessage(pv, primaryProposal)
	if err != nil {
		return false, fmt.Errorf("failed to create primary proposal message: %w", err)
	}

	s.prevotes.Store(s.publicKeyBytes(), spv)

	msg, err := primProposal.ToConsensusMessage()
	if err != nil {
		return false, fmt.Errorf("failed to encode finalisation message: %w", err)
	}

	s.network.GossipMessage(msg)
	return true, nil
}

// broadcast commit message from the previous round to the network
// ignore errors, since it's not critical to broadcast
func (s *Service) primaryBroadcastCommitMessage() {
	cm, err := s.newCommitMessage(s.head, s.state.round-1, s.state.setID)
	if err != nil {
		return
	}

	// send finalised block from previous round to network
	msg, err := cm.ToConsensusMessage()
	if err != nil {
		logger.Warnf("failed to encode finalisation message: %s", err)
	}

	s.network.GossipMessage(msg)
}

func (s *Service) checkRoundCompletable() (bool, error) {
	// check if the current round contains a finalized block
	has, err := s.blockState.HasFinalisedBlock(s.state.round, s.state.setID)
	if err != nil {
		return false, fmt.Errorf("checking for finalised block in block state: %w", err)
	}

	if has {
		logger.Debugf("block was finalised for round %d", s.state.round)
		return true, nil
	}

	// a block was finalised, seems like we missed some messages
	highestRound, highestSetID, err := s.blockState.GetHighestRoundAndSetID()
	if err != nil {
		return false, fmt.Errorf("getting highest round and set id: %w", err)
	}

	if highestRound > s.state.round {
		logger.Debugf("block was finalised for round %d and set id %d",
			highestRound, highestSetID)
		return true, nil
	}

	// a block was finalised, seems like we missed some messages
	if highestSetID > s.state.setID {
		logger.Debugf("block was finalised for round %d and set id %d",
			highestRound, highestSetID)
		return true, nil
	}

	return false, nil
}

func (s *Service) retrieveBestFinalCandidate() (bestFinalCandidate *types.GrandpaVote,
	precommitCount uint64, err error) {

	bestFinalCandidate, err = s.getBestFinalCandidate()
	if err != nil {
		return nil, 0, fmt.Errorf("getting best final candidate: %w", err)
	}

	if bestFinalCandidate.Number < uint32(s.head.Number) {
		return nil, 0, fmt.Errorf("%w: candidate number %d, latest finalized block number %d",
			errBeforeFinalizedBlock, bestFinalCandidate.Number, s.head.Number)
	}

	precommitCount, err = s.getTotalVotesForBlock(bestFinalCandidate.Hash, precommit)
	if err != nil {
		return nil, 0, fmt.Errorf("getting total votes for block %s: %w",
			bestFinalCandidate.Hash.Short(), err)
	}

	return bestFinalCandidate, precommitCount, nil
}

// attemptToFinalize check if we should finalize the current round
func (s *Service) attemptToFinalize() (isFinalizable bool, err error) {
	bestFinalCandidate, precommitCount, err := s.retrieveBestFinalCandidate()
	if err != nil {
		return false, fmt.Errorf("getting best final candidate: %w", err)
	}

	// once we reach the threshold we should stop sending precommit messages to other peers
	if bestFinalCandidate.Number < uint32(s.head.Number) || precommitCount <= s.state.threshold() {
		return false, nil
	}

	err = s.finalise()
	if err != nil {
		return false, fmt.Errorf("finalising: %w", err)
	}

	// if we haven't received a finalisation message for this block yet, broadcast a finalisation message
	votes := s.getDirectVotes(precommit)
	logger.Debugf("block was finalised for round %d and set id %d. "+
		"Head hash is %s, %d direct votes for bfc and %d total votes for bfc",
		s.state.round, s.state.setID, s.head.Hash(), votes[*bestFinalCandidate], precommitCount)

	return true, nil
}

func (s *Service) sendPrecommitMessage(voteMessage *VoteMessage) (err error) {
	logger.Debugf("sending pre-commit message %s...", voteMessage.Message)

	consensusMessage, err := voteMessage.ToConsensusMessage()
	if err != nil {
		return fmt.Errorf("transforming pre-commit into consensus message: %w", err)
	}

	s.network.GossipMessage(consensusMessage)
	logger.Tracef("sent pre-commit message: %v", consensusMessage)
	return nil
}

func (s *Service) sendPrevoteMessage(vm *VoteMessage) (err error) {
	logger.Debugf("sending pre-vote message %s...", vm)

	consensusMessage, err := vm.ToConsensusMessage()
	if err != nil {
		return fmt.Errorf("transforming pre-vote into consensus message: %w", err)
	}

	s.network.GossipMessage(consensusMessage)
	logger.Tracef("sent pre-vote message: %v", consensusMessage)
	return nil
}

func (s *Service) loadVote(key ed25519.PublicKeyBytes, stage Subround) (*SignedVote, bool) {
	var (
		v   interface{}
		has bool
	)

	switch stage {
	case prevote, primaryProposal:
		v, has = s.prevotes.Load(key)
	case precommit:
		v, has = s.precommits.Load(key)
	}

	if !has {
		return nil, false
	}

	return v.(*SignedVote), true
}

func (s *Service) deleteVote(key ed25519.PublicKeyBytes, stage Subround) {
	switch stage {
	case prevote, primaryProposal:
		s.prevotes.Delete(key)
	case precommit:
		s.precommits.Delete(key)
	}
}

// determinePreVote determines what block is our pre-voted block for the current round
func (s *Service) determinePreVote() (*Vote, error) {
	var vote *Vote

	bestBlockHeader, err := s.blockState.BestBlockHeader()
	if err != nil {
		return nil, fmt.Errorf("cannot get best block header: %w", err)
	}

	// if we receive a vote message from the primary with a
	// block that's greater than or equal to the current pre-voted block
	// and greater than the best final candidate from the last round, we choose that.
	// otherwise, we simply choose the head of our chain.
	primary := s.derivePrimary()
	prm, has := s.loadVote(primary.PublicKeyBytes(), prevote)
	if has && prm.Vote.Number >= uint32(s.head.Number) {
		vote = &prm.Vote
	} else {
		vote = NewVoteFromHeader(bestBlockHeader)
	}

	nextChange, err := s.grandpaState.NextGrandpaAuthorityChange(bestBlockHeader.Hash(), bestBlockHeader.Number)
	if errors.Is(err, state.ErrNoNextAuthorityChange) {
		return vote, nil
	} else if err != nil {
		return nil, fmt.Errorf("cannot get next grandpa authority change: %w", err)
	}

	if uint(vote.Number) > nextChange {
		header, err := s.blockState.GetHeaderByNumber(nextChange)
		if err != nil {
			return nil, err
		}
		vote = NewVoteFromHeader(header)
	}

	return vote, nil
}

// determinePreCommit determines what block is our pre-committed block for the current round
func (s *Service) determinePreCommit() (*Vote, error) {
	// the pre-committed block is simply the pre-voted block (GRANDPA-GHOST)
	pvb, err := s.getPreVotedBlock()
	if err != nil {
		return nil, err
	}

	s.mapLock.Lock()
	s.preVotedBlock[s.state.round] = &pvb
	s.mapLock.Unlock()

	nextChange, err := s.grandpaState.NextGrandpaAuthorityChange(pvb.Hash, uint(pvb.Number))
	if errors.Is(err, state.ErrNoNextAuthorityChange) {
		return &pvb, nil
	} else if err != nil {
		return nil, fmt.Errorf("cannot get next grandpa authority change: %w", err)
	}

	if uint(pvb.Number) > nextChange {
		header, err := s.blockState.GetHeaderByNumber(nextChange)
		if err != nil {
			return nil, err
		}

		pvb = *NewVoteFromHeader(header)
	}
	return &pvb, nil
}

// finalise finalises the round by setting the best final candidate for this round
func (s *Service) finalise() error {
	// get best final candidate
	bfc, err := s.getBestFinalCandidate()
	if err != nil {
		return err
	}

	pv, err := s.getPreVotedBlock()
	if err != nil {
		return err
	}

	s.mapLock.Lock()
	defer s.mapLock.Unlock()

	s.preVotedBlock[s.state.round] = &pv

	// set best final candidate
	s.bestFinalCandidate[s.state.round] = bfc

	// create prevote justification ie. list of all signed prevotes for the bfc
	pvs, err := s.createJustification(bfc.Hash, prevote)
	if err != nil {
		return err
	}

	// create precommit justification ie. list of all signed precommits for the bfc
	pcs, err := s.createJustification(bfc.Hash, precommit)
	if err != nil {
		return err
	}

	pcj, err := scale.Marshal(*newJustification(s.state.round, bfc.Hash, bfc.Number, pcs))
	if err != nil {
		return err
	}

	if err = s.blockState.SetJustification(bfc.Hash, pcj); err != nil {
		return err
	}

	if err = s.grandpaState.SetPrevotes(s.state.round, s.state.setID, pvs); err != nil {
		return err
	}

	if err = s.grandpaState.SetPrecommits(s.state.round, s.state.setID, pcs); err != nil {
		return err
	}

	s.head, err = s.blockState.GetHeader(bfc.Hash)
	if err != nil {
		return err
	}

	// set finalised head for round in db
	if err = s.blockState.SetFinalisedHash(bfc.Hash, s.state.round, s.state.setID); err != nil {
		return err
	}

	return s.grandpaState.SetLatestRound(s.state.round)
}

// createJustification collects the signed precommits received for this round and turns them into
// a justification by adding all signed precommits that are for the best finalised candidate or
// a descendent of the bfc
func (s *Service) createJustification(bfc common.Hash, stage Subround) ([]SignedVote, error) {
	var (
		spc  *sync.Map
		err  error
		just []SignedVote
		eqv  map[ed25519.PublicKeyBytes][]*SignedVote
	)

	switch stage {
	case prevote:
		spc = s.prevotes
		eqv = s.pvEquivocations
	case precommit:
		spc = s.precommits
		eqv = s.pcEquivocations
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedSubround, stage)
	}

	spc.Range(func(_, value interface{}) bool {
		pc := value.(*SignedVote)
		var isDescendant bool

		isDescendant, err = s.blockState.IsDescendantOf(bfc, pc.Vote.Hash)
		if err != nil {
			return false
		}

		if !isDescendant {
			return true
		}

		just = append(just, *pc)
		return true
	})

	if err != nil {
		return nil, err
	}

	for _, votes := range eqv {
		for _, vote := range votes {
			just = append(just, *vote)
		}
	}

	return just, nil
}

// derivePrimary returns the primary for the current round
func (s *Service) derivePrimary() Voter {
	return s.state.voters[s.state.round%uint64(len(s.state.voters))]
}

// getBestFinalCandidate calculates the set of blocks that are less than or equal to the pre-voted block in height,
// with >2/3 pre-commit votes, then returns the block with the highest number from this set.
func (s *Service) getBestFinalCandidate() (*Vote, error) {
	prevoted, err := s.getPreVotedBlock()
	if err != nil {
		return nil, err
	}

	// get all blocks with >2/3 pre-commits
	blocks, err := s.getPossibleSelectedBlocks(precommit, s.state.threshold())
	if err != nil {
		return nil, err
	}

	// if there are no blocks with >2/3 pre-commits, just return the pre-voted block
	// TODO: is this correct? the spec implies that it should return nil, but discussions have suggested
	// that we return the prevoted block. (#1815)
	if len(blocks) == 0 {
		return &prevoted, nil
	}

	// if there are multiple blocks, get the one with the highest number
	// that is also an ancestor of the prevoted block (or is the prevoted block)
	bfc := &Vote{
		Hash:   s.blockState.GenesisHash(),
		Number: 0,
	}

	for h, n := range blocks {
		// check if the current block is an ancestor of prevoted block
		isDescendant, err := s.blockState.IsDescendantOf(h, prevoted.Hash)
		if err != nil {
			return nil, err
		}

		if !isDescendant {
			// find common ancestor, implicitly has >2/3 votes
			pred, err := s.blockState.LowestCommonAncestor(h, prevoted.Hash)
			if err != nil {
				return nil, err
			}

			v, err := NewVoteFromHash(pred, s.blockState)
			if err != nil {
				return nil, err
			}

			n = v.Number
			h = pred
		}

		// choose block with highest number
		if n > bfc.Number {
			bfc = &Vote{
				Hash:   h,
				Number: n,
			}
		}
	}

	return bfc, nil
}

// getPreVotedBlock returns the current pre-voted block B. also known as GRANDPA-GHOST.
// the pre-voted block is the block with the highest block number in the set of all the blocks with
// total votes >2/3 the total number of voters, where the total votes is determined by getTotalVotesForBlock.
func (s *Service) getPreVotedBlock() (Vote, error) {
	blocks, err := s.getPossibleSelectedBlocks(prevote, s.state.threshold())
	if err != nil {
		return Vote{}, err
	}

	// if there are no blocks with >=2/3 voters, then just pick the highest voted block
	if len(blocks) == 0 {
		return s.getGrandpaGHOST()
	}

	// if there is one block, return it
	if len(blocks) == 1 {
		for h, n := range blocks {
			return Vote{
				Hash:   h,
				Number: n,
			}, nil
		}
	}

	// if there are multiple, find the one with the highest number and return it
	highest := Vote{
		Hash:   s.head.Hash(),
		Number: uint32(s.head.Number),
	}

	for h, n := range blocks {
		if n > highest.Number {
			highest = Vote{
				Hash:   h,
				Number: n,
			}
		}
	}

	return highest, nil
}

// getGrandpaGHOST returns the block with the most votes. if there are multiple blocks with the same number
// of votes, it picks the one with the highest number.
func (s *Service) getGrandpaGHOST() (Vote, error) {
	threshold := s.state.threshold()

	var (
		blocks map[common.Hash]uint32
		err    error
	)

	for {
		blocks, err = s.getPossibleSelectedBlocks(prevote, threshold)
		if err != nil {
			return Vote{}, err
		}

		if len(blocks) > 0 || threshold == 0 {
			break
		}

		threshold--
	}

	if len(blocks) == 0 {
		return Vote{}, ErrNoGHOST
	}

	// if there are multiple, find the one with the highest number and return it
	highest := Vote{
		Hash:   s.head.Hash(),
		Number: uint32(s.head.Number),
	}

	for h, n := range blocks {
		if n > highest.Number {
			highest = Vote{
				Hash:   h,
				Number: n,
			}
		}
	}

	return highest, nil
}

// getPossibleSelectedBlocks returns blocks with total votes >threshold in a map of block hash -> block number.
// if there are no blocks that have >threshold direct votes,
// this function will find ancestors of those blocks that do have >threshold votes.
// note that by voting for a block, all of its ancestor blocks
// are automatically voted for.
// thus, if there are no blocks with >threshold total votes,
// but the sum of votes for blocks A and B is >threshold, then this function returns
// the first common ancestor of A and B.
// in general, this function will return the highest block on each chain with >threshold votes.
func (s *Service) getPossibleSelectedBlocks(stage Subround, threshold uint64) (map[common.Hash]uint32, error) {
	// get blocks that were directly voted for
	votes := s.getDirectVotes(stage)
	blocks := make(map[common.Hash]uint32)

	// check if any of them have >threshold votes
	for v := range votes {
		total, err := s.getTotalVotesForBlock(v.Hash, stage)
		if err != nil {
			return nil, err
		}

		if total > threshold {
			blocks[v.Hash] = v.Number
		}
	}

	// since we want to select the block with the highest number that has >threshold votes,
	// we can return here since their ancestors won't have a higher number.
	if len(blocks) != 0 {
		return blocks, nil
	}

	// no block has >threshold direct votes, check for votes for ancestors recursively
	var err error
	va := s.getVotes(stage)

	for v := range votes {
		blocks, err = s.getPossibleSelectedAncestors(va, v.Hash, blocks, stage, threshold)
		if err != nil {
			return nil, err
		}
	}

	return blocks, nil
}

// getPossibleSelectedAncestors recursively searches for ancestors with >2/3 votes
// it returns a map of block hash -> number, such that the blocks in the map have >2/3 votes
func (s *Service) getPossibleSelectedAncestors(votes []Vote, curr common.Hash,
	selected map[common.Hash]uint32, stage Subround,
	threshold uint64) (map[common.Hash]uint32, error) {
	for _, v := range votes {
		if v.Hash == curr {
			continue
		}

		// find common ancestor, check if votes for it is >threshold or not
		pred, err := s.blockState.LowestCommonAncestor(v.Hash, curr)
		if errors.Is(err, blocktree.ErrNodeNotFound) {
			continue
		} else if err != nil {
			return nil, err
		}

		if pred == curr {
			return selected, nil
		}

		total, err := s.getTotalVotesForBlock(pred, stage)
		if err != nil {
			return nil, err
		}

		if total > threshold {
			var h *types.Header
			h, err = s.blockState.GetHeader(pred)
			if err != nil {
				return nil, err
			}

			selected[pred] = uint32(h.Number)
		} else {
			selected, err = s.getPossibleSelectedAncestors(votes, pred, selected, stage, threshold)
			if err != nil {
				return nil, err
			}
		}
	}

	return selected, nil
}

// getTotalVotesForBlock returns the total number of observed votes for a block B in a subround, which is equal
// to the direct votes for B and B's descendants plus the total number of equivocating voters
func (s *Service) getTotalVotesForBlock(hash common.Hash, stage Subround) (uint64, error) {
	// observed votes for block
	dv, err := s.getVotesForBlock(hash, stage)
	if err != nil {
		return 0, err
	}

	// equivocatory votes
	var ev int
	if stage == prevote {
		ev = len(s.pvEquivocations)
	} else {
		ev = len(s.pcEquivocations)
	}

	return dv + uint64(ev), nil
}

// getVotesForBlock returns the number of observed votes for a block B.
// The set of all observed votes by v in the sub-round stage of round r for block B is
// equal to all of the observed direct votes cast for block B and all of the B's descendants
func (s *Service) getVotesForBlock(hash common.Hash, stage Subround) (uint64, error) {
	votes := s.getDirectVotes(stage)

	// B will be counted as in it's own subchain, so don't need to start with B's vote count
	votesForBlock := uint64(0)

	for v, c := range votes {
		// check if the current block is a descendant of B
		isDescendant, err := s.blockState.IsDescendantOf(hash, v.Hash)
		if errors.Is(err, blocktree.ErrStartNodeNotFound) || errors.Is(err, blocktree.ErrEndNodeNotFound) {
			continue
		} else if err != nil {
			return 0, err
		}

		if !isDescendant {
			continue
		}

		votesForBlock += c
	}

	return votesForBlock, nil
}

// getDirectVotes returns a map of Votes to direct vote counts
func (s *Service) getDirectVotes(stage Subround) map[Vote]uint64 {
	votes := make(map[Vote]uint64)

	var src *sync.Map
	if stage == prevote {
		src = s.prevotes
	} else {
		src = s.precommits
	}

	src.Range(func(_, value interface{}) bool {
		sv := value.(*SignedVote)
		votes[sv.Vote]++
		return true
	})

	return votes
}

// getVotes returns all the current votes as an array
func (s *Service) getVotes(stage Subround) []Vote {
	votes := s.getDirectVotes(stage)
	return maps.Keys(votes)
}

// GetSetID returns the current setID
func (s *Service) GetSetID() uint64 {
	return s.state.setID
}

// GetRound return the current round number
func (s *Service) GetRound() uint64 {
	// Tim: I don't think we need to lock in this case.  Reading an int will
	// not produce a concurrent read/write panic
	s.roundLock.Lock()
	defer s.roundLock.Unlock()
	return s.state.round
}

// GetVoters returns the list of current grandpa.Voters
func (s *Service) GetVoters() Voters {
	return s.state.voters
}

// PreVotes returns the current prevotes to the current round
func (s *Service) PreVotes() []ed25519.PublicKeyBytes {
	s.mapLock.Lock()
	defer s.mapLock.Unlock()

	votes := make([]ed25519.PublicKeyBytes, 0, s.lenVotes(prevote)+len(s.pvEquivocations))

	s.prevotes.Range(func(k interface{}, _ interface{}) bool {
		b := k.(ed25519.PublicKeyBytes)
		votes = append(votes, b)
		return true
	})

	votes = append(votes, maps.Keys(s.pvEquivocations)...)
	return votes
}

// PreCommits returns the current precommits to the current round
func (s *Service) PreCommits() []ed25519.PublicKeyBytes {
	s.mapLock.Lock()
	defer s.mapLock.Unlock()

	votes := make([]ed25519.PublicKeyBytes, 0, s.lenVotes(precommit)+len(s.pcEquivocations))

	s.precommits.Range(func(k interface{}, _ interface{}) bool {
		b := k.(ed25519.PublicKeyBytes)
		votes = append(votes, b)
		return true
	})

	votes = append(votes, maps.Keys(s.pvEquivocations)...)
	return votes
}

func (s *Service) lenVotes(stage Subround) int {
	var count int

	switch stage {
	case prevote, primaryProposal:
		s.prevotes.Range(func(_, _ interface{}) bool {
			count++
			return true
		})
	case precommit:
		s.precommits.Range(func(_, _ interface{}) bool {
			count++
			return true
		})
	}

	return count
}

func (s *Service) handleVoteMessage(from peer.ID, vote *VoteMessage) (err error) {
	logger.Debugf("received vote message, (peer: %s, msg: %+v)", from, vote)
	s.sendTelemetryVoteMessage(vote)

	grandpaVote, err := s.validateVoteMessage(from, vote)
	if err != nil {
		return fmt.Errorf("validating vote message: %w", err)
	}

	threshold := s.state.threshold() + 1
	logger.Debugf(
		"validated vote message %v from %s, round %d, subround %d, "+
			"prevote count %d, precommit count %d, votes needed %d",
		grandpaVote, vote.Message.AuthorityID, vote.Round, vote.Message.Stage,
		s.lenVotes(prevote), s.lenVotes(precommit), threshold)
	return nil
}

func (s *Service) handleCommitMessage(commitMessage *CommitMessage) error {
	logger.Debugf("received commit message: %+v", commitMessage)

	err := verifyBlockHashAgainstBlockNumber(s.blockState,
		commitMessage.Vote.Hash, uint(commitMessage.Vote.Number))
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			s.tracker.addCommit(commitMessage)
		}

		return fmt.Errorf("verifying block hash against block number: %w", err)
	}

	containsPrecommitsSignedBy := make([]string, len(commitMessage.AuthData))
	for i, authData := range commitMessage.AuthData {
		containsPrecommitsSignedBy[i] = authData.AuthorityID.String()
	}

	s.telemetry.SendMessage(
		telemetry.NewAfgReceivedCommit(
			commitMessage.Vote.Hash,
			fmt.Sprint(commitMessage.Vote.Number),
			containsPrecommitsSignedBy,
		),
	)

	has, err := s.blockState.HasFinalisedBlock(commitMessage.Round, s.state.setID)
	if err != nil {
		return fmt.Errorf("checking for a finalized block in the block state: %w", err)
	}

	if has {
		return nil
	}

	// check justification here
	err = verifyCommitMessageJustification(*commitMessage, s.state.setID,
		s.state.threshold(), s.authorityKeySet(), s.blockState)
	if err != nil {
		if errors.Is(err, blocktree.ErrStartNodeNotFound) {
			// we haven't synced the committed block yet, add this to the tracker for later processing
			s.tracker.addCommit(commitMessage)
		}
		return fmt.Errorf("verifying commit message justification: %w", err)
	}

	err = s.blockState.SetFinalisedHash(commitMessage.Vote.Hash, commitMessage.Round, s.state.setID)
	if err != nil {
		return fmt.Errorf("setting finalised hash: %w", err)
	}

	preCommitSigned, err := compactToJustification(commitMessage.Precommits, commitMessage.AuthData)
	if err != nil {
		return fmt.Errorf("compacting justification: %w", err)
	}

	err = s.grandpaState.SetPrecommits(commitMessage.Round, commitMessage.SetID, preCommitSigned)
	if err != nil {
		return fmt.Errorf("setting precommits: %w", err)
	}

	// TODO: re-add catch-up logic (#1531)
	return nil
}

func verifyCommitMessageJustification(commitMessage CommitMessage, setID uint64, threshold uint64,
	authorityKeySet map[string]struct{}, blockState BlockState) error {
	if len(commitMessage.Precommits) != len(commitMessage.AuthData) {
		return fmt.Errorf("%w: precommits len: %d, authorities len: %d",
			ErrPrecommitSignatureMismatch, len(commitMessage.Precommits), len(commitMessage.AuthData))
	}

	if commitMessage.SetID != setID {
		return fmt.Errorf("%w: grandpa state set id %d, set id in the commit message %d",
			ErrSetIDMismatch, setID, commitMessage.SetID)
	}

	highestFinalizedHeader, err := blockState.GetHighestFinalisedHeader()
	if err != nil {
		return fmt.Errorf("getting highest finalised header: %w", err)
	}

	isDescendant, err := blockState.IsDescendantOf(highestFinalizedHeader.Hash(), commitMessage.Vote.Hash)
	if err != nil {
		return fmt.Errorf("verifying ancestry of highest finalised block: %w", err)
	}

	if !isDescendant {
		return fmt.Errorf("%w: vote hash %s and highest finalised header %s",
			errVoteBlockMismatch, commitMessage.Vote.Hash.Short(), highestFinalizedHeader.Hash())
	}

	eqvVoters := getEquivocatoryVoters(commitMessage.AuthData)
	var totalValidPrecommits int
	for i, preCommit := range commitMessage.Precommits {
		justification := &SignedVote{
			Vote:        preCommit,
			Signature:   commitMessage.AuthData[i].Signature,
			AuthorityID: commitMessage.AuthData[i].AuthorityID,
		}

		err := verifyJustification(justification, commitMessage.Round, setID, precommit, authorityKeySet)
		if err != nil {
			logger.Errorf("verifying justification: %s", err)
			continue
		}

		isDescendant, err := blockState.IsDescendantOf(commitMessage.Vote.Hash, justification.Vote.Hash)
		if err != nil {
			logger.Warnf("could not check for descendant: %s", err)
			continue
		}

		precommitedHeader, err := blockState.GetHeader(preCommit.Hash)
		if err != nil {
			return fmt.Errorf("getting header: %w", err)
		}

		if precommitedHeader.Number != uint(preCommit.Number) {
			const errFormat = "%w: pre commit corresponding header has block number %d " +
				"and pre commit has block number %d"

			return fmt.Errorf(errFormat,
				ErrBlockNumbersMismatch, precommitedHeader.Number, preCommit.Number)
		}

		if _, ok := eqvVoters[commitMessage.AuthData[i].AuthorityID]; ok {
			continue
		}

		if isDescendant {
			totalValidPrecommits++
		}
	}

	validAndEqv := uint64(totalValidPrecommits) + uint64(len(eqvVoters))
	// confirm total # signatures >= grandpa threshold
	if validAndEqv < threshold {
		return fmt.Errorf("%w: for finalisation message; need %d votes but received only %d valid votes",
			ErrMinVotesNotMet, threshold, validAndEqv)
	}

	logger.Debugf("validated commit message: %v", commitMessage)
	return nil
}

func verifyJustification(justification *SignedVote, round, setID uint64,
	stage Subround, authorityKeys map[string]struct{}) error {
	fullVote := FullVote{
		Stage: stage,
		Vote:  justification.Vote,
		Round: round,
		SetID: setID,
	}

	encodedFullVote, err := scale.Marshal(fullVote)
	if err != nil {
		return fmt.Errorf("scale encoding full vote: %w", err)
	}

	publicKey, err := ed25519.NewPublicKey(justification.AuthorityID[:])
	if err != nil {
		return fmt.Errorf("creating ed25519 public key: %w", err)
	}

	ok, err := publicKey.Verify(encodedFullVote, justification.Signature[:])
	if err != nil {
		return fmt.Errorf("verifying signature: %w", err)
	}

	if !ok {
		return fmt.Errorf("%w: 0x%x for message {%v}", ErrInvalidSignature, justification.Signature, fullVote)
	}

	justificationKey, err := justification.AuthorityID.Encode()
	if err != nil {
		return fmt.Errorf("encoding authority key: %w", err)
	}

	_, has := authorityKeys[string(justificationKey)]
	if has {
		return nil
	}

	return fmt.Errorf("%w: authority ID 0x%x", ErrVoterNotFound, justificationKey)
}
