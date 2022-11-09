// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/dot/telemetry"
)

var errServicesStopFailed = errors.New("services stop failed")

type ephemeralService interface {
	Start() error
	Stop() error
}

type finalizationHandler struct {
	servicesLock       sync.Mutex
	finalizationEngine ephemeralService
	votingRound        ephemeralService

	newServices   func() (engine, voting ephemeralService)
	timeout       time.Duration
	initiateRound func() error

	stopCh      chan struct{}
	handlerDone chan struct{}
	errorCh     chan error
}

func newFinalizationHandler(service *Service) *finalizationHandler {
	// builder is a constructor function which takes care to instantiate
	// and return the services needed to finalize a round, those services
	// are ephemeral services with a lifetime of a round
	builder := func() (engine, voting ephemeralService) {
		finalizationEngine := newFinalizationEngine(service)
		votingRound := newHandleVotingRound(service, finalizationEngine.actionCh)
		return finalizationEngine, votingRound
	}

	return &finalizationHandler{
		newServices: builder,

		timeout:       5 * time.Second,
		initiateRound: service.initiateRound,
		stopCh:        make(chan struct{}),
		handlerDone:   make(chan struct{}),
		errorCh:       make(chan error),
	}
}

func (fh *finalizationHandler) Start() (errorCh <-chan error, err error) {
	go fh.start()
	return fh.errorCh, nil
}

func (fh *finalizationHandler) start() {
	defer func() {
		close(fh.errorCh)
		close(fh.handlerDone)
	}()

	for {
		select {
		case <-fh.stopCh:
			return
		default:
		}

		err := fh.initiateRound()
		if err != nil {
			fh.errorCh <- fmt.Errorf("initiating round: %w", err)
			return
		}

		err = fh.waitServices()
		if err != nil {
			fh.errorCh <- fmt.Errorf("waiting for services: %w", err)
			return
		}
	}
}

func (fh *finalizationHandler) stop() (err error) {
	fh.servicesLock.Lock()
	defer fh.servicesLock.Unlock()

	finalizationEngineErrCh := make(chan error)
	go func() {
		finalizationEngineErrCh <- fh.finalizationEngine.Stop()
	}()

	votingRoundErrCh := make(chan error)
	go func() {
		votingRoundErrCh <- fh.votingRound.Stop()
	}()

	finalizationEngErr := <-finalizationEngineErrCh
	votingRoundErr := <-votingRoundErrCh

	if finalizationEngErr != nil && votingRoundErr != nil {
		return fmt.Errorf("%w: %s; %s", errServicesStopFailed, finalizationEngErr, votingRoundErr)
	}

	if finalizationEngErr != nil {
		return fmt.Errorf("%w: %s", errServicesStopFailed, finalizationEngErr)
	}

	if votingRoundErr != nil {
		return fmt.Errorf("%w: %s", errServicesStopFailed, votingRoundErr)
	}

	return nil
}

func (fh *finalizationHandler) Stop() (err error) {
	close(fh.stopCh)
	<-fh.errorCh
	<-fh.handlerDone

	err = fh.stop()
	return err
}

// waitServices will start the ephemeral services that handles the
// votes for the current round and once those services finishes the
// waitServices return without errors, if one ephemeral service face
// a problem the waitServices will shut down all the running ephemeral
// services and return an error, this function also returns if the
// finalizationHandler.Stop() method is called
func (fh *finalizationHandler) waitServices() error {
	fh.servicesLock.Lock()
	fh.finalizationEngine, fh.votingRound = fh.newServices()
	fh.servicesLock.Unlock()

	finalizationEngineErr := make(chan error)
	go func() {
		finalizationEngineErr <- fh.finalizationEngine.Start()
	}()

	votingRoundErr := make(chan error)
	go func() {
		votingRoundErr <- fh.votingRound.Start()
	}()

	for {
		select {
		case <-fh.stopCh:
			return nil

		case err := <-votingRoundErr:
			fmt.Printf("waiting services got an err: %s\n", err)
			if err == nil {
				votingRoundErr = nil
				// go out from the select case
				break
			}

			stopErr := fh.stop()
			if stopErr != nil {
				logger.Warnf("stopping finalisation handler: %s", stopErr)
			}
			return err

		case err := <-finalizationEngineErr:
			if err == nil {
				finalizationEngineErr = nil
				// go out from the select case
				break
			}

			stopErr := fh.stop()
			if stopErr != nil {
				logger.Warnf("stopping finalisation handler: %s", stopErr)
			}
			return err
		}

		finish := votingRoundErr == nil && finalizationEngineErr == nil
		if finish {
			return nil
		}
	}
}

