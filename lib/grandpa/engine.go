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

var errVotingRound = errors.New("voting round error")

type finalizationHandler struct {
	timeoutStop    time.Duration
	grandpaService *Service
	observableErrs chan error
	stopCh         chan struct{}
	handlerDone    chan struct{}
}

func newFinalizationHandler(service *Service) *finalizationHandler {
	return &finalizationHandler{
		timeoutStop:    5 * time.Second,
		grandpaService: service,
		observableErrs: make(chan error),
		stopCh:         make(chan struct{}),
		handlerDone:    make(chan struct{}),
	}
}

func (fh *finalizationHandler) stopServices(finalizationEngine *finalizationEngine,
	votingRound *handleVotingRound) error {
	stopErr := make([]error, 2)
	err := finalizationEngine.Stop()
	if err != nil {
		stopErr = append(stopErr, fmt.Errorf("stopping finalisation engine: %w", err))
	}

	err = votingRound.Stop()
	if err != nil {
		stopErr = append(stopErr, fmt.Errorf("stopping voting round: %w", err))
	}

	if len(stopErr) > 0 {
		return stopErr[0]
	}

	return nil
}

var errRunFinalization = errors.New("runFinalization error")

func (fh *finalizationHandler) runFinalization() {
	defer close(fh.handlerDone)

	for {
		err := fh.grandpaService.initiateRound()
		if err != nil {
			fh.observableErrs <- fmt.Errorf("%w: initiating round: %s", errRunFinalization, err)
			return
		}

		finalizationEngine := newFinalizationEngine(fh.grandpaService)
		votingRound := newHandleVotingRound(fh.grandpaService, finalizationEngine.actionCh)

		finalizationEngErrsCh, err := finalizationEngine.Start()
		if err != nil {
			fh.observableErrs <- fmt.Errorf("%w: finalisation engine: %s", errRunFinalization, err)
			return
		}

		handleVotingErrsCh, err := votingRound.Start()
		if err != nil {
			fh.observableErrs <- fmt.Errorf("%w: voting round: %s", errRunFinalization, err)
			return

		}

		innerWg := sync.WaitGroup{}
		innerWg.Add(2)

		go func() {
			defer innerWg.Done()
			for err := range handleVotingErrsCh {
				if err != nil {
					fh.observableErrs <- fmt.Errorf("%w: %s", errVotingRound, err)
					continue
				}
				return
			}
		}()

		go func() {
			defer innerWg.Done()
			for err := range finalizationEngErrsCh {
				if err != nil {
					fh.observableErrs <- err
					continue
				}
				return
			}
		}()

		waitServicesCh := make(chan struct{})
		go func() {
			defer close(waitServicesCh)
			innerWg.Wait()
		}()

		select {
		case <-waitServicesCh:
			err := fh.stopServices(finalizationEngine, votingRound)
			if err != nil {
				fh.observableErrs <- err
				return
			}

		case <-fh.grandpaService.ctx.Done():
			err := fh.stopServices(finalizationEngine, votingRound)
			innerWg.Wait()

			if err != nil {
				fh.observableErrs <- err
			}
			return

		case <-fh.stopCh:
			err := fh.stopServices(finalizationEngine, votingRound)
			innerWg.Wait()

			if err != nil {
				fh.observableErrs <- err
			}
			return
		}
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

var errTimeoutWhileStoping = errors.New("timeout while stopping")

type handleVotingRound struct {
	grandpaService *Service
	errsCh         chan error

	timeoutStop          time.Duration
	finalizationEngineCh <-chan engineAction
	stopCh               chan struct{}
	engineDone           chan struct{}
}

func newHandleVotingRound(service *Service, finalizationEngineCh <-chan engineAction) *handleVotingRound {
	return &handleVotingRound{
		timeoutStop:          5 * time.Second,
		grandpaService:       service,
		errsCh:               make(chan error),
		stopCh:               make(chan struct{}),
		engineDone:           make(chan struct{}),
		finalizationEngineCh: finalizationEngineCh,
	}
}

func (h *handleVotingRound) Start() (errsCh <-chan error, err error) {
	go h.playGrandpaRound()
	return h.errsCh, nil
}

func (h *handleVotingRound) Stop() (err error) {
	close(h.stopCh)

	select {
	case <-h.engineDone:
	case <-time.After(h.timeoutStop):
		return fmt.Errorf("%w", errTimeoutWhileStoping)
	}

	close(h.errsCh)
	return
}

// playGrandpaRound executes a round of GRANDPA
// at the end of this round, a block will be finalised.
func (h *handleVotingRound) playGrandpaRound() {
	defer close(h.engineDone)

	start := time.Now()

	logger.Debugf("starting round %d with set id %d",
		h.grandpaService.state.round, h.grandpaService.state.setID)

	for {
		select {
		case <-h.stopCh:
			return

		case action, ok := <-h.finalizationEngineCh:
			if !ok {
				return
			}

			switch action {
			case determinePrevote:
				isPrimary, err := h.grandpaService.handleIsPrimary()
				if err != nil {
					h.errsCh <- fmt.Errorf("determining pre-vote: %w", err)
					return
				}

				// broadcast pre-vote
				preVote, err := h.grandpaService.determinePreVote()
				if err != nil {
					h.errsCh <- fmt.Errorf("determining pre-vote: %w", err)
					return
				}

				signedpreVote, prevoteMessage, err :=
					h.grandpaService.createSignedVoteAndVoteMessage(preVote, prevote)
				if err != nil {
					h.errsCh <- fmt.Errorf("creating signed vote: %w", err)
					return
				}

				if !isPrimary {
					h.grandpaService.prevotes.Store(h.grandpaService.publicKeyBytes(), signedpreVote)
				}

				logger.Warnf("sending pre-vote message: {%v}", prevoteMessage)
				h.grandpaService.sendPrevoteMessage(prevoteMessage)

			case determinePrecommit:
				preComit, err := h.grandpaService.determinePreCommit()
				if err != nil {
					h.errsCh <- fmt.Errorf("determining pre-commit: %w", err)
					return
				}

				signedpreComit, precommitMessage, err :=
					h.grandpaService.createSignedVoteAndVoteMessage(preComit, precommit)
				if err != nil {
					h.errsCh <- fmt.Errorf("creating signed vote: %w", err)
					return
				}

				logger.Warnf("sending pre-commit message: {%v}", precommitMessage)

				h.grandpaService.precommits.Store(h.grandpaService.publicKeyBytes(), signedpreComit)

				h.grandpaService.sendPrecommitMessage(precommitMessage)

			case finalize:
				commitMessage, err := h.grandpaService.newCommitMessage(
					h.grandpaService.head, h.grandpaService.state.round, h.grandpaService.state.setID)
				if err != nil {
					h.errsCh <- fmt.Errorf("creating commit message: %w", err)
					return
				}

				commitConsensusMessage, err := commitMessage.ToConsensusMessage()
				if err != nil {
					h.errsCh <- fmt.Errorf("transforming commit into consensus message: %w", err)
					return
				}

				logger.Debugf("sending commit message: %v", commitMessage)

				h.grandpaService.network.GossipMessage(commitConsensusMessage)
				h.grandpaService.telemetry.SendMessage(telemetry.NewAfgFinalizedBlocksUpTo(
					h.grandpaService.head.Hash(),
					fmt.Sprint(h.grandpaService.head.Number),
				))

				logger.Debugf("round completed in %s", time.Since(start))
				h.errsCh <- nil
				return

			case alreadyFinalized:
				logger.Debugf("round completed in %s", time.Since(start))
				h.errsCh <- nil
				return
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
	errsCh      chan error
}

func newFinalizationEngine(service *Service) *finalizationEngine {
	return &finalizationEngine{
		grandpaService: service,
		timeoutStop:    5 * time.Second,
		actionCh:       make(chan engineAction),
		errsCh:         make(chan error),
		stopCh:         make(chan struct{}),
		engineDone:     make(chan struct{}),
	}
}

func (f *finalizationEngine) Start() (errsCh <-chan error, err error) {
	go f.playFinalization(f.grandpaService.interval, f.stopCh)
	return f.errsCh, nil
}

func (f *finalizationEngine) Stop() (err error) {
	close(f.stopCh)

	select {
	case <-f.engineDone:
	case <-time.After(f.timeoutStop):
		return fmt.Errorf("%w", errTimeoutWhileStoping)
	}

	close(f.errsCh)

	return nil
}

func (f *finalizationEngine) playFinalization(gossipInterval time.Duration, stop <-chan struct{}) {
	defer close(f.engineDone)

	determinePrevoteTimer := time.NewTimer(gossipInterval * 2)
	determinePrecommitTimer := time.NewTimer(gossipInterval * 4)

	var precommited bool = false

	for !precommited {
		select {
		case <-stop:
			if !determinePrevoteTimer.Stop() {
				<-determinePrevoteTimer.C
			}

			if !determinePrecommitTimer.Stop() {
				<-determinePrecommitTimer.C
			}

			return

		case <-determinePrevoteTimer.C:
			alreadyCompletable, err := f.grandpaService.checkRoundAlreadyCompletable()
			if err != nil {
				f.errsCh <- fmt.Errorf("%w: checking round is completable: %s",
					errVotingRound, err)
				return
			}

			if alreadyCompletable {
				f.actionCh <- alreadyFinalized
				return
			}

			f.actionCh <- determinePrevote

		case <-determinePrecommitTimer.C:
			alreadyCompletable, err := f.grandpaService.checkRoundAlreadyCompletable()
			if err != nil {
				f.errsCh <- fmt.Errorf("%w: checking round is completable: %s",
					errVotingRound, err)
				return
			}

			if alreadyCompletable {
				f.actionCh <- alreadyFinalized
				return
			}

			prevoteGrandpaGhost, err := f.grandpaService.getPreVotedBlock()
			if err != nil {
				f.errsCh <- fmt.Errorf("getting grandpa ghost: %w", err)
				return
			}

			total, err := f.grandpaService.getTotalVotesForBlock(prevoteGrandpaGhost.Hash, prevote)
			if err != nil {
				f.errsCh <- fmt.Errorf("%w: getting grandpa ghost: %s", errVotingRound, err)
				return
			}

			if total <= f.grandpaService.state.threshold() {
				determinePrecommitTimer.Reset(gossipInterval * 4)
				break
			}

			latestFinalizedHash := f.grandpaService.head.Hash()
			isDescendant, err := f.grandpaService.blockState.IsDescendantOf(latestFinalizedHash, prevoteGrandpaGhost.Hash)
			if err != nil {
				f.errsCh <- fmt.Errorf("checking grandpa ghost ancestry: %w", err)
				return
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
			f.errsCh <- fmt.Errorf("checking round is completable: %w", err)
			return
		}

		if alreadyCompletable {
			f.actionCh <- alreadyFinalized
			f.errsCh <- nil
			return
		}

		finalizable, err := f.grandpaService.attemptToFinalize()
		if err != nil {
			f.errsCh <- fmt.Errorf("attempting to finalize: %w", err)
			return
		}

		if finalizable {
			f.actionCh <- finalize
			f.errsCh <- nil
			return
		}

		select {
		case <-stop:
			return
		case <-attemptFinalizationTicker.C:
		}
	}

}
