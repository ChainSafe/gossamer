package grandpa

import (
	"bytes"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/internal/client/api"
	peerid "github.com/ChainSafe/gossamer/internal/client/network/types/peer-id"
	"github.com/ChainSafe/gossamer/internal/primitives/consensus/common"
	primitives "github.com/ChainSafe/gossamer/internal/primitives/consensus/grandpa"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime/generic"
	grandpa "github.com/ChainSafe/gossamer/pkg/finality-grandpa"
	"github.com/stretchr/testify/require"
)

type TestChainState[H runtime.Hash, N runtime.Number, Header runtime.Header[N, H]] struct {
	sender         chan api.BlockImportNotification[H, N, Header]
	knownBlocks    map[H]N
	knownBlocksMtx sync.Mutex
}

func NewTestChainState[H runtime.Hash, N runtime.Number, Header runtime.Header[N, H]]() (*TestChainState[H, N, Header], api.ImportNotifications[H, N, Header]) {
	ch := make(api.ImportNotifications[H, N, Header], 100000)
	return &TestChainState[H, N, Header]{
		sender:      ch,
		knownBlocks: make(map[H]N),
	}, ch
}

func (t *TestChainState[H, N, Header]) BlockStatus() TestBlockStatus[H, N] {
	return TestBlockStatus[H, N]{
		inner:    &t.knownBlocks,
		innerMtx: &t.knownBlocksMtx,
	}
}

func (t *TestChainState[H, N, Header]) ImportHeader(header Header) {
	hash := header.Hash()
	number := header.Number()
	t.knownBlocksMtx.Lock()
	t.knownBlocks[hash] = number
	t.knownBlocksMtx.Unlock()

	t.sender <- api.BlockImportNotification[H, N, Header]{
		Hash:      hash,
		Origin:    common.FileBlockOrigin,
		Header:    header,
		IsNewBest: false,
		TreeRoute: nil,
	}
}

type TestBlockStatus[H runtime.Hash, N runtime.Number] struct {
	inner    *map[H]N
	innerMtx *sync.Mutex
}

func (tbs TestBlockStatus[H, N]) Number(hash H) (*N, error) {
	num, ok := (*tbs.inner)[hash]
	if !ok {
		return nil, nil
	}
	return &num, nil
}

type TestBlockSyncRequester[H runtime.Hash, N runtime.Number] struct {
	requests    []HashNumber[H, N]
	requestsMtx sync.Mutex
}

func (tbsr *TestBlockSyncRequester[H, N]) SetSyncForkRequest(peers []peerid.PeerID, hash H, number N) {
	tbsr.requestsMtx.Lock()
	defer tbsr.requestsMtx.Unlock()
	tbsr.requests = append(tbsr.requests, HashNumber[H, N]{Hash: hash, Number: number})
}

func makeHeader(number uint64) *generic.Header[uint64, hash.H256, runtime.BlakeTwo256] {
	return generic.NewHeader[uint64, hash.H256, runtime.BlakeTwo256](
		number,
		"",
		"",
		"",
		runtime.Digest{},
	)
}

type numCompactCommit[H runtime.Hash, N runtime.Number] struct {
	Number uint64
	primitives.CompactCommit[H, N]
}

// unwrap the commit from `CommunicationIn` returning its fields in a tuple,
// panics if the given message isn't a commit
func unapplyCommit[H runtime.Hash, N runtime.Number](msg communicationIn[H, N]) numCompactCommit[H, N] {
	switch msg := msg.(type) {
	case grandpa.CommunicationInCommit[H, N, primitives.AuthoritySignature, primitives.AuthorityID]:
		return numCompactCommit[H, N]{
			Number:        msg.Number,
			CompactCommit: primitives.CompactCommit[H, N](msg.CompactCommit),
		}
	default:
		panic("expected commit")
	}
}

// unwrap the catch up from `CommunicationIn` returning its inner representation,
// panics if the given message isn't a catch up
func unapplyCatchUp[H runtime.Hash, N runtime.Number](msg communicationIn[H, N]) grandpa.CatchUp[H, N, primitives.AuthoritySignature, primitives.AuthorityID] {
	switch msg := msg.(type) {
	case grandpa.CommunicationInCatchUp[H, N, primitives.AuthoritySignature, primitives.AuthorityID]:
		return msg.CatchUp
	default:
		panic("expected catch up")
	}
}

