package dispute

import (
	"encoding/binary"
	"sync"
	"testing"

	"github.com/ChainSafe/gossamer/dot/parachain/dispute/types"

	parachain "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/require"
)

func newTestQueue(size int) *QueueHandler {
	return &QueueHandler{
		bestEffort:        newSyncedBTree(participationItemComparator),
		priority:          newSyncedBTree(participationItemComparator),
		bestEffortMaxSize: size,
		priorityMaxSize:   size,
	}
}

func newComparator(blockNumber, order uint32) CandidateComparator {
	candidateHash := make([]byte, 4)
	binary.LittleEndian.PutUint32(candidateHash, order)

	return CandidateComparator{
		relayParentBlockNumber: &blockNumber,
		candidateHash:          common.NewHash(candidateHash),
	}
}

func dummyParticipationData(priority ParticipationPriority) ParticipationData {
	return ParticipationData{
		types.ParticipationRequest{
			CandidateHash:    [32]byte{},
			CandidateReceipt: parachain.CandidateReceipt{},
			Session:          1,
		},
		priority,
	}
}

type test struct {
	name string
	// operation one of "queue", "dequeue", "prioritise", "pop_priority", "pop_best_effort",
	// "len_priority", "len_best_effort"
	operation     string
	comparator    CandidateComparator
	participation ParticipationData
	expected      any
	mustError     bool
}

func runTests(t *testing.T, tests []test, queue Queue) {
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			switch tt.operation {
			case "queue":
				err := queue.Queue(tt.comparator, tt.participation)
				if tt.mustError {
					require.Error(t, err)
					require.Equal(t, err, tt.expected)
					return
				}
				require.NoError(t, err)
			case "dequeue":
				item := queue.Dequeue()
				require.Equal(t, tt.expected, item)
			case "prioritise":
				err := queue.PrioritiseIfPresent(tt.comparator)
				if tt.mustError {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
			case "pop_priority":
				item := queue.PopPriority()
				require.Equal(t, tt.expected, item)
			case "pop_best_effort":
				item := queue.PopBestEffort()
				require.Equal(t, tt.expected, item)
			case "len_priority":
				require.Equal(t, tt.expected, queue.Len(ParticipationPriorityHigh))
			case "len_best_effort":
				require.Equal(t, tt.expected, queue.Len(ParticipationPriorityBestEffort))
			default:
				t.Fatalf("unknown operation %s", tt.operation)
			}
		})
	}
}

// TestQueue_CompareRelayParentBlock tests the following:
// - queueing 3 requests into priority queue with different relay parent block numbers
// - queueing 1 request with the best effort priority
// - dequeue must return the request with the lowest relay parent block number from the priority queue
func TestQueue_CompareRelayParentBlock(t *testing.T) {
	expectedParticipation := dummyParticipationData(ParticipationPriorityHigh)
	tests := []test{
		{
			name:          "block 1",
			operation:     "queue",
			comparator:    newComparator(1, 1),
			participation: dummyParticipationData(ParticipationPriorityHigh),
		},
		{
			name:          "block 2",
			operation:     "queue",
			comparator:    newComparator(2, 1),
			participation: dummyParticipationData(ParticipationPriorityHigh),
		},
		{
			name:          "block 3",
			operation:     "queue",
			comparator:    newComparator(3, 1),
			participation: dummyParticipationData(ParticipationPriorityHigh),
		},
		{
			name:          "block 1 - best effort",
			operation:     "queue",
			comparator:    newComparator(1, 1),
			participation: dummyParticipationData(ParticipationPriorityHigh),
		},
		{
			name:      "dequeue",
			operation: "dequeue",
			expected: &ParticipationItem{
				comparator: newComparator(1, 1),
				request:    &expectedParticipation.request,
			},
		},
	}

	runTests(t, tests, newTestQueue(10))
}