type handleVotingRound struct {
	grandpaService       *Service
	finalizationEngineCh <-chan engineAction
	stopCh               chan struct{}
	engineDone           chan struct{}
}

func newHandleVotingRound(service *Service, finalizationEngineCh <-chan engineAction) *handleVotingRound {
	return &handleVotingRound{
		grandpaService:       service,
		stopCh:               make(chan struct{}),
		engineDone:           make(chan struct{}),
		finalizationEngineCh: finalizationEngineCh,
	}
}

func (h *handleVotingRound) Stop() (err error) {
	if h.engineDone == nil {
		return nil
	}

	close(h.stopCh)
	<-h.engineDone

	h.stopCh = nil
	return nil
}

// playGrandpaRound executes a round of GRANDPA
// at the end of this round, a block will be finalised.
// TODO(test): stopping handleVotingRound first and then stopping
// finalizationEngine might cause a write in a non-reading unbuff channel
// blocking the finalizationEngine to stop and triggering the stop timeout timer
func (h *handleVotingRound) Start() (err error) {
	defer func() {
		close(h.engineDone)
		h.engineDone = nil
	}()

	start := time.Now()

	logger.Debugf("starting round %d with set id %d",
		h.grandpaService.state.round, h.grandpaService.state.setID)

	for {
		select {
		case <-h.stopCh:
			return nil

		case action, ok := <-h.finalizationEngineCh:
			if !ok {
				return nil
			}

			switch action {
			case determinePrevote:
				isPrimary, err := h.grandpaService.handleIsPrimary()
				if err != nil {
					return fmt.Errorf("handling primary: %w", err)
				}

				// broadcast pre-vote
				preVote, err := h.grandpaService.determinePreVote()
				if err != nil {
					return fmt.Errorf("determining pre-vote: %w", err)
				}

				signedpreVote, prevoteMessage, err :=
					h.grandpaService.createSignedVoteAndVoteMessage(preVote, prevote)
				if err != nil {
					return fmt.Errorf("creating signed vote: %w", err)
				}

				if !isPrimary {
					h.grandpaService.prevotes.Store(h.grandpaService.publicKeyBytes(), signedpreVote)
				}

				logger.Debugf("sending pre-vote message: {%v}", prevoteMessage)
				h.grandpaService.sendPrevoteMessage(prevoteMessage)

			case determinePrecommit:
				preComit, err := h.grandpaService.determinePreCommit()
				if err != nil {
					return fmt.Errorf("determining pre-commit: %w", err)
				}

				signedpreComit, precommitMessage, err :=
					h.grandpaService.createSignedVoteAndVoteMessage(preComit, precommit)
				if err != nil {
					return fmt.Errorf("creating signed vote: %w", err)
				}

				logger.Debugf("sending pre-commit message: {%v}", precommitMessage)
				h.grandpaService.precommits.Store(h.grandpaService.publicKeyBytes(), signedpreComit)
				h.grandpaService.sendPrecommitMessage(precommitMessage)

			case finalize:
				commitMessage, err := h.grandpaService.newCommitMessage(
					h.grandpaService.head, h.grandpaService.state.round, h.grandpaService.state.setID)
				if err != nil {
					return fmt.Errorf("creating commit message: %w", err)
				}

				commitConsensusMessage, err := commitMessage.ToConsensusMessage()
				if err != nil {
					return fmt.Errorf("transforming commit into consensus message: %w", err)
				}

				logger.Debugf("sending commit message: %v", commitMessage)
				h.grandpaService.network.GossipMessage(commitConsensusMessage)
				h.grandpaService.telemetry.SendMessage(telemetry.NewAfgFinalizedBlocksUpTo(
					h.grandpaService.head.Hash(),
					fmt.Sprint(h.grandpaService.head.Number),
				))

				logger.Debugf("round completed in %s", time.Since(start))
				return nil

			case alreadyFinalized:
				logger.Debugf("round completed in %s", time.Since(start))
				return nil
			}
		}
	}
}

// actions that should take place accordingly to votes the
// finalisation engine knows about
type engineAction byte