func messageAllDependenciesSatisfied(
	t *testing.T,
	msg communicationIn[hash.H256, uint64],
	enactDependencies func(*TestChainState[hash.H256, uint64, generic.Header[uint64, hash.H256, runtime.BlakeTwo256]]),
) communicationIn[hash.H256, uint64] {
	chainState, importNotifications := NewTestChainState[hash.H256, uint64, generic.Header[uint64, hash.H256, runtime.BlakeTwo256]]()
	blockStatus := chainState.BlockStatus()

	// enact all dependencies before importing the message
	enactDependencies(chainState)

	global := make(chan communicationIn[hash.H256, uint64], 100000)

	untilImported := newUntilGlobalMessageBlocksImported(
		importNotifications,
		&TestBlockSyncRequester[hash.H256, uint64]{},
		blockStatus,
		global,
		"global",
	)

	global <- msg

	untilImportedChan := untilImported.Chan()

	be := <-untilImportedChan
	require.NoError(t, be.Error)
	require.NotNil(t, be.Blocked)

	return be.Blocked
}

func blockingMessageOnDependencies(
	t *testing.T,
	msg communicationIn[hash.H256, uint64],
	enactDependencies func(*TestChainState[hash.H256, uint64, generic.Header[uint64, hash.H256, runtime.BlakeTwo256]]),
) communicationIn[hash.H256, uint64] {
	chainState, importNotifications := NewTestChainState[hash.H256, uint64, generic.Header[uint64, hash.H256, runtime.BlakeTwo256]]()
	blockStatus := chainState.BlockStatus()

	global := make(chan communicationIn[hash.H256, uint64], 100000)

	untilImported := newUntilGlobalMessageBlocksImported(
		importNotifications,
		&TestBlockSyncRequester[hash.H256, uint64]{},
		blockStatus,
		global,
		"global",
	)

	global <- msg

	timeout := time.NewTimer(100 * time.Millisecond)
	var timeoutFired bool
	untilImportedChan := untilImported.Chan()
	for {
		select {
		case <-timeout.C:
			// timeout fired. push in the headers.
			timeoutFired = true
			enactDependencies(chainState)
		case be := <-untilImportedChan:
			if !timeoutFired {
				t.Fatalf("timeout should have fired first")
			}
			require.NoError(t, be.Error)
			require.NotNil(t, be.Blocked)

			return be.Blocked
		}
	}
}

