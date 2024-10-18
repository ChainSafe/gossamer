// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
)

type mockTask struct {
	id           TaskID
	err          error
	execCount    uint
	succeedAfter uint
}

func (m *mockTask) ID() TaskID {
	return m.id
}

func (m *mockTask) Do(p peer.ID) (Result, error) {
	time.Sleep(time.Millisecond * 100) // simulate network roundtrip
	defer func() {
		m.execCount++
	}()

	res := Result(fmt.Sprintf("%s - %s great success!", m.id, p))
	if m.err != nil {
		if m.succeedAfter > 0 && m.execCount >= m.succeedAfter {
			return res, nil
		}
		return nil, m.err
	}
	return res, nil
}

func (m *mockTask) String() string {
	return fmt.Sprintf("mockTask %s", m.id)
}

func makeTasksAndPeers(num, idOffset int) ([]Task, []peer.ID) {
	tasks := make([]Task, num)
	peers := make([]peer.ID, num)

	for i := 0; i < num; i++ {
		tasks[i] = &mockTask{id: TaskID(fmt.Sprintf("t-%d", i+idOffset))}
		peers[i] = peer.ID(fmt.Sprintf("p-%d", i+idOffset))
	}
	return tasks, peers
}

func waitForCompletion(wp WorkerPool, numTasks int) {
	resultsReceived := 0

	for {
		<-wp.Results()
		resultsReceived++

		if resultsReceived == numTasks {
			break
		}
	}
}

func TestWorkerPoolHappyPath(t *testing.T) {
	numTasks := 10

	var setup = func() (WorkerPool, []Task) {
		tasks, peers := makeTasksAndPeers(numTasks, 0)
		wp := NewWorkerPool(WorkerPoolConfig{})

		for _, who := range peers {
			err := wp.AddPeer(who)
			assert.NoError(t, err)
		}

		return wp, tasks
	}

	t.Run("receive_results_on_channel", func(t *testing.T) {
		wp, tasks := setup()
		results := make([]TaskResult, 0, numTasks)
		_, err := wp.SubmitBatch(tasks)

		assert.NoError(t, err)

		for {
			result := <-wp.Results()
			assert.True(t, result.Completed)
			assert.False(t, result.Failed())
			assert.Equal(t, uint(0), result.Retries)

			results = append(results, result)
			if len(results) == numTasks {
				break
			}
		}
	})

	t.Run("check_batch_status_on_completion", func(t *testing.T) {
		wp, tasks := setup()
		batchID, err := wp.SubmitBatch(tasks)
		assert.NoError(t, err)

		waitForCompletion(wp, numTasks)
		status, ok := wp.GetBatch(batchID)

		assert.True(t, ok)
		assert.True(t, status.Completed(numTasks))
		assert.Equal(t, numTasks, len(status.Success))
		assert.Equal(t, 0, len(status.Failed))
	})
}

func TestWorkerPoolPeerHandling(t *testing.T) {
	numTasks := 3

	t.Run("accepts_batch_without_any_peers", func(t *testing.T) {
		tasks, _ := makeTasksAndPeers(numTasks, 0)
		wp := NewWorkerPool(WorkerPoolConfig{})

		_, err := wp.SubmitBatch(tasks)
		assert.NoError(t, err)

		wp.Shutdown()
	})

	t.Run("completes_batch_with_fewer_peers_than_tasks", func(t *testing.T) {
		tasks, peers := makeTasksAndPeers(numTasks, 0)
		wp := NewWorkerPool(WorkerPoolConfig{})
		assert.NoError(t, wp.AddPeer(peers[0]))
		assert.NoError(t, wp.AddPeer(peers[1]))

		bID, err := wp.SubmitBatch(tasks)
		assert.NoError(t, err)

		waitForCompletion(wp, numTasks)
		status, ok := wp.GetBatch(bID)
		assert.True(t, ok)
		assert.True(t, status.Completed(numTasks))
		assert.Equal(t, numTasks, len(status.Success))
		assert.Equal(t, 0, len(status.Failed))
	})

	t.Run("refuses_to_re_add_ignored_peer", func(t *testing.T) {
		_, peers := makeTasksAndPeers(numTasks, 0)
		wp := NewWorkerPool(WorkerPoolConfig{})

		for _, who := range peers {
			err := wp.AddPeer(who)
			assert.NoError(t, err)
		}
		assert.Equal(t, len(peers), wp.NumPeers())

		badPeer := peers[2]
		wp.IgnorePeer(badPeer)
		assert.Equal(t, len(peers)-1, wp.NumPeers())

		err := wp.AddPeer(badPeer)
		assert.ErrorIs(t, err, ErrPeerIgnored)
		assert.Equal(t, len(peers)-1, wp.NumPeers())
	})
}