// TestQueue_CompareCandidateHash tests the following:
// - queueing 3 requests with same relay parent block number into priority queue with different candidate hashes
// - queueing 1 request with the best effort priority
// - dequeue must return the request with the lowest candidate hash from the priority queue
func TestQueue_CompareCandidateHash(t *testing.T) {
	expectedParticipation := dummyParticipationData(ParticipationPriorityHigh)
	tests := []test{
		{
			name:          "block 1",
			operation:     "queue",
			comparator:    newComparator(1, 1),
			participation: dummyParticipationData(ParticipationPriorityHigh),
		},
		{
			name:          "block 1 - 2",
			operation:     "queue",
			comparator:    newComparator(1, 2),
			participation: dummyParticipationData(ParticipationPriorityHigh),
		},
		{
			name:      "dequeue",
			operation: "dequeue",
			expected: &ParticipationItem{
				comparator: newComparator(1, 1),
				request:    &expectedParticipation.request,
			},
		},
	}

	runTests(t, tests, newTestQueue(10))
}

// TestQueue_EndToEnd tests the following:
// - queueing 3 requests into priority queue
// - queueing 3 requests into best effort queue
// - popping the best effort queue
// - popping the priority queue
// - prioritising a request in the best effort queue
func TestQueue_EndToEnd(t *testing.T) {
	expectedParticipation := dummyParticipationData(ParticipationPriorityHigh)
	tests := []test{
		{
			name:          "block 1, order 1",
			operation:     "queue",
			comparator:    newComparator(1, 1),
			participation: dummyParticipationData(ParticipationPriorityHigh),
		},
		{
			name:          "block 2, order 1",
			operation:     "queue",
			comparator:    newComparator(2, 1),
			participation: dummyParticipationData(ParticipationPriorityHigh),
		},
		{
			name:          "block 3, order 1",
			operation:     "queue",
			comparator:    newComparator(3, 1),
			participation: dummyParticipationData(ParticipationPriorityHigh),
		},
		{
			name:          "block 1, order 2 - best effort",
			operation:     "queue",
			comparator:    newComparator(1, 2),
			participation: dummyParticipationData(ParticipationPriorityBestEffort),
		},
		{
			name:          "block 2, order 2 - best effort",
			operation:     "queue",
			comparator:    newComparator(2, 2),
			participation: dummyParticipationData(ParticipationPriorityBestEffort),
		},
		{
			name:          "block 3, order 2 - best effort",
			operation:     "queue",
			comparator:    newComparator(3, 2),
			participation: dummyParticipationData(ParticipationPriorityBestEffort),
		},
		{
			name:      "priority length",
			operation: "len_priority",
			expected:  3,
		},
		{
			name:      "best effort length",
			operation: "len_best_effort",
			expected:  3,
		},
		{
			name:      "pop priority",
			operation: "pop_priority",
			expected: &ParticipationItem{
				comparator: newComparator(1, 1),
				request:    &expectedParticipation.request,
			},
		},
		{
			name:      "pop best effort",
			operation: "pop_best_effort",
			expected: &ParticipationItem{
				comparator: newComparator(1, 2),
				request:    &expectedParticipation.request,
			},
		},
		{
			name:      "priority length",
			operation: "len_priority",
			expected:  2,
		},
		{
			name:      "best effort length",
			operation: "len_best_effort",
			expected:  2,
		},
		{
			name:       "prioritise best effort",
			operation:  "prioritise",
			comparator: newComparator(2, 2),
		},
		{
			name:      "priority length",
			operation: "len_priority",
			expected:  3,
		},
		{
			name:      "dequeue",
			operation: "dequeue",
			expected: &ParticipationItem{
				comparator: newComparator(2, 1),
				request:    &expectedParticipation.request,
			},
		},
		{
			name:      "dequeue",
			operation: "dequeue",
			expected: &ParticipationItem{
				comparator: newComparator(2, 2),
				request:    &expectedParticipation.request,
			},
		},
	}

	runTests(t, tests, newTestQueue(10))
}

// TestQueue_OverflowPriority tests the following:
// - queueing 5 requests into priority queue with the max length of 4
// - the 5th request must return an error
func TestQueue_OverflowPriority(t *testing.T) {
	tests := []test{
		{
			name:          "block 1",
			operation:     "queue",
			comparator:    newComparator(1, 1),
			participation: dummyParticipationData(ParticipationPriorityHigh),
		},
		{
			name:          "block 1 - 2",
			operation:     "queue",
			comparator:    newComparator(1, 2),
			participation: dummyParticipationData(ParticipationPriorityHigh),
		},
		{
			name:          "block 1 - 3",
			operation:     "queue",
			comparator:    newComparator(1, 3),
			participation: dummyParticipationData(ParticipationPriorityHigh),
		},
		{
			name:          "block 1 - 4",
			operation:     "queue",
			comparator:    newComparator(1, 4),
			participation: dummyParticipationData(ParticipationPriorityHigh),
		},
		{
			name:          "block 1 - 5",
			operation:     "queue",
			comparator:    newComparator(1, 5),
			participation: dummyParticipationData(ParticipationPriorityHigh),
			mustError:     true,
			expected:      errorPriorityQueueFull,
		},
	}

	runTests(t, tests, newTestQueue(4))
}

