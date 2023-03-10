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
	errfinalisationEngineFailed = errors.New("finalisation engine ephemeral failed")
)

type ephemeralService interface {
	Run() error
	Stop() error
}

type finalisationHandler struct {
	servicesLock       sync.Mutex
	finalisationEngine ephemeralService
	votingRound        ephemeralService

	// newServices is a constructor function which takes care to instantiate
	// and return the services needed to finalize a round, those services
	// are ephemeral services with a lifetime of a round
	newServices   func() (engine, voting ephemeralService)
	initiateRound func() error

	stopCh      chan struct{}
	handlerDone chan struct{}

	firstRun bool
}

func newFinalisationHandler(service *Service) *finalisationHandler {
	return &finalisationHandler{
		newServices: func() (engine, voting ephemeralService) {
			finalisationEngine := newfinalisationEngine(service)
			votingRound := newvotingRoundHandler(service, finalisationEngine.actionCh)
			return finalisationEngine, votingRound
		},
		initiateRound: service.initiateRound,
		stopCh:        make(chan struct{}),
		handlerDone:   make(chan struct{}),
		firstRun:      true,
	}
}

func (fh *finalisationHandler) Start() (<-chan error, error) {
	errorCh := make(chan error)
	ready := make(chan struct{})

	go fh.run(errorCh, ready)
	<-ready

	return errorCh, nil
}

func (fh *finalisationHandler) run(errorCh chan<- error, ready chan<- struct{}) {
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

		err = fh.runEphemeralServices(ready)
		if err != nil {
			errorCh <- fmt.Errorf("running ephemeral services: %w", err)
			return
		}
	}
}

func (fh *finalisationHandler) stop() (err error) {
	fh.servicesLock.Lock()
	defer fh.servicesLock.Unlock()

	finalisationEngineErrCh := make(chan error)
	go func() {
		finalisationEngineErrCh <- fh.finalisationEngine.Stop()
	}()

	votingRoundErrCh := make(chan error)
	go func() {
		votingRoundErrCh <- fh.votingRound.Stop()
	}()

	finalisationEngErr := <-finalisationEngineErrCh
	votingRoundErr := <-votingRoundErrCh

	switch {
	case finalisationEngErr != nil && votingRoundErr != nil:
		return fmt.Errorf("%w: %s; %s", errServicesStopFailed, finalisationEngErr, votingRoundErr)
	case finalisationEngErr != nil:
		return fmt.Errorf("%w: %s", errServicesStopFailed, finalisationEngErr)
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
func (fh *finalisationHandler) runEphemeralServices(ready chan<- struct{}) error {
	fh.servicesLock.Lock()
	fh.finalisationEngine, fh.votingRound = fh.newServices()
	fh.servicesLock.Unlock()

	if fh.firstRun {
		fh.firstRun = false
		close(ready)
	}

	finalisationEngineErr := make(chan error)
	go func() {
		finalisationEngineErr <- fh.finalisationEngine.Run()
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

			stopErr := fh.finalisationEngine.Stop()
			if stopErr != nil {
				logger.Warnf("stopping finalisation engine: %s", stopErr)
			}
			return fmt.Errorf("%w: %s", errvotingRoundHandlerFailed, err)

		case err := <-finalisationEngineErr:
			if err == nil {
				finalisationEngineErr = nil
				// go out from the select case
				break
			}

			stopErr := fh.votingRound.Stop()
			if stopErr != nil {
				logger.Warnf("stopping voting round: %s", stopErr)
			}

			return fmt.Errorf("%w: %s", errfinalisationEngineFailed, err)
		}

		finish := votingRoundErr == nil && finalisationEngineErr == nil
		if finish {
			return nil
		}
	}
}

// votingRoundHandler interacts with finalisationEngine service
// executing the actions based on what it receives throuhg channel
type votingRoundHandler struct {
	grandpaService       *Service
	finalisationEngineCh <-chan engineAction
	stopCh               chan struct{}
	engineDone           chan struct{}
}

func newvotingRoundHandler(service *Service, finalisationEngineCh <-chan engineAction) *votingRoundHandler {
	return &votingRoundHandler{
		grandpaService:       service,
		stopCh:               make(chan struct{}),
		engineDone:           make(chan struct{}),
		finalisationEngineCh: finalisationEngineCh,
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

		case action := <-h.finalisationEngineCh:
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

type finalisationEngine struct {
	grandpaService *Service
	stopCh         chan struct{}
	engineDone     chan struct{}
	actionCh       chan engineAction
}

func newfinalisationEngine(service *Service) *finalisationEngine {
	return &finalisationEngine{
		grandpaService: service,
		actionCh:       make(chan engineAction),
		stopCh:         make(chan struct{}),
		engineDone:     make(chan struct{}),
	}
}

func (f *finalisationEngine) Stop() (err error) {
	close(f.stopCh)
	<-f.engineDone

	close(f.actionCh)
	return nil
}

var errFinalisationEngineStopped = errors.New("finalisation engine stopped")

func (f *finalisationEngine) Run() (err error) {
	defer close(f.engineDone)

	err = f.defineRoundVotes()
	if errors.Is(err, errFinalisationEngineStopped) {
		return nil
	} else if err != nil {
		return fmt.Errorf("defining round votes: %w", err)
	}

	err = f.finalizeRound()
	if err != nil {
		return fmt.Errorf("finalising round: %w", err)
	}

	return nil
}

func (f *finalisationEngine) defineRoundVotes() (err error) {
	gossipInterval := f.grandpaService.interval
	determinePrevoteTimer := time.NewTimer(2 * gossipInterval)
	determinePrecommitTimer := time.NewTimer(4 * gossipInterval)

	precommited := false

	for !precommited {
		select {
		case <-f.stopCh:
			determinePrevoteTimer.Stop()
			determinePrecommitTimer.Stop()
			return fmt.Errorf("%w", errFinalisationEngineStopped)

		case <-determinePrevoteTimer.C:
			alreadyCompletable, err := f.grandpaService.checkRoundCompletable()
			if err != nil {
				return fmt.Errorf("checking round is completable: %w", err)
			}

			if alreadyCompletable {
				f.actionCh <- alreadyFinalized

				determinePrecommitTimer.Stop()
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

func (f *finalisationEngine) finalizeRound() error {
	gossipInterval := f.grandpaService.interval
	attemptfinalisationTicker := time.NewTicker(gossipInterval / 2)
	defer attemptfinalisationTicker.Stop()

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
		case <-attemptfinalisationTicker.C:
		}
	}
}
