// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type lifecycleItem = GlobalInItem[string, uint32, Signature, ID]

// newLifecycleVoter builds a voter over its own inbound channel, for tests about
// starting and stopping rather than about voting.
func newLifecycleVoter(t *testing.T, network *Network, globalIn chan lifecycleItem,
) *Voter[string, uint32, Signature, ID] {
	t.Helper()
	var localID ID = 5
	voters := NewVoterSet([]IDWeight[ID]{{localID, 100}})

	env := newEnvironment(network, localID)
	var lastFinalized HashNumber[string, uint32]
	env.WithChain(func(chain *dummyChain) {
		chain.PushBlocks(GenesisHash, []string{"A", "B", "C"})
		lastFinalized.Hash, lastFinalized.Number = chain.LastFinalized()
	})

	return NewVoter[string, uint32, Signature, ID](
		&env, *voters, globalIn,
		func(CommunicationOut[string, uint32, Signature, ID]) error { return nil },
		0, nil, lastFinalized, lastFinalized,
	)
}

// forwardersInState counts wakerChan.start goroutines blocked in the given
// runtime state: "chan send" is one parked on the unbuffered `out` holding an
// item, "chan receive" one waiting on `in`.
func forwardersInState(state string) int {
	buf := make([]byte, 1<<22)
	n := runtime.Stack(buf, true)
	count := 0
	for _, blk := range strings.Split(string(buf[:n]), "\ngoroutine ") {
		nl := strings.Index(blk, "\n")
		if nl < 0 {
			continue
		}
		if strings.Contains(blk[:nl], state) &&
			strings.Contains(blk[nl:], "wakerChan") && strings.Contains(blk[nl:], ".start(") {
			count++
		}
	}
	return count
}

// Closing globalIn is the shutdown signal: Start returns of its own accord, and
// reports nil because an orderly shutdown is not a failure.
func TestVoter_CloseGlobalInShutsDown(t *testing.T) {
	network := NewNetwork()
	defer network.Stop()

	globalIn := make(chan lifecycleItem, 10)
	v := newLifecycleVoter(t, network, globalIn)

	errCh := make(chan error, 1)
	go func() { errCh <- v.Start() }()
	time.Sleep(50 * time.Millisecond)

	close(globalIn)
	select {
	case err := <-errCh:
		assert.NoError(t, err, "an orderly shutdown is not an error")
	case <-time.After(5 * time.Second):
		t.Fatal("Start did not return after globalIn was closed")
	}
	assert.NoError(t, v.Wait())
}

// The retired voter's forwarder must be gone once Wait returns. There is no
// stopChan any more, so the poll loop cannot exit before it has drained what the
// forwarder is holding: it only leaves by observing the close.
func TestVoter_RotationLeavesNoForwarder(t *testing.T) {
	network := NewNetwork()
	defer network.Stop()

	baseSend := forwardersInState("chan send")
	baseRecv := forwardersInState("chan receive")

	const rotations = 10
	for i := 0; i < rotations; i++ {
		globalIn := make(chan lifecycleItem, 100)
		v := newLifecycleVoter(t, network, globalIn)

		done := make(chan struct{})
		go func() { defer close(done); _ = v.Start() }()
		time.Sleep(30 * time.Millisecond)

		// Inbound traffic in flight when the set rotates.
		for j := 0; j < 100; j++ {
			select {
			case globalIn <- lifecycleItem{}:
			default:
			}
		}

		close(globalIn)
		<-done
		require.NoError(t, v.Wait())
	}

	time.Sleep(300 * time.Millisecond)
	send := forwardersInState("chan send") - baseSend
	recv := forwardersInState("chan receive") - baseRecv
	t.Logf("after %d rotations: send-parked=%+d recv-parked=%+d", rotations, send, recv)
	assert.Zero(t, send, "a retired voter's globalIn forwarder is still holding an item")
}

// Rebuilding the voter across an authority-set rotation: close, wait, build the
// next one, start it. Start is launched from a goroutine, so a rotation can reach
// the next Wait before that Start has begun.
func TestVoter_RebuildAcrossRotations(t *testing.T) {
	network := NewNetwork()
	defer network.Stop()

	var starts sync.WaitGroup
	globalIn := make(chan lifecycleItem, 100)
	first := newLifecycleVoter(t, network, globalIn)
	voter := first
	starts.Add(1)
	go func() { defer starts.Done(); _ = first.Start() }()

	const rotations = 100
	for i := 0; i < rotations; i++ {
		close(globalIn)
		require.NoError(t, voter.Wait())

		globalIn = make(chan lifecycleItem, 100)
		v := newLifecycleVoter(t, network, globalIn)
		voter = v
		starts.Add(1)
		go func() { defer starts.Done(); _ = v.Start() }()
	}
	close(globalIn)
	require.NoError(t, voter.Wait())

	done := make(chan struct{})
	go func() { starts.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("a Start goroutine never returned")
	}
}

// Wait does not signal shutdown. Calling it without closing globalIn must report
// a timeout rather than block forever or claim success.
func TestVoter_WaitWithoutCloseTimesOut(t *testing.T) {
	network := NewNetwork()
	defer network.Stop()

	globalIn := make(chan lifecycleItem, 10)
	v := newLifecycleVoter(t, network, globalIn)
	v.waitTimeout = 200 * time.Millisecond

	done := make(chan struct{})
	go func() { defer close(done); _ = v.Start() }()
	time.Sleep(50 * time.Millisecond)

	assert.ErrorContains(t, v.Wait(), "timeout waiting for the voter to stop")

	close(globalIn)
	<-done
}

// Wait is idempotent, and concurrent callers all get the same answer.
func TestVoter_WaitIsIdempotent(t *testing.T) {
	network := NewNetwork()
	defer network.Stop()

	globalIn := make(chan lifecycleItem, 10)
	v := newLifecycleVoter(t, network, globalIn)

	done := make(chan struct{})
	go func() { defer close(done); _ = v.Start() }()
	time.Sleep(50 * time.Millisecond)
	close(globalIn)
	<-done

	const callers = 4
	errs := make([]error, callers)
	var wg sync.WaitGroup
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs[i] = v.Wait()
		}(i)
	}
	wg.Wait()
	for i, err := range errs {
		assert.Equal(t, errs[0], err, "Wait caller %d saw a different result", i)
	}
	assert.NoError(t, v.Wait())
}

// A voter runs once. Waiting on one that never started must not sit out the whole
// timeout, and a later Start must decline rather than come up into a torn-down
// voter.
func TestVoter_WaitBeforeStart(t *testing.T) {
	network := NewNetwork()
	defer network.Stop()

	globalIn := make(chan lifecycleItem, 10)
	v := newLifecycleVoter(t, network, globalIn)
	v.waitTimeout = 10 * time.Second

	began := time.Now()
	assert.NoError(t, v.Wait())
	assert.Less(t, time.Since(began), time.Second, "Wait sat out its timeout for a Start that never ran")
	assert.Error(t, v.Start(), "a voter that has been waited on must not start")
}
