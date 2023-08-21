package dispute

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	parachain "github.com/ChainSafe/gossamer/dot/parachain/runtime"

	"github.com/ChainSafe/gossamer/dot/parachain/dispute/overseer"
	"github.com/ChainSafe/gossamer/dot/parachain/dispute/types"
	parachainTypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

// CandidateComparator comparator for ordering of disputes for candidate.
type CandidateComparator struct {
	relayParentBlockNumber *uint32
	candidateHash          common.Hash
}

// NewCandidateComparator creates a new CandidateComparator.
func NewCandidateComparator(relayParentBlockNumber *uint32,
	receipt parachainTypes.CandidateReceipt,
) (CandidateComparator, error) {

	candidateHash, err := receipt.Hash()
	if err != nil {
		return CandidateComparator{}, fmt.Errorf("hash candidate receipt: %w", err)
	}

	return CandidateComparator{
		relayParentBlockNumber: relayParentBlockNumber,
		candidateHash:          candidateHash,
	}, nil
}

// ParticipationRequest a dispute participation request
type ParticipationRequest struct {
	candidateHash    common.Hash
	candidateReceipt parachainTypes.CandidateReceipt
	session          parachainTypes.SessionIndex
	//TODO: requestTimer for metrics
}

// ParticipationStatement is a statement as result of the validation process.
type ParticipationStatement struct {
	Session          parachainTypes.SessionIndex
	CandidateHash    common.Hash
	CandidateReceipt parachainTypes.CandidateReceipt
	Outcome          types.ParticipationOutcome
}

// Participation keeps track of the disputes we need to participate in.
type Participation interface {
	// Queue a dispute for the node to participate in
	Queue(context overseer.Context, request ParticipationRequest, priority ParticipationPriority) error

	// Clear clears a participation request. This is called when we have the dispute result.
	Clear(candidateHash common.Hash) error

	// ProcessActiveLeavesUpdate processes an active leaves update
	ProcessActiveLeavesUpdate(update overseer.ActiveLeavesUpdate) error

	// BumpPriority bumps the priority for the given receipts
	BumpPriority(ctx overseer.Context, receipts []parachainTypes.CandidateReceipt) error
}

type block struct {
	Number uint32
	Hash   common.Hash
}

// ParticipationHandler handles dispute participation.
type ParticipationHandler struct {
	runningParticipation sync.Map
	workers              atomic.Int32

	queue       Queue
	sender      overseer.Sender // TODO: revisit this once we have the overseer
	recentBlock *block

	runtime parachain.RuntimeInstance

	//TODO: metrics
}

const MaxParallelParticipation = 3

func (p *ParticipationHandler) Queue(ctx overseer.Context,
	request ParticipationRequest,
	priority ParticipationPriority,
) error {
	if _, ok := p.runningParticipation.Load(request.candidateHash); ok {
		return nil
	}

	// if we already have a recent block, participate right away
	if p.recentBlock != nil && p.numberOfWorkers() < MaxParallelParticipation {
		p.forkParticipation(&request, p.recentBlock.Hash)
		return nil
	}

	blockNumber, err := getBlockNumber(ctx.Sender, request.candidateReceipt)
	if err != nil {
		return fmt.Errorf("get block number: %w", err)
	}

	comparator, err := NewCandidateComparator(&blockNumber, request.candidateReceipt)
	if err != nil {
		return fmt.Errorf("create candidate comparator: %w", err)
	}

	if err := p.queue.Queue(comparator, &request, priority); err != nil {
		return fmt.Errorf("queue ParticipationHandler request: %w", err)
	}

	return nil
}

func (p *ParticipationHandler) Clear(candidateHash common.Hash) error {
	p.runningParticipation.Delete(candidateHash)
	p.workers.Add(-1)

	if p.recentBlock == nil {
		panic("we never ever reset recentBlock to nil and we already received a result, so it must have been set before. qed")
	}

	p.dequeueUntilCapacity(p.recentBlock.Hash)
	return nil
}

func (p *ParticipationHandler) ProcessActiveLeavesUpdate(update overseer.ActiveLeavesUpdate) error {
	if update.Activated == nil {
		return nil
	}

	if p.recentBlock == nil {
		p.recentBlock = &block{
			Number: update.Activated.Number,
			Hash:   update.Activated.Hash,
		}

		p.dequeueUntilCapacity(update.Activated.Hash)
	} else {
		if update.Activated.Number > p.recentBlock.Number {
			p.recentBlock.Number = update.Activated.Number
			p.recentBlock.Hash = update.Activated.Hash
		}
	}

	return nil
}

func (p *ParticipationHandler) BumpPriority(ctx overseer.Context, receipts []parachainTypes.CandidateReceipt) error {
	for _, receipt := range receipts {
		blockNumber, err := getBlockNumber(ctx.Sender, receipt)
		if err != nil {
			logger.Errorf(
				"failed to get block number. CommitmentsHash: %s, Error: %s",
				receipt.CommitmentsHash.String(),
				err,
			)
			continue
		}

		comparator, err := NewCandidateComparator(&blockNumber, receipt)
		if err != nil {
			logger.Errorf(
				"failed to create candidate comparator. CommitmentsHash: %s, Error: %s",
				receipt.CommitmentsHash.String(),
				err,
			)
			continue
		}

		if err := p.queue.PrioritiseIfPresent(comparator); err != nil {
			logger.Errorf(
				"failed to prioritise candidate. CommitmentsHash: %s, Error: %s",
				receipt.CommitmentsHash.String(),
				err,
			)
			continue
		}
	}

	return nil
}

func (p *ParticipationHandler) numberOfWorkers() int {
	return int(p.workers.Load())
}