// TestQueue_OverflowBestEffort tests the following:
// - queueing 5 requests into the best effort queue with the max length of 4
// - the 5th request must return an error
func TestQueue_OverflowBestEffort(t *testing.T) {
	tests := []test{
		{
			name:          "block 1",
			operation:     "queue",
			comparator:    newComparator(1, 1),
			participation: dummyParticipationData(ParticipationPriorityBestEffort),
		},
		{
			name:          "block 1 - 2",
			operation:     "queue",
			comparator:    newComparator(1, 2),
			participation: dummyParticipationData(ParticipationPriorityBestEffort),
		},
		{
			name:          "block 1 - 3",
			operation:     "queue",
			comparator:    newComparator(1, 3),
			participation: dummyParticipationData(ParticipationPriorityBestEffort),
		},
		{
			name:          "block 1 - 4",
			operation:     "queue",
			comparator:    newComparator(1, 4),
			participation: dummyParticipationData(ParticipationPriorityBestEffort),
		},
		{
			name:          "block 1 - 5",
			operation:     "queue",
			comparator:    newComparator(1, 5),
			participation: dummyParticipationData(ParticipationPriorityBestEffort),
			mustError:     true,
			expected:      errorBestEffortQueueFull,
		},
	}

	runTests(t, tests, newTestQueue(4))
}

// TestQueueConcurrency_Dequeue tests the following:
// - concurrent queueing of 1000 requests
// - concurrent dequeue of 1000 requests
func TestQueueConcurrency_Dequeue(t *testing.T) {
	q := NewQueue()
	numberOfOperations := 1000

	wg := sync.WaitGroup{}
	wg.Add(numberOfOperations)
	for i := 0; i < numberOfOperations; i++ {
		block := uint32(i)
		go func() {
			defer wg.Done()

			err := q.Queue(newComparator(block, 1), dummyParticipationData(ParticipationPriorityHigh))
			require.NoError(t, err)
		}()
	}
	wg.Wait()

	require.Equal(t, numberOfOperations, q.Len(ParticipationPriorityHigh))

	wg.Add(numberOfOperations)
	for i := 0; i < numberOfOperations; i++ {
		go func() {
			defer wg.Done()
			q.Dequeue()
		}()
	}
	wg.Wait()

	require.Equal(t, 0, q.Len(ParticipationPriorityHigh))
}

// TestQueueConcurrency_Prioritise tests the following:
// - concurrent queueing of 1000 requests into the best effort queue
// - concurrent prioritise of 1000 requests
func TestQueueConcurrency_Prioritise(t *testing.T) {
	q := NewQueue()
	numberOfOperations := 100

	wg := sync.WaitGroup{}
	wg.Add(numberOfOperations)
	for i := 0; i < numberOfOperations; i++ {
		block := uint32(i)
		go func() {
			defer wg.Done()

			err := q.Queue(newComparator(block, 1), dummyParticipationData(ParticipationPriorityBestEffort))
			require.NoError(t, err)
		}()
	}
	wg.Wait()

	require.Equal(t, numberOfOperations, q.Len(ParticipationPriorityBestEffort))

	wg.Add(numberOfOperations)
	for i := 0; i < numberOfOperations; i++ {
		block := uint32(i)
		go func() {
			defer wg.Done()
			err := q.PrioritiseIfPresent(newComparator(block, 1))
			require.NoError(t, err)
		}()
	}
	wg.Wait()

	require.Equal(t, numberOfOperations, q.Len(ParticipationPriorityHigh))
	require.Equal(t, 0, q.Len(ParticipationPriorityBestEffort))
}

