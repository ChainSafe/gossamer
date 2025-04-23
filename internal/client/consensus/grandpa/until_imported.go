// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"time"

	"github.com/ChainSafe/gossamer/internal/client/api"
	primitives "github.com/ChainSafe/gossamer/internal/primitives/consensus/grandpa"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/gammazero/deque"
)

const logPendingInterval = 15 * time.Second

// Something that needs to be withheld until specific blocks are available.
//
// For example a GRANDPA commit message which is not of any use without the corresponding block that it commits on.
type blockUntilImported[H runtime.Hash, N runtime.Number, Blocked any] interface {
	// Check if a new incoming item needs awaiting until a block(s) is imported.
	NeedsWaiting(
		input Blocked,
		statusCheck BlockStatus[H, N],
	) (discardWaitOrReady, error)

	// called when the wait has completed. The canonical number is passed through for further checks.
	WaitCompleted(canonNumber N) *Blocked
}

// Describes whether a given blockUntilImported (a) should be discarded, (b) is waiting for specific blocks to be
// imported or (c) is ready to be used.
//
// A reason for discarding a blockUntilImported would be if a referenced block is perceived under a different number
// than specified in the message.
type discardWaitOrReady interface {
	isDiscardWaitOrReady()
}
type discard struct{}

func (discard) isDiscardWaitOrReady() {}

type wait[H, N, M any] []struct {
	TargetHash   H
	TargetNumber N
	Wait         M
}

func (wait[H, N, M]) isDiscardWaitOrReady() {}

type ready[R any] struct {
	Ready R
}

func (ready[R]) isDiscardWaitOrReady() {}

// Buffering incoming messages until blocks with given hashes are imported.
type untilImported[
	H runtime.Hash,
	N runtime.Number,
	Header runtime.Header[N, H],
	Blocked any,
	M blockUntilImported[H, N, Blocked],
] struct {
	importNotifications <-chan api.BlockImportNotification[H, N, Header]
	blockSyncRequester  BlockSyncRequester[H, N]
	statusCheck         BlockStatus[H, N]
	incomingMessages    <-chan Blocked
	ready               deque.Deque[Blocked]
	// Interval at which to check status of each awaited block.
	checkPending <-chan time.Time
	// Mapping block hashes to their block number, the point in time it was first encountered (Instant) and a list of
	// GRANDPA messages referencing the block hash.
	pending map[H]pendingEntry[H, N, Blocked]
	// Queue identifier for differentiation in logs.
	identifier string
	// TODO: metrics
}
type pendingEntry[H runtime.Hash, N runtime.Number, Blocked any] struct {
	BlockNumber N
	LastLog     time.Time
	Wait        []blockUntilImported[H, N, Blocked]
}

func newUntilImported[
	H runtime.Hash,
	N runtime.Number,
	Header runtime.Header[N, H],
	Blocked any,
	M blockUntilImported[H, N, Blocked],
](
	importNotifications <-chan api.BlockImportNotification[H, N, Header],
	blockSyncRequester BlockSyncRequester[H, N],
	statusCheck BlockStatus[H, N],
	incomingMessages <-chan Blocked,
	identifier string,
) untilImported[H, N, Header, Blocked, M] {
	// how often to check if pending messages that are waiting for blocks to be imported can be checked.
	//
	// the import notifications interval takes care of most of this; this is used in the event of missed
	// import notifications
	const checkPendingInterval = 5 * time.Second

	checkPending := time.NewTicker(checkPendingInterval).C

	return untilImported[H, N, Header, Blocked, M]{
		importNotifications: importNotifications,
		blockSyncRequester:  blockSyncRequester,
		statusCheck:         statusCheck,
		incomingMessages:    incomingMessages,
		ready:               deque.Deque[Blocked]{},
		checkPending:        checkPending,
		pending:             make(map[H]pendingEntry[H, N, Blocked]),
		identifier:          identifier,
	}
}

func (ui *untilImported[H, N, Header, Blocked, M]) Chan() <-chan Blocked {
	ch := make(chan Blocked)
	go func() {
		defer close(ch)
		for {
			ready, blocked, err := ui.pollNext()
			if err != nil {
				return
			}
			if ready && blocked != nil {
				ch <- *blocked
			}
		}
	}()
	return ch
}

