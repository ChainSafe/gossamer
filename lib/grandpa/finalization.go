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

var (
	errServicesStopFailed       = errors.New("services stop failed")
	errvotingRoundHandlerFailed = errors.New("voting round ephemeral failed")
	errFinalizationEngineFailed = errors.New("finalisation engine ephemeral failed")
)

type ephemeralService interface {
	Run() error
	Stop() error
}

type finalisationHandler struct {
	servicesLock       sync.Mutex
	finalizationEngine ephemeralService
	votingRound        ephemeralService

	// newServices is a constructor function which takes care to instantiate
	// and return the services needed to finalize a round, those services
	// are ephemeral services with a lifetime of a round
	newServices   func() (engine, voting ephemeralService)
	initiateRound func() error

	stopCh      chan struct{}
	handlerDone chan struct{}
}

func newFinalisationHandler(service *Service) *finalisationHandler {
	return &finalisationHandler{
		newServices: func() (engine, voting ephemeralService) {
			finalizationEngine := newFinalizationEngine(service)
			votingRound := newvotingRoundHandler(service, finalizationEngine.actionCh)
			return finalizationEngine, votingRound
		},
		initiateRound: service.initiateRound,
		stopCh:        make(chan struct{}),
		handlerDone:   make(chan struct{}),
	}
}

func (fh *finalisationHandler) Start() (<-chan error, error) {
	errorCh := make(chan error)
	go fh.run(errorCh)

	return errorCh, nil
}

func (fh *finalisationHandler) run(errorCh chan<- error) {
	defer func() {
		close(errorCh)
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
			errorCh <- fmt.Errorf("initiating round: %w", err)
			return
		}

		err = fh.runEphemeralServices()
		if err != nil {
			errorCh <- fmt.Errorf("running ephemeral services: %w", err)
			return
		}
	}
}

func (fh *finalisationHandler) stop() (err error) {
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

	switch {
	case finalizationEngErr != nil && votingRoundErr != nil:
		return fmt.Errorf("%w: %s; %s", errServicesStopFailed, finalizationEngErr, votingRoundErr)
	case finalizationEngErr != nil:
		return fmt.Errorf("%w: %s", errServicesStopFailed, finalizationEngErr)
	case votingRoundErr != nil:
		return fmt.Errorf("%w: %s", errServicesStopFailed, votingRoundErr)
	}

	return nil
}

func (fh *finalisationHandler) Stop() (err error) {
	close(fh.stopCh)
	<-fh.handlerDone

	return fh.stop()
}

// runEphemeralServices starts the two ephemeral services that handle the
// votes for the current round, and returns with nil when the two
// service runs succeed.
// If any service run fails, the other service run is stopped and
// an error is returned. The function returns nil is the finalisation
// handler is stopped.
func (fh *finalisationHandler) runEphemeralServices() error {
	fh.servicesLock.Lock()
	fh.finalizationEngine, fh.votingRound = fh.newServices()
	fh.servicesLock.Unlock()

	finalizationEngineErr := make(chan error)
	go func() {
		finalizationEngineErr <- fh.finalizationEngine.Run()
	}()

	votingRoundErr := make(chan error)
	go func() {
		votingRoundErr <- fh.votingRound.Run()
	}()

	for {
		select {
		case <-fh.stopCh:
			return nil

		case err := <-votingRoundErr:
			if err == nil {
				votingRoundErr = nil
				// go out from the select case
				break
			}

			stopErr := fh.finalizationEngine.Stop()
			if stopErr != nil {
				logger.Warnf("stopping finalisation engine: %s", stopErr)
			}
			return fmt.Errorf("%w: %s", errvotingRoundHandlerFailed, err)

		case err := <-finalizationEngineErr:
			if err == nil {
				finalizationEngineErr = nil
				// go out from the select case
				break
			}

			stopErr := fh.votingRound.Stop()
			if stopErr != nil {
				logger.Warnf("stopping voting round: %s", stopErr)
			}

			return fmt.Errorf("%w: %s", errFinalizationEngineFailed, err)
		}

		finish := votingRoundErr == nil && finalizationEngineErr == nil
		if finish {
			return nil
		}
	}
}

// votingRoundHandler interacts with finalizationEngine service
// executing the actions based on what it receives throuhg channel
type votingRoundHandler struct {
	grandpaService       *Service
	finalizationEngineCh <-chan engineAction
	stopCh               chan struct{}
	engineDone           chan struct{}
}

func newvotingRoundHandler(service *Service, finalizationEngineCh <-chan engineAction) *votingRoundHandler {
	return &votingRoundHandler{
		grandpaService:       service,
		stopCh:               make(chan struct{}),
		engineDone:           make(chan struct{}),
		finalizationEngineCh: finalizationEngineCh,
	}
}

func (h *votingRoundHandler) Stop() (err error) {
	close(h.stopCh)
	<-h.engineDone

	return nil
}

func (h *votingRoundHandler) Run() (err error) {
	defer close(h.engineDone)

	start := time.Now()

	logger.Debugf("starting round %d with set id %d",
		h.grandpaService.state.round, h.grandpaService.state.setID)

	for {
		select {
		case <-h.stopCh:
			return nil

		case action := <-h.finalizationEngineCh:
			switch action {
			case determinePrevote:
				isPrimary, err := h.grandpaService.handleIsPrimary()
				if err != nil {
					return fmt.Errorf("handling primary: %w", err)
				}

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
				err = h.grandpaService.sendPrevoteMessage(prevoteMessage)
				if err != nil {
					return fmt.Errorf("sending pre-vote message: %w", err)
				}

			case determinePrecommit:
				preCommit, err := h.grandpaService.determinePreCommit()
				if err != nil {
					return fmt.Errorf("determining pre-commit: %w", err)
				}

				signedPreCommit, precommitMessage, err :=
					h.grandpaService.createSignedVoteAndVoteMessage(preCommit, precommit)
				if err != nil {
					return fmt.Errorf("creating signed vote: %w", err)
				}

				h.grandpaService.precommits.Store(h.grandpaService.publicKeyBytes(), signedPreCommit)
				logger.Debugf("sending pre-commit message: {%v}", precommitMessage)
				err = h.grandpaService.sendPrecommitMessage(precommitMessage)
				if err != nil {
					logger.Errorf("sending pre-commit message: %s", err)
				}

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
	close(f.stopCh)
	<-f.engineDone

	close(f.actionCh)
	return nil
}

func (f *finalizationEngine) Run() (err error) {
	defer close(f.engineDone)

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

	precommited := false

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
