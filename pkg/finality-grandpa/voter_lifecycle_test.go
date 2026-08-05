// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type lifecycleItem = GlobalInItem[string, uint32, Signature, ID]

// newLifecycleVoter builds a voter over its own inbound channel, for tests about
// shutting down rather than about voting. NewVoter starts it.
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

// Closing globalIn is the shutdown signal, and Done reports nil for it: an
// orderly shutdown is not a failure.
func TestVoter_CloseGlobalInShutsDown(t *testing.T) {
	network := NewNetwork()
	defer network.Stop()

	globalIn := make(chan lifecycleItem, 10)
	v := newLifecycleVoter(t, network, globalIn)
	time.Sleep(50 * time.Millisecond)

	select {
	case <-v.Done():
		t.Fatal("Done fired while the voter was still running")
	default:
	}

	close(globalIn)
	select {
	case err := <-v.Done():
		assert.NoError(t, err, "an orderly shutdown is not an error")
	case <-time.After(5 * time.Second):
		t.Fatal("the voter did not stop after globalIn was closed")
	}
}

// Receiving from Done means the voter has finished releasing what it owned, not
// merely that its loop stopped.
func TestVoter_DoneImpliesTeardown(t *testing.T) {
	network := NewNetwork()
	defer network.Stop()

	globalIn := make(chan lifecycleItem, 10)
	v := newLifecycleVoter(t, network, globalIn)
	time.Sleep(50 * time.Millisecond)

	close(globalIn)
	require.NoError(t, <-v.Done())

	// Teardown closes the voter's own finalization channel, which ends its
	// forwarder and closes the channel the voter was reading.
	require.Eventually(t, func() bool {
		select {
		case _, open := <-v.finalizedNotifications.channel():
			return !open
		default:
			return false
		}
	}, 2*time.Second, 10*time.Millisecond, "teardown did not release the finalization channel")
}

// The error goes to one receiver and the channel closes behind it. A voter has a
// single owner; this pins the contract so it is not discovered by accident.
func TestVoter_DoneDeliversToOneReceiver(t *testing.T) {
	network := NewNetwork()
	defer network.Stop()

	globalIn := make(chan lifecycleItem, 10)
	v := newLifecycleVoter(t, network, globalIn)
	time.Sleep(50 * time.Millisecond)
	close(globalIn)

	require.NoError(t, <-v.Done())
	select {
	case err, open := <-v.Done():
		assert.False(t, open, "Done should be closed after delivering")
		assert.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("Done blocked after delivering; it should be closed")
	}
}

// The retired voter's forwarder must be gone once Done fires. There is no early
// exit from the poll loop, so it can only leave by observing the close, which
// means first draining whatever the forwarder is holding.
func TestVoter_RotationLeavesNoForwarder(t *testing.T) {
	network := NewNetwork()
	defer network.Stop()

	baseSend := forwardersInState("chan send")

	const rotations = 10
	for i := 0; i < rotations; i++ {
		globalIn := make(chan lifecycleItem, 100)
		v := newLifecycleVoter(t, network, globalIn)
		time.Sleep(30 * time.Millisecond)

		// Inbound traffic in flight when the set rotates.
		for j := 0; j < 100; j++ {
			select {
			case globalIn <- lifecycleItem{}:
			default:
			}
		}

		close(globalIn)
		require.NoError(t, <-v.Done())
	}

	time.Sleep(300 * time.Millisecond)
	assert.Equal(t, baseSend, forwardersInState("chan send"),
		"a retired voter's globalIn forwarder is still holding an item")
}

// Rebuilding the voter across an authority-set rotation. With no separate start
// step there is no window in which a half-live voter can be torn down.
func TestVoter_RebuildAcrossRotations(t *testing.T) {
	network := NewNetwork()
	defer network.Stop()

	globalIn := make(chan lifecycleItem, 100)
	voter := newLifecycleVoter(t, network, globalIn)

	const rotations = 100
	for i := 0; i < rotations; i++ {
		close(globalIn)
		require.NoError(t, <-voter.Done())

		globalIn = make(chan lifecycleItem, 100)
		voter = newLifecycleVoter(t, network, globalIn)
	}
	close(globalIn)
	require.NoError(t, <-voter.Done())
}

// A voter that keeps running must not accumulate goroutines. Every round wraps
// its inbound stream and its timers, and both have to be released as rounds
// advance rather than only at shutdown.
func TestVoter_RoundsDoNotAccumulateForwarders(t *testing.T) {
	network := NewNetwork()
	defer network.Stop()

	globalIn := make(chan lifecycleItem, 10)
	v := newLifecycleVoter(t, network, globalIn)
	defer func() {
		close(globalIn)
		<-v.Done()
	}()

	live := func() int {
		return forwardersInState("chan receive") + forwardersInState("chan send")
	}

	// Let the voter settle into a steady state before taking the baseline, so
	// start-up rounds are not counted as growth.
	time.Sleep(2 * time.Second)
	v.inner.Lock()
	firstRound := v.inner.bestRound.roundNumber()
	v.inner.Unlock()
	base := live()

	time.Sleep(8 * time.Second)
	v.inner.Lock()
	lastRound := v.inner.bestRound.roundNumber()
	v.inner.Unlock()
	grew := live() - base

	rounds := lastRound - firstRound
	require.Greater(t, rounds, uint64(2), "test needs several rounds to have elapsed")
	t.Logf("%d rounds elapsed, forwarders grew by %+d", rounds, grew)
	assert.LessOrEqual(t, grew, 2,
		"forwarders grow with rounds: %d over %d rounds", grew, rounds)
}
