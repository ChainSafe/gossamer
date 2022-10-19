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

type ephemeralService interface {
	Start() error
	Stop() error
}

type finalizationHandler struct {
	servicesBuilder func() (engine, voting ephemeralService)
	timeoutStop     time.Duration
	initiateRound   func() error
	observableErrs  chan error
	stopCh          chan struct{}
	handlerDone     chan struct{}
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
		servicesBuilder: builder,
		timeoutStop:     5 * time.Second,
		initiateRound:   service.initiateRound,
		observableErrs:  make(chan error),
		stopCh:          make(chan struct{}),
		handlerDone:     make(chan struct{}),
	}
}

func (fh *finalizationHandler) Start() (errsCh <-chan error, err error) {
	go fh.runFinalization()
	return fh.observableErrs, nil
}

func (fh *finalizationHandler) Stop() (err error) {
	close(fh.stopCh)

	select {
	case <-fh.handlerDone:
	case <-time.After(fh.timeoutStop):
		return fmt.Errorf("%w", errTimeoutWhileStoping)
	}

	close(fh.observableErrs)
	return nil
}

func (fh *finalizationHandler) runFinalization() {
	defer close(fh.handlerDone)

	for {
		select {
		case <-fh.stopCh:
			return
		default:
		}

		err := fh.initiateRound()
		if err != nil {
			fh.observableErrs <- fmt.Errorf("initiating round: %w", err)
			return
		}

		err = fh.waitServices()
		if err != nil {
			fh.observableErrs <- err
			return
		}
	}
}

// waitServices will start the services and wait until they complete or
func (fh *finalizationHandler) waitServices() error {
	finalizationEngine, votingRound := fh.servicesBuilder()

	finalizationEngineErr := make(chan error)
	go func() {
		defer close(finalizationEngineErr)
		err := finalizationEngine.Start()
		if err != nil {
			finalizationEngineErr <- err
		}
	}()

	votingRoundErr := make(chan error)
	go func() {
		defer close(votingRoundErr)
		err := votingRound.Start()
		if err != nil {
			votingRoundErr <- err
		}
	}()

	for {
		select {
		case <-fh.stopCh:
			stopWg := new(sync.WaitGroup)
			stopWg.Add(2)

			go func() {
				defer stopWg.Done()
				votingRound.Stop()
				<-votingRoundErr
			}()

			go func() {
				defer stopWg.Done()
				finalizationEngine.Stop()
				<-finalizationEngineErr
			}()

			stopWg.Wait()
			return nil

		case err := <-votingRoundErr:
			finalizationEngine.Stop()
			engineErr := <-finalizationEngineErr
			if err == nil && engineErr != nil {
				return engineErr
			}

			return err
		case err := <-finalizationEngineErr:
			votingRound.Stop()
			votingErr := <-votingRoundErr
			if err == nil && votingErr != nil {
				return votingErr
			}

			return err
		}
	}
}

var errTimeoutWhileStoping = errors.New("timeout while stopping")

type handleVotingRound struct {
	grandpaService       *Service
	timeoutStop          time.Duration
	finalizationEngineCh <-chan engineAction
	stopCh               chan struct{}
	engineDone           chan struct{}
}

func newHandleVotingRound(service *Service, finalizationEngineCh <-chan engineAction) *handleVotingRound {
	return &handleVotingRound{
		timeoutStop:          5 * time.Second,
		grandpaService:       service,
		stopCh:               make(chan struct{}),
		engineDone:           make(chan struct{}),
		finalizationEngineCh: finalizationEngineCh,
	}
}

func (h *handleVotingRound) Stop() (err error) {
	defer func() {
		fmt.Println("STOPPING VOTING ROUND")
	}()
	close(h.stopCh)

	select {
	case <-h.engineDone:
	case <-time.After(h.timeoutStop):
		return fmt.Errorf("%w", errTimeoutWhileStoping)
	}

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
		fmt.Println("RETURNING VOTING ROUND")
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

				logger.Warnf("sending pre-vote message: {%v}", prevoteMessage)
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

				logger.Warnf("sending pre-commit message: {%v}", precommitMessage)
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

	timeoutStop time.Duration
	stopCh      chan struct{}
	engineDone  chan struct{}
	actionCh    chan engineAction
}

func newFinalizationEngine(service *Service) *finalizationEngine {
	return &finalizationEngine{
		grandpaService: service,
		timeoutStop:    5 * time.Second,
		actionCh:       make(chan engineAction),
		stopCh:         make(chan struct{}),
		engineDone:     make(chan struct{}),
	}
}

func (f *finalizationEngine) Stop() (err error) {
	defer func() {
		fmt.Println("STOPPING FINALIZATION ENGINE")
	}()
	close(f.stopCh)

	select {
	case <-f.engineDone:
	case <-time.After(f.timeoutStop):
		return fmt.Errorf("%w", errTimeoutWhileStoping)
	}

	close(f.actionCh)
	return nil
}

func (f *finalizationEngine) Start() (err error) {
	defer func() {
		close(f.engineDone)
		fmt.Println("RETURNING FINALIZATION ENGINE")
	}()

	gossipInterval := f.grandpaService.interval
	determinePrevoteTimer := time.NewTimer(gossipInterval * 2)
	determinePrecommitTimer := time.NewTimer(gossipInterval * 4)

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
			alreadyCompletable, err := f.grandpaService.checkRoundAlreadyCompletable()
			if err != nil {
				return fmt.Errorf("checking round is completable: %w", err)
			}

			if alreadyCompletable {
				f.actionCh <- alreadyFinalized
				return nil
			}

			f.actionCh <- determinePrevote

		case <-determinePrecommitTimer.C:
			alreadyCompletable, err := f.grandpaService.checkRoundAlreadyCompletable()
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
				determinePrecommitTimer.Reset(gossipInterval * 4)
				break
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

	attemptFinalizationTicker := time.NewTicker(gossipInterval / 2)
	defer attemptFinalizationTicker.Stop()

	for {
		alreadyCompletable, err := f.grandpaService.checkRoundAlreadyCompletable()
		if err != nil {
			return fmt.Errorf("checking round is completable: %w", err)
		}

		if alreadyCompletable {
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
