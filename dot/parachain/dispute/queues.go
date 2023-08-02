package dispute

import (
	"bytes"
	"sync"

	parachain "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"

	"github.com/google/btree"
	"github.com/pkg/errors"
)

// This file contains the types and methods for the queues used by the dispute coordinator
// The implementation is based on parity's participation queue
// Read more here: https://paritytech.github.io/polkadot/book/node/disputes/dispute-coordinator.html
// https://github.com/paritytech/polkadot/blob/master/node/core/dispute-coordinator/src/participation/queues/mod.rs
// It uses btree. Find it here: https://github.com/google/btree

// TODO: Parity's implementation captures metrics for the queue. We should do the same.
// However, I will not be implementing it right away. It will be picked up as a single task for the
// entire dispute module https://github.com/ChainSafe/gossamer/issues/3313.

// CandidateComparator comparator for ordering of disputes for candidate.
type CandidateComparator struct {
	relayParentBlockNumber *uint32
	candidateHash          common.Hash
}

// ParticipationRequest a dispute participation request
type ParticipationRequest struct {
	candidateHash    common.Hash
	candidateReceipt parachain.CandidateReceipt
	session          parachain.SessionIndex
	//TODO: requestTimer for metrics
}

// ParticipationItem implements btree.Item
type ParticipationItem struct {
	comparator CandidateComparator
	request    *ParticipationRequest
}

// Less returns true if the current item is less than the other item
// it uses the CandidateComparator to determine the order
func (q ParticipationItem) Less(than btree.Item) bool {
	other := than.(*ParticipationItem)

	if q.comparator.relayParentBlockNumber == nil && other.comparator.relayParentBlockNumber == nil {
		return bytes.Compare(q.comparator.candidateHash[:], other.comparator.candidateHash[:]) < 0
	}

	if other.comparator.relayParentBlockNumber == nil {
		return false
	}

	if q.comparator.relayParentBlockNumber == nil {
		return true
	}

	if isEqual := *q.comparator.relayParentBlockNumber == *other.comparator.relayParentBlockNumber; isEqual {
		return bytes.Compare(q.comparator.candidateHash[:], other.comparator.candidateHash[:]) < 0
	}

	return *q.comparator.relayParentBlockNumber < *other.comparator.relayParentBlockNumber
}

func newParticipationItem(comparator CandidateComparator, request *ParticipationRequest) *ParticipationItem {
	return &ParticipationItem{
		comparator: comparator,
		request:    request,
	}
}

// ParticipationPriority the priority of a participation request
type ParticipationPriority int

const (
	// ParticipationPriorityBestEffort is the lowest priority
	ParticipationPriorityBestEffort ParticipationPriority = iota
	// ParticipationPriorityHigh is the highest priority
	ParticipationPriorityHigh
)

// IsPriority returns true if the priority is high
func (p ParticipationPriority) IsPriority() bool {
	return p == ParticipationPriorityHigh
}

var (
	// errorBestEffortQueueFull is returned when the best effort queue is full and the request could not be processed
	errorBestEffortQueueFull = errors.New("best effort queue is full")
	// errorPriorityQueueFull is returned when the priority queue is full and the request could not be processed
	errorPriorityQueueFull = errors.New("priority queue is full")
)

// Queue the dispute participation queue
type Queue interface {
	// Queue adds a new participation request to the queue
	Queue(comparator CandidateComparator, request *ParticipationRequest, priority ParticipationPriority) error

	// Dequeue gets the next best request for dispute participation if any.
	Dequeue() *ParticipationItem

	// PrioritiseIfPresent moves a participation request from the best effort queue to the priority queue
	PrioritiseIfPresent(comparator CandidateComparator) error

	// PopBestEffort removes the next participation request from the best effort queue
	PopBestEffort() *ParticipationItem

	// PopPriority removes the next participation request from the priority queue
	PopPriority() *ParticipationItem

	// Len returns the number of items in the specified queue
	Len(queueType ParticipationPriority) int
}

// QueueHandler implements Queue
// It uses two btree's to store the requests. One for best effort and one for priority.
// The queues store participationItem's.
// The btree is ordered by the CandidateComparator of participationItem.
type QueueHandler struct {
	bestEffort *btree.BTree
	priority   *btree.BTree

	bestEffortLock sync.RWMutex
	priorityLock   sync.RWMutex

	bestEffortMaxSize int
	priorityMaxSize   int

	//TODO: add metrics
}

const (
	// bestEffortQueueSize is the maximum size of the best effort queue
	bestEffortQueueSize = 100
	// priorityQueueSize is the maximum size of the priority queue
	priorityQueueSize = 20000
)

func (q *QueueHandler) Queue(
	comparator CandidateComparator,
	request *ParticipationRequest,
	priority ParticipationPriority,
) error {
	if priority.IsPriority() {
		if q.Len(ParticipationPriorityHigh) >= q.priorityMaxSize {
			return errorPriorityQueueFull
		}

		q.priorityLock.Lock()
		q.priority.ReplaceOrInsert(newParticipationItem(comparator, request))
		q.priorityLock.Unlock()
	} else {
		if q.Len(ParticipationPriorityBestEffort) >= q.bestEffortMaxSize {
			return errorBestEffortQueueFull
		}

		q.bestEffortLock.Lock()
		q.bestEffort.ReplaceOrInsert(newParticipationItem(comparator, request))
		q.bestEffortLock.Unlock()
	}

	return nil
}

func (q *QueueHandler) Dequeue() *ParticipationItem {
	if item := q.PopPriority(); item != nil {
		return item
	}

	return q.PopBestEffort()
}

func (q *QueueHandler) PrioritiseIfPresent(comparator CandidateComparator) error {
	if q.Len(ParticipationPriorityHigh) >= priorityQueueSize {
		return errorPriorityQueueFull
	}

	q.bestEffortLock.Lock()
	// We remove the item from the best effort queue and add it to the priority queue if it exists
	if item := q.bestEffort.Delete(newParticipationItem(comparator, nil)); item != nil {
		q.priorityLock.Lock()
		q.priority.ReplaceOrInsert(item)
		q.priorityLock.Unlock()
	}
	q.bestEffortLock.Unlock()

	return nil
}

func (q *QueueHandler) PopBestEffort() *ParticipationItem {
	q.bestEffortLock.Lock()
	defer q.bestEffortLock.Unlock()
	if item := q.bestEffort.DeleteMin(); item != nil {
		return item.(*ParticipationItem)
	}

	return nil
}

func (q *QueueHandler) PopPriority() *ParticipationItem {
	q.priorityLock.Lock()
	defer q.priorityLock.Unlock()
	if item := q.priority.DeleteMin(); item != nil {
		return item.(*ParticipationItem)
	}

	return nil
}

func (q *QueueHandler) Len(queueType ParticipationPriority) int {
	if queueType.IsPriority() {
		q.priorityLock.RLock()
		defer q.priorityLock.RUnlock()
		return q.priority.Len()
	}

	q.bestEffortLock.RLock()
	defer q.bestEffortLock.RUnlock()
	return q.bestEffort.Len()
}

var _ Queue = (*QueueHandler)(nil)

func NewQueue() *QueueHandler {
	return &QueueHandler{
		bestEffort:        btree.New(bestEffortQueueSize / 2),
		priority:          btree.New(priorityQueueSize / 2),
		bestEffortMaxSize: bestEffortQueueSize,
		priorityMaxSize:   priorityQueueSize,
	}
}