func (p *ParticipationHandler) dequeueUntilCapacity(recentHead common.Hash) {
	for p.numberOfWorkers() < MaxParallelParticipation {
		request := p.queue.Dequeue()
		if request == nil {
			break
		}

		p.forkParticipation(request.request, recentHead)
	}
}

func (p *ParticipationHandler) forkParticipation(request *ParticipationRequest, recentHead common.Hash) {
	_, ok := p.runningParticipation.LoadOrStore(request.candidateHash, nil)
	if ok {
		return
	}

	p.workers.Add(1)
	go func() {
		if err := p.participate(recentHead, *request); err != nil {
			logger.Debugf(
				"failed to participate in dispute. CandidateHash: %s, Error: %s",
				request.candidateHash.String(),
				err,
			)
		}
	}()
}

func (p *ParticipationHandler) participate(blockHash common.Hash, request ParticipationRequest) error {
	// get available data from the sender
	availableDataTx := make(chan overseer.AvailabilityRecoveryResponse, 1)
	if err := p.sender.SendMessage(overseer.AvailabilityRecoveryMessage{
		CandidateReceipt: request.candidateReceipt,
		SessionIndex:     request.session,
		GroupIndex:       nil,
		ResponseChannel:  availableDataTx,
	}); err != nil {
		return fmt.Errorf("send availability recovery message: %w", err)
	}

	recoverDataCtx, recoverDataCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer recoverDataCancel()
	var availableData overseer.AvailabilityRecoveryResponse
	select {
	case <-recoverDataCtx.Done():
		return recoverDataCtx.Err() // Return the context error if timeout exceeded
	case availableData = <-availableDataTx:
		if availableData.Error != nil {
			switch *availableData.Error {
			case overseer.RecoveryErrorInvalid:
				sendResult(p.sender, request, types.ParticipationOutcomeInvalid)
				return fmt.Errorf("invalid available data: %s", availableData.Error.String())
			case overseer.RecoveryErrorUnavailable:
				sendResult(p.sender, request, types.ParticipationOutcomeUnAvailable)
				return fmt.Errorf("unavailable data: %s", availableData.Error.String())
			default:
				return fmt.Errorf("unexpected recovery error: %d", availableData.Error)
			}
		}
	}

	validationCode, err := p.runtime.ParachainHostValidationCodeByHash(
		blockHash,
		request.candidateReceipt.Descriptor.ValidationCodeHash)
	if err != nil || validationCode == nil {
		sendResult(p.sender, request, types.ParticipationOutcomeError)
		return fmt.Errorf("failed to get validation code: %w", err)
	}

	if len(*validationCode) == 0 {
		logger.Errorf(
			"validation code is empty. CandidateHash: %s",
			request.candidateHash.String(),
		)
		sendResult(p.sender, request, types.ParticipationOutcomeError)
		return fmt.Errorf("validation code is empty")
	}

	validateCtx, validateCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer validateCancel()

	// validate the request and send the result
	tx := make(chan overseer.ValidationResult, 1)
	if err := p.sender.SendMessage(overseer.ValidateFromChainState{
		CandidateReceipt:   request.candidateReceipt,
		PoV:                availableData.AvailableData.POV,
		PvfExecTimeoutKind: overseer.PvfExecTimeoutKindApproval,
		ResponseChannel:    tx,
	}); err != nil {
		sendResult(p.sender, request, types.ParticipationOutcomeError)
	}

	select {
	case <-validateCtx.Done():
		return validateCtx.Err()
	case result := <-tx:
		if result.Error != nil {
			// validation failed
			sendResult(p.sender, request, types.ParticipationOutcomeError)
			return fmt.Errorf("validation failed: %s", result.Error)
		}

		if !result.IsValid {
			sendResult(p.sender, request, types.ParticipationOutcomeInvalid)
			return fmt.Errorf("validation failed: %s", result.Error)
		} else {
			sendResult(p.sender, request, types.ParticipationOutcomeValid)
			return nil
		}
	}
}

var _ Participation = &ParticipationHandler{}

func NewParticipation(sender overseer.Sender, runtime parachain.RuntimeInstance) *ParticipationHandler {
	return &ParticipationHandler{
		runningParticipation: sync.Map{},
		queue:                NewQueue(),
		sender:               sender,
		runtime:              runtime,
	}
}

func getBlockNumber(sender overseer.Sender, receipt parachainTypes.CandidateReceipt) (uint32, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tx := make(chan *uint32, 1)
	relayParent, err := receipt.Hash()
	if err != nil {
		return 0, fmt.Errorf("get hash: %w", err)
	}

	if err := sender.SendMessage(overseer.ChainAPIMessage{
		RelayParent:     relayParent,
		ResponseChannel: tx,
	}); err != nil {
		return 0, fmt.Errorf("send message: %w", err)
	}

	select {
	case result := <-tx:
		if result == nil {
			return 0, fmt.Errorf("failed to get block number")
		}
		return *result, nil
	case <-ctx.Done():
		return 0, ctx.Err() // Return the context error if timeout exceeded
	}
}

func sendResult(sender overseer.Sender, request ParticipationRequest, outcome types.ParticipationOutcomeType) {
	participationOutcome, err := types.NewCustomParticipationOutcome(outcome)
	if err != nil {
		logger.Errorf(
			"failed to create participation outcome: %s, error: %s",
			outcome,
			err,
		)
		return
	}

	statement := ParticipationStatement{
		Session:          request.session,
		CandidateHash:    request.candidateHash,
		CandidateReceipt: request.candidateReceipt,
		Outcome:          participationOutcome,
	}
	if err := sender.Feed(statement); err != nil {
		logger.Errorf(
			"failed to send participation result: %s, error: %s",
			statement,
			err,
		)
	}
}