func TestUntilImported(t *testing.T) {
	t.Run("blocking_commit_message", func(t *testing.T) {
		h1 := makeHeader(5)
		h2 := makeHeader(6)
		h3 := makeHeader(7)

		unknownCommit := primitives.CompactCommit[hash.H256, uint64]{
			TargetHash:   h1.Hash(),
			TargetNumber: 5,
			Precommits: []grandpa.Precommit[hash.H256, uint64]{
				{TargetHash: h2.Hash(), TargetNumber: 6},
				{TargetHash: h3.Hash(), TargetNumber: 7},
			},
			AuthData: nil, // not used
		}

		uc := grandpa.CommunicationInCommit[hash.H256, uint64, primitives.AuthoritySignature, primitives.AuthorityID]{
			Number:        0,
			CompactCommit: grandpa.CompactCommit[hash.H256, uint64, primitives.AuthoritySignature, primitives.AuthorityID](unknownCommit),
			Callback:      func(cpo grandpa.CommitProcessingOutcome) {},
		}

		res := blockingMessageOnDependencies(t, uc, func(chainState *TestChainState[hash.H256, uint64, generic.Header[uint64, hash.H256, runtime.BlakeTwo256]]) {
			chainState.ImportHeader(*h1)
			chainState.ImportHeader(*h2)
			chainState.ImportHeader(*h3)
		})

		require.Equal(t, unapplyCommit[hash.H256, uint64](res), unapplyCommit[hash.H256, uint64](uc))
	})

	t.Run("commit_message_all_known", func(t *testing.T) {
		h1 := makeHeader(5)
		h2 := makeHeader(6)
		h3 := makeHeader(7)

		knownCommit := primitives.CompactCommit[hash.H256, uint64]{
			TargetHash:   h1.Hash(),
			TargetNumber: 5,
			Precommits: []grandpa.Precommit[hash.H256, uint64]{
				{TargetHash: h2.Hash(), TargetNumber: 6},
				{TargetHash: h3.Hash(), TargetNumber: 7},
			},
			AuthData: nil, // not used
		}

		kc := grandpa.CommunicationInCommit[hash.H256, uint64, primitives.AuthoritySignature, primitives.AuthorityID]{
			Number:        0,
			CompactCommit: grandpa.CompactCommit[hash.H256, uint64, primitives.AuthoritySignature, primitives.AuthorityID](knownCommit),
			Callback:      func(cpo grandpa.CommitProcessingOutcome) {},
		}

		res := blockingMessageOnDependencies(t, kc, func(chainState *TestChainState[hash.H256, uint64, generic.Header[uint64, hash.H256, runtime.BlakeTwo256]]) {
			chainState.ImportHeader(*h1)
			chainState.ImportHeader(*h2)
			chainState.ImportHeader(*h3)
		})

		require.Equal(t, unapplyCommit[hash.H256, uint64](res), unapplyCommit[hash.H256, uint64](kc))
	})

	t.Run("blocking_catch_up_message", func(t *testing.T) {
		h1 := makeHeader(5)
		h2 := makeHeader(6)
		h3 := makeHeader(7)

		signedPrevote := func(header *generic.Header[uint64, hash.H256, runtime.BlakeTwo256]) grandpa.SignedPrevote[hash.H256, uint64, primitives.AuthoritySignature, primitives.AuthorityID] {
			return grandpa.SignedPrevote[hash.H256, uint64, primitives.AuthoritySignature, primitives.AuthorityID]{
				ID:        primitives.AuthorityID(bytes.Repeat([]byte{1}, 32)),
				Signature: primitives.AuthoritySignature(bytes.Repeat([]byte{1}, 64)),
				Prevote: grandpa.Prevote[hash.H256, uint64]{
					TargetHash:   header.Hash(),
					TargetNumber: header.Number(),
				},
			}
		}

		signedPrecommit := func(header *generic.Header[uint64, hash.H256, runtime.BlakeTwo256]) grandpa.SignedPrecommit[hash.H256, uint64, primitives.AuthoritySignature, primitives.AuthorityID] {
			return grandpa.SignedPrecommit[hash.H256, uint64, primitives.AuthoritySignature, primitives.AuthorityID]{
				ID:        primitives.AuthorityID(bytes.Repeat([]byte{1}, 32)),
				Signature: primitives.AuthoritySignature(bytes.Repeat([]byte{1}, 64)),
				Precommit: grandpa.Precommit[hash.H256, uint64]{
					TargetHash:   header.Hash(),
					TargetNumber: header.Number(),
				},
			}
		}

		prevotes := []grandpa.SignedPrevote[hash.H256, uint64, primitives.AuthoritySignature, primitives.AuthorityID]{
			signedPrevote(h1),
			signedPrevote(h3),
		}

		precommits := []grandpa.SignedPrecommit[hash.H256, uint64, primitives.AuthoritySignature, primitives.AuthorityID]{
			signedPrecommit(h1),
			signedPrecommit(h2),
		}

		unknownCatchUp := grandpa.CatchUp[hash.H256, uint64, primitives.AuthoritySignature, primitives.AuthorityID]{
			RoundNumber: 1,
			Prevotes:    prevotes,
			Precommits:  precommits,
			BaseHash:    h1.Hash(),
			BaseNumber:  h1.Number(),
		}

		uc := grandpa.CommunicationInCatchUp[hash.H256, uint64, primitives.AuthoritySignature, primitives.AuthorityID]{
			CatchUp:  unknownCatchUp,
			Callback: func(cpo grandpa.CatchUpProcessingOutcome) {},
		}

		res := blockingMessageOnDependencies(t, uc, func(chainState *TestChainState[hash.H256, uint64, generic.Header[uint64, hash.H256, runtime.BlakeTwo256]]) {
			chainState.ImportHeader(*h1)
			chainState.ImportHeader(*h2)
			chainState.ImportHeader(*h3)
		})

		require.Equal(t, unapplyCatchUp[hash.H256, uint64](res), unapplyCatchUp[hash.H256, uint64](uc))
	})

	t.Run("catch_up_message_all_known", func(t *testing.T) {
		h1 := makeHeader(5)
		h2 := makeHeader(6)
		h3 := makeHeader(7)

		signedPrevote := func(header *generic.Header[uint64, hash.H256, runtime.BlakeTwo256]) grandpa.SignedPrevote[hash.H256, uint64, primitives.AuthoritySignature, primitives.AuthorityID] {
			return grandpa.SignedPrevote[hash.H256, uint64, primitives.AuthoritySignature, primitives.AuthorityID]{
				ID:        primitives.AuthorityID(bytes.Repeat([]byte{1}, 32)),
				Signature: primitives.AuthoritySignature(bytes.Repeat([]byte{1}, 64)),
				Prevote: grandpa.Prevote[hash.H256, uint64]{
					TargetHash:   header.Hash(),
					TargetNumber: header.Number(),
				},
			}
		}

		signedPrecommit := func(header *generic.Header[uint64, hash.H256, runtime.BlakeTwo256]) grandpa.SignedPrecommit[hash.H256, uint64, primitives.AuthoritySignature, primitives.AuthorityID] {
			return grandpa.SignedPrecommit[hash.H256, uint64, primitives.AuthoritySignature, primitives.AuthorityID]{
				ID:        primitives.AuthorityID(bytes.Repeat([]byte{1}, 32)),
				Signature: primitives.AuthoritySignature(bytes.Repeat([]byte{1}, 64)),
				Precommit: grandpa.Precommit[hash.H256, uint64]{
					TargetHash:   header.Hash(),
					TargetNumber: header.Number(),
				},
			}
		}

		prevotes := []grandpa.SignedPrevote[hash.H256, uint64, primitives.AuthoritySignature, primitives.AuthorityID]{
			signedPrevote(h1),
			signedPrevote(h3),
		}

		precommits := []grandpa.SignedPrecommit[hash.H256, uint64, primitives.AuthoritySignature, primitives.AuthorityID]{
			signedPrecommit(h1),
			signedPrecommit(h2),
		}

		unknownCatchUp := grandpa.CatchUp[hash.H256, uint64, primitives.AuthoritySignature, primitives.AuthorityID]{
			RoundNumber: 1,
			Prevotes:    prevotes,
			Precommits:  precommits,
			BaseHash:    h1.Hash(),
			BaseNumber:  h1.Number(),
		}

		uc := grandpa.CommunicationInCatchUp[hash.H256, uint64, primitives.AuthoritySignature, primitives.AuthorityID]{
			CatchUp:  unknownCatchUp,
			Callback: func(cpo grandpa.CatchUpProcessingOutcome) {},
		}

		res := messageAllDependenciesSatisfied(t, uc, func(chainState *TestChainState[hash.H256, uint64, generic.Header[uint64, hash.H256, runtime.BlakeTwo256]]) {
			chainState.ImportHeader(*h1)
			chainState.ImportHeader(*h2)
			chainState.ImportHeader(*h3)
		})

		require.Equal(t, unapplyCatchUp[hash.H256, uint64](res), unapplyCatchUp[hash.H256, uint64](uc))
	})

	t.Run("request_block_sync_for_needed_blocks", func(t *testing.T) {
		chainState, importNotifications := NewTestChainState[hash.H256, uint64, generic.Header[uint64, hash.H256, runtime.BlakeTwo256]]()
		blockStatus := chainState.BlockStatus()

		global := make(chan communicationIn[hash.H256, uint64], 100000)

		blockSyncRequester := &TestBlockSyncRequester[hash.H256, uint64]{}

		untilImported := newUntilGlobalMessageBlocksImported(
			importNotifications,
			blockSyncRequester,
			blockStatus,
			global,
			"global",
		)

		h1 := makeHeader(5)
		h2 := makeHeader(6)
		h3 := makeHeader(7)

		// we create a commit message, with precommits for blocks 6 and 7 which
		// we haven't imported.
		unknownCommit := primitives.CompactCommit[hash.H256, uint64]{
			TargetHash:   h1.Hash(),
			TargetNumber: 5,
			Precommits: []grandpa.Precommit[hash.H256, uint64]{
				{TargetHash: h2.Hash(), TargetNumber: 6},
				{TargetHash: h3.Hash(), TargetNumber: 7},
			},
			AuthData: nil, // not used
		}

		uc := grandpa.CommunicationInCommit[hash.H256, uint64, primitives.AuthoritySignature, primitives.AuthorityID]{
			Number:        0,
			CompactCommit: grandpa.CompactCommit[hash.H256, uint64, primitives.AuthoritySignature, primitives.AuthorityID](unknownCommit),
			Callback:      func(cpo grandpa.CommitProcessingOutcome) {},
		}

		// we send the commit message and spawn the until_imported stream
		global <- uc
		go func() {
			for range untilImported.Chan() {
				// consume the channel
			}
		}()

		// assert that we will make sync requests
		assert := make(chan struct{})
		go func() {
			for {
				blockSyncRequester.requestsMtx.Lock()
				blockSyncRequests := blockSyncRequester.requests
				blockSyncRequester.requestsMtx.Unlock()

				// we request blocks targeted by the precommits that aren't imported
				containsH2 := slices.ContainsFunc(blockSyncRequests, func(block HashNumber[hash.H256, uint64]) bool {
					return block.Hash == h2.Hash() && block.Number == h2.Number()
				})
				containsH3 := slices.ContainsFunc(blockSyncRequests, func(block HashNumber[hash.H256, uint64]) bool {
					return block.Hash == h3.Hash() && block.Number == h3.Number()
				})

				if containsH2 && containsH3 {
					close(assert)
					return
				}
			}
		}()

		// the `until_imported` stream doesn't request the blocks immediately,
		// but it should request them after a small timeout
		timeout := time.NewTimer(60 * time.Second)
		select {
		case <-assert:
			// success
		case <-timeout.C:
			t.Fatalf("timed out waiting for block sync request")
		}
	})

	t.Run("block_global_message_wait_completed_return_when_all_awaited", func(t *testing.T) {
		msg := testCatchUp()
		inner := &refCount[communicationIn[hash.H256, uint64]]{
			inner: &msg,
		}

		inner.count++
		waitingBlock1 := blockGlobalMessage[hash.H256, uint64]{
			inner:        inner,
			targetNumber: 1,
		}

		inner.count++
		waitingBlock2 := blockGlobalMessage[hash.H256, uint64]{
			inner:        inner,
			targetNumber: 2,
		}

		// waiting_block_2 is still waiting for block 2, thus this should return `None`.
		require.Nil(t, waitingBlock1.WaitCompleted(1))

		// Message only depended on block 1 and 2. Both have been imported, thus this should yield
		// the message.
		require.NotNil(t, waitingBlock2.WaitCompleted(2))
	})

	t.Run("block_global_message_wait_completed_return_none_on_block_number_mismatch", func(t *testing.T) {
		msg := testCatchUp()
		inner := &refCount[communicationIn[hash.H256, uint64]]{
			inner: &msg,
		}

		inner.count++
		waitingBlock1 := blockGlobalMessage[hash.H256, uint64]{
			inner:        inner,
			targetNumber: 1,
		}

		inner.count++
		waitingBlock2 := blockGlobalMessage[hash.H256, uint64]{
			inner:        inner,
			targetNumber: 2,
		}

		// Calling wait_completed with wrong block number should yield None.
		require.Nil(t, waitingBlock1.WaitCompleted(1234))

		// All blocks, that the message depended on, have been imported. Still, given the above
		// block number mismatch this should return None.
		require.Nil(t, waitingBlock2.WaitCompleted(2))
	})

}

func testCatchUp() communicationIn[hash.H256, uint64] {
	header := makeHeader(5)

	unknownCatchUp := grandpa.CatchUp[hash.H256, uint64, primitives.AuthoritySignature, primitives.AuthorityID]{
		RoundNumber: 1,
		Precommits:  []grandpa.SignedPrecommit[hash.H256, uint64, primitives.AuthoritySignature, primitives.AuthorityID]{},
		Prevotes:    []grandpa.SignedPrevote[hash.H256, uint64, primitives.AuthoritySignature, primitives.AuthorityID]{},
		BaseHash:    header.Hash(),
		BaseNumber:  header.Number(),
	}

	catchUp := grandpa.CommunicationInCatchUp[hash.H256, uint64, primitives.AuthoritySignature, primitives.AuthorityID]{
		CatchUp:  unknownCatchUp,
		Callback: func(cpo grandpa.CatchUpProcessingOutcome) {},
	}

	return catchUp
}