const (
	determinePrevote engineAction = iota
	determinePrecommit
	alreadyFinalized
	finalize
)

type finalizationEngine struct {
	grandpaService *Service
	stopCh         chan struct{}
	engineDone     chan struct{}
	actionCh       chan engineAction
}

func newFinalizationEngine(service *Service) *finalizationEngine {
	return &finalizationEngine{
		grandpaService: service,
		actionCh:       make(chan engineAction),
		stopCh:         make(chan struct{}),
		engineDone:     make(chan struct{}),
	}
}

func (f *finalizationEngine) Stop() (err error) {
	if f.engineDone == nil {
		return nil
	}

	close(f.stopCh)
	<-f.engineDone

	f.stopCh = nil
	close(f.actionCh)
	return nil
}

func (f *finalizationEngine) Start() (err error) {
	defer func() {
		close(f.engineDone)
		f.engineDone = nil
	}()

	err = f.defineRoundVotes()
	if err != nil {
		return fmt.Errorf("defining round votes: %w", err)
	}

	err = f.finalizeRound()
	if err != nil {
		return fmt.Errorf("finalising round: %w", err)
	}

	return nil
}

func (f *finalizationEngine) defineRoundVotes() error {
	gossipInterval := f.grandpaService.interval
	determinePrevoteTimer := time.NewTimer(2 * gossipInterval)
	determinePrecommitTimer := time.NewTimer(4 * gossipInterval)

	var precommited bool = false

	for !precommited {
		select {
		case <-f.stopCh:
			if !determinePrevoteTimer.Stop() {
				<-determinePrevoteTimer.C
			}

			if !determinePrecommitTimer.Stop() {
				<-determinePrecommitTimer.C
			}

			return nil

		case <-determinePrevoteTimer.C:
			alreadyCompletable, err := f.grandpaService.checkRoundCompletable()
			if err != nil {
				return fmt.Errorf("checking round is completable: %w", err)
			}

			if alreadyCompletable {
				f.actionCh <- alreadyFinalized

				if !determinePrecommitTimer.Stop() {
					<-determinePrecommitTimer.C
				}

				return nil
			}

			f.actionCh <- determinePrevote

		case <-determinePrecommitTimer.C:
			alreadyCompletable, err := f.grandpaService.checkRoundCompletable()
			if err != nil {
				return fmt.Errorf("checking round is completable: %w", err)
			}

			if alreadyCompletable {
				f.actionCh <- alreadyFinalized
				return nil
			}

			prevoteGrandpaGhost, err := f.grandpaService.getPreVotedBlock()
			if err != nil {
				return fmt.Errorf("getting grandpa ghost: %w", err)
			}

			total, err := f.grandpaService.getTotalVotesForBlock(prevoteGrandpaGhost.Hash, prevote)
			if err != nil {
				return fmt.Errorf("getting grandpa ghost: %w", err)
			}

			if total <= f.grandpaService.state.threshold() {
				determinePrecommitTimer.Reset(4 * gossipInterval)
				continue
			}

			latestFinalizedHash := f.grandpaService.head.Hash()
			isDescendant, err := f.grandpaService.blockState.IsDescendantOf(
				latestFinalizedHash, prevoteGrandpaGhost.Hash)
			if err != nil {
				return fmt.Errorf("checking grandpa ghost ancestry: %w", err)
			}

			if !isDescendant {
				panic("block with supermajority does not belong to the latest finalized block chain")
			}

			f.actionCh <- determinePrecommit
			precommited = true
		}
	}

	return nil
}

func (f *finalizationEngine) finalizeRound() error {
	gossipInterval := f.grandpaService.interval
	attemptFinalizationTicker := time.NewTicker(gossipInterval / 2)
	defer attemptFinalizationTicker.Stop()

	for {
		completable, err := f.grandpaService.checkRoundCompletable()
		if err != nil {
			return fmt.Errorf("checking round is completable: %w", err)
		}

		if completable {
			f.actionCh <- alreadyFinalized
			return nil
		}

		finalizable, err := f.grandpaService.attemptToFinalize()
		if err != nil {
			return fmt.Errorf("attempting to finalize: %w", err)
		}

		if finalizable {
			f.actionCh <- finalize
			return nil
		}

		select {
		case <-f.stopCh:
			return nil
		case <-attemptFinalizationTicker.C:
		}
	}
}
