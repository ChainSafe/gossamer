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

// Closing globalIn is the shutdown signal: the voter stops of its own accord and
// Wait reports nil, because an orderly shutdown is not a failure.
func TestVoter_CloseGlobalInShutsDown(t *testing.T) {
	network := NewNetwork()
	defer network.Stop()

	globalIn := make(chan lifecycleItem, 10)
	v := newLifecycleVoter(t, network, globalIn)
	time.Sleep(50 * time.Millisecond)

	close(globalIn)
	select {
	case <-v.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("the voter did not stop after globalIn was closed")
	}
	assert.NoError(t, v.Wait())
}

// Done lets an owner observe termination without committing to a blocking Wait,
// and is still closed for a voter that has already been waited on.
func TestVoter_DoneSignalsTermination(t *testing.T) {
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
	require.NoError(t, v.Wait())

	select {
	case <-v.Done():
	default:
		t.Fatal("Done was not closed after the voter stopped")
	}
}

// The retired voter's forwarder must be gone once Wait returns. There is no
// early exit from the poll loop, so it can only leave by observing the close,
// which means first draining whatever the forwarder is holding.
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
		require.NoError(t, v.Wait())
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
		require.NoError(t, voter.Wait())

		globalIn = make(chan lifecycleItem, 100)
		voter = newLifecycleVoter(t, network, globalIn)
	}
	close(globalIn)
	require.NoError(t, voter.Wait())
}

// Wait does not signal shutdown. Calling it without closing globalIn must report
// a timeout rather than block forever or claim success, and must leave the voter
// running.
func TestVoter_WaitWithoutCloseTimesOut(t *testing.T) {
	network := NewNetwork()
	defer network.Stop()

	globalIn := make(chan lifecycleItem, 10)
	v := newLifecycleVoter(t, network, globalIn)
	v.waitTimeout = 200 * time.Millisecond
	time.Sleep(50 * time.Millisecond)

	assert.ErrorContains(t, v.Wait(), "timeout waiting for the voter to stop")
	select {
	case <-v.Done():
		t.Fatal("a timed-out Wait stopped the voter")
	default:
	}

	// The voter is still live, so the real shutdown still works.
	v.waitTimeout = 5 * time.Second
	close(globalIn)
	assert.NoError(t, v.Wait())
}

// Wait is idempotent, and concurrent callers all get the same answer.
func TestVoter_WaitIsIdempotent(t *testing.T) {
	network := NewNetwork()
	defer network.Stop()

	globalIn := make(chan lifecycleItem, 10)
	v := newLifecycleVoter(t, network, globalIn)
	time.Sleep(50 * time.Millisecond)
	close(globalIn)

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
