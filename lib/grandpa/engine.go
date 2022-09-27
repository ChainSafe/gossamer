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
	votingRound        *handleVotingRound
	finalizationEngine *finalizationEngine

	wg             sync.WaitGroup
	observableErrs chan error
	stopCh         chan struct{}
}

func newFinalizationHandler(service *Service) *finalizationHandler {
	finalizationEngine := newFinalizationEngine(service)
	votingRound := newHandleVotingRound(service, finalizationEngine.actionCh)

	return &finalizationHandler{
		votingRound:        votingRound,
		finalizationEngine: finalizationEngine,
		observableErrs:     make(chan error),
		stopCh:             make(chan struct{}),
	}
}

func (fh *finalizationHandler) Start() (errsCh <-chan error, err error) {

	finalizationEngErrsCh, err := fh.finalizationEngine.Start()
	if err != nil {
		return nil, fmt.Errorf("starting finalization engine: %w", err)
	}

	handleVotingErrsCh, err := fh.votingRound.Start()
	if err != nil {
		return nil, fmt.Errorf("starting handle voting: %w", err)
	}

	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()
		for err := range handleVotingErrsCh {
			fh.observableErrs <- fmt.Errorf("%w: %s", errVotingRound, err)
		}
	}()

	go func() {
		defer wg.Done()
		for err := range finalizationEngErrsCh {
			fh.observableErrs <- err
		}
	}()

	return fh.observableErrs, nil
}

func (fh *finalizationHandler) Stop() (err error) {
	stopErrs := make([]error, 0, 2)

	err = fh.votingRound.Stop()
	if err != nil {
		stopErrs = append(stopErrs, fmt.Errorf("stopping voting round: %w", err))
	}

	err = fh.finalizationEngine.Stop()
	if err != nil {
		stopErrs = append(stopErrs, fmt.Errorf("stopping finalization engine: %w", err))
	}

	close(fh.stopCh)
	fh.wg.Wait()

	close(fh.observableErrs)

	if len(stopErrs) > 0 {
		return stopErrs[0]
	}

	return nil
}

var errTimeoutWhileStoping = errors.New("timeout while stopping")
var errEngineChannelClosed = errors.New("engine channel closed")

type handleVotingRound struct {
	grandpaService *Service
	errsCh         chan error

	finalizationEngineCh <-chan action
	stopCh               chan<- struct{}
	engineDone           <-chan struct{}
}

func newHandleVotingRound(service *Service, finalizationEngineCh <-chan action) *handleVotingRound {
	return &handleVotingRound{
		grandpaService:       service,
		errsCh:               make(chan error),
		stopCh:               make(chan struct{}),
		engineDone:           make(chan struct{}),
		finalizationEngineCh: finalizationEngineCh,
	}
}

func (h *handleVotingRound) Start() (errsCh <-chan error, err error) {
	err = h.grandpaService.initiateRound()
	if err != nil {
		return nil, fmt.Errorf("initiating round: %w", err)
	}

	go h.playGrandpaRound()
	return h.errsCh, nil
}

func (h *handleVotingRound) Stop() (err error) {
	close(h.stopCh)

	timeout := 5 * time.Second
	select {
	case <-h.engineDone:
	case <-time.After(timeout):
		return fmt.Errorf("%w", errTimeoutWhileStoping)
	}

	close(h.errsCh)
	return
}

// playGrandpaRound executes a round of GRANDPA
// at the end of this round, a block will be finalised.
func (h *handleVotingRound) playGrandpaRound() {
	start := time.Now()

	logger.Debugf("starting round %d with set id %d",
		h.grandpaService.state.round, h.grandpaService.state.setID)

	for { //nolint:gosimple
		select {
		case <-h.grandpaService.ctx.Done():
			err := h.grandpaService.ctx.Err()
			if err != nil {
				h.errsCh <- err
			}
			return

		case action, ok := <-h.finalizationEngineCh:
			if !ok {
				return
			}

			switch action {
			case determinePrevote:
				isPrimary, err := h.grandpaService.handleIsPrimary()
				if err != nil {
					h.errsCh <- fmt.Errorf("handling primary: %w", err)
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

				logger.Warnf("sending pre-vote message: {%v}", precommitMessage)

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
				return
			}
		}
	}
}

// actions that should take place accordingly to votes the
// finalisation engine knows about
type action byte

const (
	determinePrevote action = iota
	determinePrecommit
	alreadyFinalized
	finalize
)

type finalizationEngine struct {
	grandpaService *Service

	stopCh     chan struct{}
	engineDone chan struct{}
	actionCh   chan action
	errsCh     chan error
}

func newFinalizationEngine(service *Service) *finalizationEngine {
	return &finalizationEngine{
		grandpaService: service,
		actionCh:       make(chan action),
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

	timeout := 5 * time.Second
	select {
	case <-f.engineDone:
	case <-time.After(timeout):
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
			f.actionCh <- determinePrevote

		case <-determinePrecommitTimer.C:
			prevoteGrandpaGhost, err := f.grandpaService.getPreVotedBlock()
			if errors.Is(err, ErrNoGHOST) {
				determinePrecommitTimer.Reset(gossipInterval * 4)
				break
			} else if err != nil {
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
				determinePrecommitTimer.Reset(gossipInterval * 4)
				break
			}

			f.actionCh <- determinePrecommit
			precommited = true
		}
	}

	attemptFinalizationTicker := time.NewTicker(gossipInterval / 2)
	defer attemptFinalizationTicker.Stop()

	for {
		select {
		case <-stop:
			return

		case <-attemptFinalizationTicker.C:
			alreadyCompletable, err := f.grandpaService.checkRoundAlreadyCompletable()
			if err != nil {
				f.errsCh <- fmt.Errorf("checking round is completable: %w", err)
				continue
			}

			if alreadyCompletable {
				f.actionCh <- alreadyFinalized
				return
			}

			finalizable, err := f.grandpaService.attemptToFinalize()
			if err != nil {
				f.errsCh <- fmt.Errorf("attempting to finalize: %w", err)
				continue
			}

			if !finalizable {
				continue
			}

			f.actionCh <- finalize
			return
		}
	}

}
