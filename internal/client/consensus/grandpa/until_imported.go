// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/internal/client/api"
	primitives "github.com/ChainSafe/gossamer/internal/primitives/consensus/grandpa"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	grandpa "github.com/ChainSafe/gossamer/pkg/finality-grandpa"
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

type wait[H, N, M any] []waitItem[H, N, M]
type waitItem[H, N, M any] struct {
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
	constructor func() M
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
	constructor func() M,
) untilImported[H, N, Header, Blocked, M] {
	// how often to check if pending messages that are waiting for blocks to be imported can be checked.
	//
	// the import notifications interval takes care of most of this; this is used in the event of missed
	// import notifications
	const checkPendingInterval = 1 * time.Second

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
		constructor:         constructor,
	}
}

type blockedError[Blocked any] struct {
	Blocked Blocked
	Error   error
}

func (ui *untilImported[H, N, Header, Blocked, M]) Chan() <-chan blockedError[Blocked] {
	ch := make(chan blockedError[Blocked])
	go func() {
		defer close(ch)
		for {
			ready, blocked, err := ui.pollNext()
			if ready {
				if err != nil {
					ch <- blockedError[Blocked]{Error: err}
				} else if blocked != nil {
					ch <- blockedError[Blocked]{Blocked: *blocked}
				} else {
					// close channel, no more messages to process.
					return
				}
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
			dwr, err := ui.constructor().NeedsWaiting(b, ui.statusCheck)
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
				now := time.Now()
				if now.After(nextLog) || now.Equal(nextLog) {
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
	// 		let (&target_hash, target_number) = msg.target();
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
		func() signedMessage[H, N] {
			return signedMessage[H, N]{}
		},
	)
	return untilVoteTargetImported[H, N, Header]{uvti}
}

type refCount[T any] struct {
	inner *T
	count int
	sync.Mutex
}

// / This blocks a global message import, i.e. a commit or catch up messages,
// / until all blocks referenced in its votes are known.
// /
// / This is used for compact commits and catch up messages which have already
// / been checked for structural soundness (e.g. valid signatures).
// /
// / We use the `Arc`'s reference count to implicitly count the number of outstanding blocks that we
// / are waiting on for the same message (i.e. other `BlockGlobalMessage` instances with the same
// / `inner`).
//
//	pub(crate) struct BlockGlobalMessage<Block: BlockT> {
//		inner: Arc<Mutex<Option<CommunicationIn<Block>>>>,
//		target_number: NumberFor<Block>,
//	}
type blockGlobalMessage[H runtime.Hash, N runtime.Number] struct {
	inner        *refCount[communicationIn[H, N]]
	targetNumber N
}

type known[N runtime.Number] struct {
	number N
}
type unknown[N runtime.Number] struct {
	number N
}

func (k known[N]) Number() N   { return k.number }
func (u unknown[N]) Number() N { return u.number }

type knownOrUnknown[N runtime.Number] interface {
	Number() N
}

type itemToAwait[H runtime.Hash, N runtime.Number] struct {
	targetHash   H
	targetNumber N
	blockGlobalMessage[H, N]
}

// impl<Block: BlockT> Unpin for BlockGlobalMessage<Block> {}

// impl<Block: BlockT> BlockUntilImported<Block> for BlockGlobalMessage<Block> {
// 	type Blocked = CommunicationIn<Block>;

// fn needs_waiting<BlockStatus: BlockStatusT<Block>>(
//
//	input: Self::Blocked,
//	status_check: &BlockStatus,
//
// ) -> Result<DiscardWaitOrReady<Block, Self, Self::Blocked>, Error> {
func (bgm blockGlobalMessage[H, N]) NeedsWaiting(
	input communicationIn[H, N],
	statusCheck BlockStatus[H, N],
) (discardWaitOrReady, error) {
	// let mut checked_hashes: HashMap<_, KnownOrUnknown<NumberFor<Block>>> = HashMap::new();
	checkedHashes := map[H]knownOrUnknown[N]{}
	{
		// returns false when should early exit.
		// let mut query_known = |target_hash, perceived_number| -> Result<bool, Error> {
		var queryKnown = func(targetHash H, perceivedNumber N) (bool, error) {
			// let canon_number = match checked_hashes.entry(target_hash) {
			// 	Entry::Occupied(entry) => *entry.get().number(),
			// 	Entry::Vacant(entry) => {
			// 		if let Some(number) = status_check.block_number(target_hash)? {
			// 			entry.insert(KnownOrUnknown::Known(number));
			// 			number
			// 		} else {
			// 			entry.insert(KnownOrUnknown::Unknown(perceived_number));
			// 			perceived_number
			// 		}
			// 	},
			// };

			// check integrity: all votes for same hash have same number.
			var canonNumber N
			entry, ok := checkedHashes[targetHash]
			if ok {
				canonNumber = entry.Number()
			} else {
				number, err := statusCheck.Number(targetHash)
				if err != nil {
					return false, err
				}
				if number != nil {
					checkedHashes[targetHash] = known[N]{number: *number}
					canonNumber = *number
				} else {
					checkedHashes[targetHash] = unknown[N]{number: perceivedNumber}
					canonNumber = perceivedNumber
				}
			}

			// if canon_number != perceived_number {
			// 	// invalid global message: messages targeting wrong number
			// 	// or at least different from other vote in same global
			// 	// message.
			// 	return Ok(false)
			// }
			if canonNumber != perceivedNumber {
				// invalid global message: messages targeting wrong number
				// or at least different from other vote in same global
				// message.
				return false, nil
			}

			return true, nil
		}

		// match input {
		switch input := input.(type) {
		// 	voter::CommunicationIn::Commit(_, ref commit, ..) => {
		case grandpa.CommunicationInCommit[H, N, primitives.AuthoritySignature, primitives.AuthorityID]:
			// add known hashes from all precommits.
			// 		let precommit_targets =
			// 			commit.precommits.iter().map(|c| (c.target_number, c.target_hash));
			var precommitTargets []HashNumber[H, N]
			for _, c := range input.CompactCommit.Precommits {
				precommitTargets = append(precommitTargets, HashNumber[H, N]{
					Hash:   c.TargetHash,
					Number: c.TargetNumber,
				})
			}

			// 		for (target_number, target_hash) in precommit_targets {
			// 			if !query_known(target_hash, target_number)? {
			// 				return Ok(DiscardWaitOrReady::Discard)
			// 			}
			// 		}
			for _, target := range precommitTargets {
				known, err := queryKnown(target.Hash, target.Number)
				if err != nil {
					return nil, err
				}
				if !known {
					return discard{}, nil
				}
			}
		// 	},
		// 	voter::CommunicationIn::CatchUp(ref catch_up, ..) => {
		case grandpa.CommunicationInCatchUp[H, N, primitives.AuthoritySignature, primitives.AuthorityID]:
			// add known hashes from all prevotes and precommits.
			// 		let prevote_targets = catch_up
			// 			.prevotes
			// 			.iter()
			// 			.map(|s| (s.prevote.target_number, s.prevote.target_hash));
			prevoteTargets := make([]HashNumber[H, N], len(input.CatchUp.Prevotes))
			for i, s := range input.CatchUp.Prevotes {
				prevoteTargets[i] = HashNumber[H, N]{
					Hash:   s.Prevote.TargetHash,
					Number: s.Prevote.TargetNumber,
				}
			}

			// 		let precommit_targets = catch_up
			// 			.precommits
			// 			.iter()
			// 			.map(|s| (s.precommit.target_number, s.precommit.target_hash));
			precommitTargets := make([]HashNumber[H, N], len(input.CatchUp.Precommits))
			for i, s := range input.CatchUp.Precommits {
				precommitTargets[i] = HashNumber[H, N]{
					Hash:   s.Precommit.TargetHash,
					Number: s.Precommit.TargetNumber,
				}
			}

			// 		let targets = prevote_targets.chain(precommit_targets);
			targets := append(prevoteTargets, precommitTargets...)

			// 		for (target_number, target_hash) in targets {
			// 			if !query_known(target_hash, target_number)? {
			// 				return Ok(DiscardWaitOrReady::Discard)
			// 			}
			// 		}
			// 	},
			for _, target := range targets {
				known, err := queryKnown(target.Hash, target.Number)
				if err != nil {
					return nil, err
				}
				if !known {
					return discard{}, nil
				}
			}
		default:
			panic("unreachable")
		}
	}

	// 		let unknown_hashes = checked_hashes
	// 			.into_iter()
	// 			.filter_map(|(hash, num)| match num {
	// 				KnownOrUnknown::Unknown(number) => Some((hash, number)),
	// 				KnownOrUnknown::Known(_) => None,
	// 			})
	// 			.collect::<Vec<_>>();
	unknownHashes := make([]HashNumber[H, N], 0)
	for hash, num := range checkedHashes {
		switch num := num.(type) {
		case unknown[N]:
			unknownHashes = append(unknownHashes, HashNumber[H, N]{
				Hash:   hash,
				Number: num.Number(),
			})
		case known[N]:
		default:
			panic("unreachable")
		}
	}

	// 		if unknown_hashes.is_empty() {
	if len(unknownHashes) == 0 {
		// none of the hashes in the global message were unknown.
		// we can just return the message directly.
		// 			return Ok(DiscardWaitOrReady::Ready(input))
		return ready[communicationIn[H, N]]{input}, nil
	}

	// 		let locked_global = Arc::new(Mutex::new(Some(input)));
	lockedGlobal := &refCount[communicationIn[H, N]]{inner: &input, count: 0}

	// 		let items_to_await = unknown_hashes
	// 			.into_iter()
	// 			.map(|(hash, target_number)| {
	// 				(
	// 					hash,
	// 					target_number,
	// 					BlockGlobalMessage { inner: locked_global.clone(), target_number },
	// 				)
	// 			})
	// 			.collect();

	itemsToAwait := make(wait[H, N, *blockGlobalMessage[H, N]], len(unknownHashes))
	for i, target := range unknownHashes {
		lockedGlobal.count++
		itemsToAwait[i] = waitItem[H, N, *blockGlobalMessage[H, N]]{
			TargetHash:   target.Hash,
			TargetNumber: target.Number,
			Wait: &blockGlobalMessage[H, N]{
				inner:        lockedGlobal,
				targetNumber: target.Number,
			},
		}
	}

	// schedule waits for all unknown messages.
	// when the last one of these has `wait_completed` called on it,
	// the global message will be returned.
	// 		Ok(DiscardWaitOrReady::Wait(items_to_await))
	return itemsToAwait, nil
}

// fn wait_completed(self, canon_number: NumberFor<Block>) -> Option<Self::Blocked> {
func (bgm *blockGlobalMessage[H, N]) WaitCompleted(canonNumber N) *communicationIn[H, N] {
	// 		if self.target_number != canon_number {
	if bgm.targetNumber != canonNumber {
		// Delete the inner message so it won't ever be forwarded. Future calls to
		// `wait_completed` on the same `inner` will ignore it.
		// 			*self.inner.lock() = None;
		// 			return None
		bgm.inner.Lock()
		bgm.inner.inner = nil
		bgm.inner.Unlock()
	}

	//		match Arc::try_unwrap(self.inner) {
	bgm.inner.Lock()
	defer bgm.inner.Unlock()
	bgm.inner.count--
	if bgm.inner.count < 0 {
		panic("unreachable")
	}
	if bgm.inner.count == 0 {
		//			// This is the last reference and thus the last outstanding block to be awaited. `inner`
		//			// is either `Some(_)` or `None`. The latter implies that a previous `wait_completed`
		//			// call witnessed a block number mismatch (see above).
		//			Ok(inner) => Mutex::into_inner(inner),
		return bgm.inner.inner
	}
	// There are still other strong references to this `Arc`, thus the message is blocked on
	// other blocks to be imported.
	return nil
}

// / A stream which gates off incoming global messages, i.e. commit and catch up
// / messages, until all referenced block hashes have been imported.
// pub(crate) type UntilGlobalMessageBlocksImported<Block, BlockStatus, BlockSyncRequester, I> =
//
//	UntilImported<Block, BlockStatus, BlockSyncRequester, I, BlockGlobalMessage<Block>>;
type untilGlobalMessageBlocksImported[H runtime.Hash, N runtime.Number, Header runtime.Header[N, H]] struct {
	untilImported[H, N, Header, communicationIn[H, N], *blockGlobalMessage[H, N]]
}

func newUntilGlobalMessageBlocksImported[H runtime.Hash, N runtime.Number, Header runtime.Header[N, H]](
	importNotifications <-chan api.BlockImportNotification[H, N, Header],
	blockSyncRequester BlockSyncRequester[H, N],
	statusCheck BlockStatus[H, N],
	incomingMessages <-chan communicationIn[H, N],
	identifier string,
) untilGlobalMessageBlocksImported[H, N, Header] {
	ui := newUntilImported[H, N, Header, communicationIn[H, N], *blockGlobalMessage[H, N]](
		importNotifications,
		blockSyncRequester,
		statusCheck,
		incomingMessages,
		identifier,
		func() *blockGlobalMessage[H, N] {
			return &blockGlobalMessage[H, N]{}
		},
	)
	return untilGlobalMessageBlocksImported[H, N, Header]{ui}
}