func (ui *untilImported[H, N, Header, Blocked, M]) pollNext() (bool, *Blocked, error) {
incoming:
	for {
		select {
		case b, ok := <-ui.incomingMessages:
			if !ok {
				return true, nil, nil
			}
			// new input: schedule wait of any parts which require blocks to be known.
			dwr, err := (*new(M)).NeedsWaiting(b, ui.statusCheck)
			if err != nil {
				return true, nil, err
			}
			switch dwr := dwr.(type) {
			case discard:
			case wait[H, N, M]:
				items := dwr
				for _, item := range items {
					_, ok := ui.pending[item.TargetHash]
					if !ok {
						ui.pending[item.TargetHash] = pendingEntry[H, N, Blocked]{
							BlockNumber: item.TargetNumber,
							LastLog:     time.Now(),
							Wait:        make([]blockUntilImported[H, N, Blocked], 0),
						}
					}
					p := ui.pending[item.TargetHash]
					p.Wait = append(ui.pending[item.TargetHash].Wait, item.Wait)
					ui.pending[item.TargetHash] = p
				}
			case ready[Blocked]:
				ui.ready.PushBack(dwr.Ready)
			}
		default:
			break incoming
		}
	}

imports:
	for {
		select {
		case notification, ok := <-ui.importNotifications:
			if !ok {
				return true, nil, nil
			}
			// new block imported. queue up all messages tied to that hash.
			if _, ok := ui.pending[notification.Hash]; ok {
				p := ui.pending[notification.Hash]
				delete(ui.pending, notification.Hash)
				canonNumber := notification.Header.Number()
				readyMessages := make([]Blocked, 0)
				for _, m := range p.Wait {
					blocked := m.WaitCompleted(canonNumber)
					if blocked != nil {
						readyMessages = append(readyMessages, *blocked)
					}
				}
				for _, m := range readyMessages {
					ui.ready.PushBack(m)
				}
			}
		default:
			break imports
		}
	}

	var updateInterval bool
	select {
	case <-ui.checkPending:
		updateInterval = true
	default:
	}

	if updateInterval {
		knownKeys := make([]HashNumber[H, N], 0)
		for blockHash, e := range ui.pending {

			number, err := ui.statusCheck.Number(blockHash)
			if err != nil {
				return true, nil, err
			}
			if number != nil {
				knownKeys = append(knownKeys, HashNumber[H, N]{Hash: blockHash, Number: *number})
			} else {
				nextLog := e.LastLog.Add(logPendingInterval)
				if time.Now().After(nextLog) {
					logger.Debugf(
						"Waiting to import block %s before %d %s messages can be imported. "+
							"Requesting network sync service to retrieve block from. Possible fork?",
						blockHash, len(e.Wait), ui.identifier)

					// NOTE: when sending an empty vec of peers the underlying should make a best effort to sync the
					// block from any peers it knows about.
					ui.blockSyncRequester.SetSyncForkRequest(
						nil,
						blockHash,
						e.BlockNumber,
					)
					e.LastLog = nextLog
					ui.pending[blockHash] = e
				}
			}
		}

		for _, hn := range knownKeys {
			if _, ok := ui.pending[hn.Hash]; ok {
				p := ui.pending[hn.Hash]
				delete(ui.pending, hn.Hash)
				readyMessages := make([]Blocked, 0)
				for _, m := range p.Wait {
					blocked := m.WaitCompleted(hn.Number)
					if blocked != nil {
						readyMessages = append(readyMessages, *blocked)
					}
				}
				for _, m := range readyMessages {
					ui.ready.PushBack(m)
				}
			}
		}
	}

	if ui.ready.Len() > 0 {
		ready := ui.ready.PopFront()
		return true, &ready, nil
	}

	return false, nil, nil
}

func warnAuthorityWrongTarget[H runtime.Hash](hash H, id primitives.AuthorityID) {
	logger.Warnf("Authority %s signed GRANDPA message with wrong block number for hash %s", id, hash)
}

type signedMessage[H runtime.Hash, N runtime.Number] struct {
	primitives.SignedMessage[H, N]
}

func (sm signedMessage[H, N]) NeedsWaiting(
	msg signedMessage[H, N],
	statusCheck BlockStatus[H, N],
) (discardWaitOrReady, error) {
	target := msg.Target()
	targetHash := target.Hash
	targetNumber := target.Number

	number, err := statusCheck.Number(targetHash)
	if err != nil {
		return nil, err
	}
	if number != nil {
		if *number != targetNumber {
			warnAuthorityWrongTarget(targetHash, msg.ID)
			return discard{}, nil
		} else {
			return ready[signedMessage[H, N]]{Ready: msg}, nil
		}
	}

	return wait[H, N, signedMessage[H, N]]{
		{
			TargetHash:   targetHash,
			TargetNumber: targetNumber,
			Wait:         msg,
		},
	}, nil
}

func (sm signedMessage[H, N]) WaitCompleted(canonNumber N) *signedMessage[H, N] {
	target := sm.Target()
	targetHash := target.Hash
	targetNumber := target.Number

	if canonNumber != targetNumber {
		warnAuthorityWrongTarget(targetHash, sm.ID)
		return nil
	}
	return &sm
}

// Helper type definition for the stream which waits until vote targets for signed messages are imported.
type untilVoteTargetImported[H runtime.Hash, N runtime.Number, Header runtime.Header[N, H]] struct {
	untilImported[H, N, Header, signedMessage[H, N], signedMessage[H, N]]
}

func newUntilVoteTargetImported[H runtime.Hash, N runtime.Number, Header runtime.Header[N, H]](
	importNotifications <-chan api.BlockImportNotification[H, N, Header],
	blockSyncRequester BlockSyncRequester[H, N],
	statusCheck BlockStatus[H, N],
	incomingMessages <-chan signedMessage[H, N],
	identifier string,
) untilVoteTargetImported[H, N, Header] {
	uvti := newUntilImported[H, N, Header, signedMessage[H, N], signedMessage[H, N]](
		importNotifications,
		blockSyncRequester,
		statusCheck,
		incomingMessages,
		identifier,
	)
	return untilVoteTargetImported[H, N, Header]{uvti}
}
