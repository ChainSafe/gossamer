package dispute

import (
	"bytes"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"sync"

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

var (
	// errorBestEffortQueueFull is returned when the best effort queue is full and the request could not be processed
	errorBestEffortQueueFull = errors.New("best effort queue is full")
	// errorPriorityQueueFull is returned when the priority queue is full and the request could not be processed
	errorPriorityQueueFull = errors.New("priority queue is full")
)

// ParticipationItem implements btree.Item
type ParticipationItem struct {
	comparator CandidateComparator
	request    *ParticipationRequest
}

func participationItemComparator(a, b any) bool {
	pi1, pi2 := a.(*ParticipationItem), b.(*ParticipationItem)

	if pi1.comparator.relayParentBlockNumber == nil && pi2.comparator.relayParentBlockNumber == nil {
		return bytes.Compare(pi1.comparator.candidateHash[:], pi2.comparator.candidateHash[:]) < 0
	}

	if pi1.comparator.relayParentBlockNumber == nil {
		return false
	}

	if pi2.comparator.relayParentBlockNumber == nil {
		return true
	}

	if isEqual := *pi1.comparator.relayParentBlockNumber == *pi2.comparator.relayParentBlockNumber; isEqual {
		return bytes.Compare(pi1.comparator.candidateHash[:], pi2.comparator.candidateHash[:]) < 0
	}

	return *pi1.comparator.relayParentBlockNumber < *pi2.comparator.relayParentBlockNumber
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

type syncedBTree struct {
	BTree scale.BTree
	sync.RWMutex
}

func newSyncedBTree(comparator func(a, b any) bool) *syncedBTree {
	return &syncedBTree{
		BTree: scale.NewBTree[ParticipationItem](comparator),
	}
}

// QueueHandler implements Queue
// It uses two btrees to store the requests. One for best effort and one for priority.
// The queues store participationItem's.
// The btree is ordered by the CandidateComparator of participationItem.
type QueueHandler struct {
	bestEffort *syncedBTree
	priority   *syncedBTree

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

		// remove the item from the best effort queue if it exists
		q.bestEffort.Lock()
		q.bestEffort.BTree.Delete(newParticipationItem(comparator, request))
		q.bestEffort.Unlock()

		q.priority.Lock()
		q.priority.BTree.Set(newParticipationItem(comparator, request))
		q.priority.Unlock()
	} else {
		// if the item is already in priority queue, do nothing
		if item := q.priority.BTree.Get(newParticipationItem(comparator, request)); item != nil {
			return nil
		}

		if q.Len(ParticipationPriorityBestEffort) >= q.bestEffortMaxSize {
			return errorBestEffortQueueFull
		}

		q.bestEffort.Lock()
		q.bestEffort.BTree.Set(newParticipationItem(comparator, request))
		q.bestEffort.Unlock()
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

	q.bestEffort.Lock()
	// We remove the item from the best effort queue and add it to the priority queue if it exists
	if item := q.bestEffort.BTree.Delete(newParticipationItem(comparator, nil)); item != nil {
		q.priority.Lock()
		q.priority.BTree.Set(item)
		q.priority.Unlock()
	}
	q.bestEffort.Unlock()

	return nil
}

func (q *QueueHandler) PopBestEffort() *ParticipationItem {
	if q.bestEffort.BTree.Len() == 0 {
		return nil
	}

	q.bestEffort.Lock()
	defer q.bestEffort.Unlock()
	return q.bestEffort.BTree.PopMin().(*ParticipationItem)
}

func (q *QueueHandler) PopPriority() *ParticipationItem {
	if q.priority.BTree.Len() == 0 {
		return nil
	}

	q.priority.Lock()
	defer q.priority.Unlock()
	return q.priority.BTree.PopMin().(*ParticipationItem)
}

func (q *QueueHandler) Len(queueType ParticipationPriority) int {
	if queueType.IsPriority() {
		q.priority.RLock()
		defer q.priority.RUnlock()
		return q.priority.BTree.Len()
	}

	q.bestEffort.RLock()
	defer q.bestEffort.RUnlock()
	return q.bestEffort.BTree.Len()
}

var _ Queue = (*QueueHandler)(nil)

func NewQueue() *QueueHandler {
	return &QueueHandler{
		bestEffort:        newSyncedBTree(participationItemComparator),
		priority:          newSyncedBTree(participationItemComparator),
		bestEffortMaxSize: bestEffortQueueSize,
		priorityMaxSize:   priorityQueueSize,
	}
}