func TestWorkerPoolTaskFailures(t *testing.T) {
	numTasks := 3
	taskErr := errors.New("kaput")

	setup := func(maxRetries uint) (failOnce *mockTask, failTwice *mockTask, batchID BatchID, wp WorkerPool) {
		tasks, peers := makeTasksAndPeers(numTasks, 0)

		failOnce = tasks[1].(*mockTask)
		failOnce.err = taskErr
		failOnce.succeedAfter = 1

		failTwice = tasks[2].(*mockTask)
		failTwice.err = taskErr
		failTwice.succeedAfter = 2

		wp = NewWorkerPool(WorkerPoolConfig{MaxRetries: maxRetries})
		for _, who := range peers {
			err := wp.AddPeer(who)
			assert.NoError(t, err)
		}

		var err error
		batchID, err = wp.SubmitBatch(tasks)
		assert.NoError(t, err)
		return
	}

	t.Run("retries_failed_tasks", func(t *testing.T) {
		failOnce, failTwice, batchID, wp := setup(10)
		waitForCompletion(wp, numTasks)

		status, ok := wp.GetBatch(batchID)
		assert.True(t, ok)
		assert.True(t, status.Completed(numTasks))
		assert.Equal(t, numTasks, len(status.Success))
		assert.Equal(t, 0, len(status.Failed))

		assert.Nil(t, status.Failed[failOnce.ID()].Error)
		assert.Equal(t, uint(1), status.Success[failOnce.ID()].Retries)

		assert.Nil(t, status.Failed[failTwice.ID()].Error)
		assert.Equal(t, uint(2), status.Success[failTwice.ID()].Retries)
	})

	t.Run("honours_max_retries", func(t *testing.T) {
		failOnce, failTwice, batchID, wp := setup(1)
		waitForCompletion(wp, numTasks)

		status, ok := wp.GetBatch(batchID)
		assert.True(t, ok)
		assert.True(t, status.Completed(numTasks))
		assert.Equal(t, numTasks-1, len(status.Success))
		assert.Equal(t, 1, len(status.Failed))

		assert.Nil(t, status.Failed[failOnce.ID()].Error)
		assert.Equal(t, uint(1), status.Success[failOnce.ID()].Retries)

		assert.ErrorIs(t, taskErr, status.Failed[failTwice.ID()].Error)
		assert.Equal(t, uint(1), status.Failed[failTwice.ID()].Retries)
	})
}

func TestWorkerPoolMultipleBatches(t *testing.T) {
	b1NumTasks := 10
	b2NumTasks := 12

	t.Run("completes_all_batches", func(t *testing.T) {
		b1Tasks, b1Peers := makeTasksAndPeers(b1NumTasks, 0)
		b2Tasks, b2Peers := makeTasksAndPeers(b2NumTasks, b1NumTasks)
		peers := append(b1Peers, b2Peers...)

		wp := NewWorkerPool(WorkerPoolConfig{})
		for _, who := range peers {
			err := wp.AddPeer(who)
			assert.NoError(t, err)
		}

		b1ID, err := wp.SubmitBatch(b1Tasks)
		assert.NoError(t, err)

		b2ID, err := wp.SubmitBatch(b2Tasks)
		assert.NoError(t, err)

		waitForCompletion(wp, b1NumTasks+b2NumTasks)

		b1Status, ok := wp.GetBatch(b1ID)
		assert.True(t, ok)
		assert.True(t, b1Status.Completed(b1NumTasks))
		assert.Equal(t, b1NumTasks, len(b1Status.Success))
		assert.Equal(t, 0, len(b1Status.Failed))

		b2Status, ok := wp.GetBatch(b2ID)
		assert.True(t, ok)
		assert.True(t, b2Status.Completed(b2NumTasks))
		assert.Equal(t, b2NumTasks, len(b2Status.Success))
		assert.Equal(t, 0, len(b2Status.Failed))
	})
}