// TestQueueConcurrency_PopBestEffort tests the following:
// - concurrent queueing of 1000 requests into the best effort queue
// - concurrent pop of 1000 requests from the best effort queue
func TestQueueConcurrency_PopBestEffort(t *testing.T) {
	q := NewQueue()
	numberOfOperations := 100

	wg := sync.WaitGroup{}
	wg.Add(numberOfOperations)
	for i := 0; i < numberOfOperations; i++ {
		block := uint32(i)
		go func() {
			defer wg.Done()

			err := q.Queue(newComparator(block, 1), dummyParticipationData(ParticipationPriorityBestEffort))
			require.NoError(t, err)
		}()
	}
	wg.Wait()

	require.Equal(t, numberOfOperations, q.Len(ParticipationPriorityBestEffort))

	wg.Add(numberOfOperations)
	for i := 0; i < numberOfOperations; i++ {
		go func() {
			defer wg.Done()
			item := q.PopBestEffort()
			require.NotNil(t, item)
		}()
	}
	wg.Wait()

	require.Equal(t, 0, q.Len(ParticipationPriorityBestEffort))
}

// TestQueueConcurrency_PopPriority tests the following:
// - concurrent queueing of 1000 requests into the high priority queue
// - concurrent pop of 1000 requests from the high priority queue
func TestQueueConcurrency_PopPriority(t *testing.T) {
	q := NewQueue()
	numberOfOperations := 100

	wg := sync.WaitGroup{}
	wg.Add(numberOfOperations)
	for i := 0; i < numberOfOperations; i++ {
		block := uint32(i)
		go func() {
			defer wg.Done()

			err := q.Queue(newComparator(block, 1), dummyParticipationData(ParticipationPriorityHigh))
			require.NoError(t, err)
		}()
	}
	wg.Wait()

	require.Equal(t, numberOfOperations, q.Len(ParticipationPriorityHigh))

	wg.Add(numberOfOperations)
	for i := 0; i < numberOfOperations; i++ {
		go func() {
			defer wg.Done()
			item := q.PopPriority()
			require.NotNil(t, item)
		}()
	}
	wg.Wait()

	require.Equal(t, 0, q.Len(ParticipationPriorityHigh))
}

func BenchmarkQueue_Queue(b *testing.B) {
	q := newTestQueue(priorityQueueSize)

	for i := 0; i < priorityQueueSize; i++ {
		err := q.Queue(newComparator(uint32(i), 1), dummyParticipationData(ParticipationPriorityHigh))
		require.NoError(b, err)
	}
}

func BenchmarkQueue_Dequeue(b *testing.B) {
	q := NewQueue()

	for i := 0; i < bestEffortQueueSize; i++ {
		err := q.Queue(newComparator(uint32(i), 1), dummyParticipationData(ParticipationPriorityBestEffort))
		require.NoError(b, err)
	}
	b.ResetTimer()

	for i := 0; i < bestEffortQueueSize; i++ {
		item := q.Dequeue()
		require.NotNil(b, item)
	}
}

func BenchmarkQueue_PopPriority(b *testing.B) {
	q := NewQueue()

	for i := 0; i < priorityQueueSize; i++ {
		err := q.Queue(newComparator(uint32(i), 1), dummyParticipationData(ParticipationPriorityHigh))
		require.NoError(b, err)
	}
	b.ResetTimer()

	for i := 0; i < priorityQueueSize; i++ {
		item := q.PopPriority()
		require.NotNil(b, item)
	}
}

func BenchmarkQueue_PopBestEffort(b *testing.B) {
	q := NewQueue()

	for i := 0; i < bestEffortQueueSize; i++ {
		err := q.Queue(newComparator(uint32(i), 1), dummyParticipationData(ParticipationPriorityBestEffort))
		require.NoError(b, err)
	}
	b.ResetTimer()

	for i := 0; i < bestEffortQueueSize; i++ {
		item := q.PopBestEffort()
		require.NotNil(b, item)
	}
}

func BenchmarkQueue_PrioritiseIfPresent(b *testing.B) {
	q := NewQueue()

	for i := 0; i < bestEffortQueueSize; i++ {
		err := q.Queue(newComparator(uint32(i), 1), dummyParticipationData(ParticipationPriorityBestEffort))
		require.NoError(b, err)
	}
	b.ResetTimer()

	for i := 0; i < priorityQueueSize; i++ {
		err := q.PrioritiseIfPresent(newComparator(uint32(i), 1))
		require.NoError(b, err)
	}
}
