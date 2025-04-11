package grandpa

import (
	"time"

	"github.com/ChainSafe/gossamer/internal/client/api"
	primitives "github.com/ChainSafe/gossamer/internal/primitives/consensus/grandpa"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/gammazero/deque"
)

// const LOG_PENDING_INTERVAL: Duration = Duration::from_secs(15);
const logPendingInterval = 15 * time.Second

// / Something that needs to be withheld until specific blocks are available.
// /
// / For example a GRANDPA commit message which is not of any use without the corresponding block
// / that it commits on.
// pub(crate) trait BlockUntilImported<Block: BlockT>: Sized {
type blockUntilImported[H runtime.Hash, N runtime.Number, Blocked any] interface {
	/// The type that is blocked on.
	// 	type Blocked;

	/// Check if a new incoming item needs awaiting until a block(s) is imported.
	// 	fn needs_waiting<S: BlockStatusT<Block>>(
	// 		input: Self::Blocked,
	// 		status_check: &S,
	// 	) -> Result<DiscardWaitOrReady<Block, Self, Self::Blocked>, Error>;
	NeedsWaiting(
		input Blocked,
		statusCheck BlockStatus[H, N],
	) (discardWaitOrReady, error)

	/// called when the wait has completed. The canonical number is passed through
	/// for further checks.
	// fn wait_completed(self, canon_number: NumberFor<Block>) -> Option<Self::Blocked>;
	WaitCompleted(canonNumber N) *Blocked
}

// / Describes whether a given [`BlockUntilImported`] (a) should be discarded, (b) is waiting for
// / specific blocks to be imported or (c) is ready to be used.
// /
// / A reason for discarding a [`BlockUntilImported`] would be if a referenced block is perceived
// / under a different number than specified in the message.
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

// /// Buffering incoming messages until blocks with given hashes are imported.
// pub(crate) struct UntilImported<Block, BlockStatus, BlockSyncRequester, I, M>
// where
//
//	Block: BlockT,
//	I: Stream<Item = M::Blocked> + Unpin,
//	M: BlockUntilImported<Block>,
//
// {
type untilImported[H runtime.Hash, N runtime.Number, Header runtime.Header[N, H], Blocked any, M blockUntilImported[H, N, Blocked]] struct {
	// 	import_notifications: Fuse<TracingUnboundedReceiver<BlockImportNotification<Block>>>,
	importNotifications <-chan api.BlockImportNotification[H, N, Header]
	// 	block_sync_requester: BlockSyncRequester,
	blockSyncRequester BlockSyncRequester[H, N]
	// 	status_check: BlockStatus,
	statusCheck BlockStatus[H, N]
	// 	incoming_messages: Fuse<I>,
	incomingMessages <-chan Blocked
	// 	ready: VecDeque<M::Blocked>,
	ready deque.Deque[Blocked]
	/// Interval at which to check status of each awaited block.
	// 	check_pending: Pin<Box<dyn Stream<Item = Result<(), std::io::Error>> + Send>>,
	checkPending <-chan time.Time
	/// Mapping block hashes to their block number, the point in time it was
	/// first encountered (Instant) and a list of GRANDPA messages referencing
	/// the block hash.
	// 	pending: HashMap<Block::Hash, (NumberFor<Block>, Instant, Vec<M>)>,
	pending map[H]pendingEntry[H, N, Blocked]

	/// Queue identifier for differentiation in logs.
	// identifier: &'static str,
	identifier string

	// TODO: metrics
}
type pendingEntry[H runtime.Hash, N runtime.Number, Blocked any] struct {
	BlockNumber N
	LastLog     time.Time
	Wait        []blockUntilImported[H, N, Blocked]
}

func newUntilImported[H runtime.Hash, N runtime.Number, Header runtime.Header[N, H], Blocked any, M blockUntilImported[H, N, Blocked]](
	importNotifications <-chan api.BlockImportNotification[H, N, Header],
	blockSyncRequester BlockSyncRequester[H, N],
	statusCheck BlockStatus[H, N],
	incomingMessages <-chan Blocked,
	identifier string,
) untilImported[H, N, Header, Blocked, M] {
	// how often to check if pending messages that are waiting for blocks to be
	// imported can be checked.
	//
	// the import notifications interval takes care of most of this; this is
	// used in the event of missed import notifications
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

// impl<Block, BStatus, BSyncRequester, I, M> Stream
// 	for UntilImported<Block, BStatus, BSyncRequester, I, M>
// where
// 	Block: BlockT,
// 	BStatus: BlockStatusT<Block>,
// 	BSyncRequester: BlockSyncRequesterT<Block>,
// 	I: Stream<Item = M::Blocked> + Unpin,
// 	M: BlockUntilImported<Block>,
// {
// 	type Item = Result<M::Blocked, Error>;

func (ui *untilImported[H, N, Header, Blocked, M]) Chan() <-chan Blocked {
	ch := make(chan Blocked)
	return ch
}

// fn poll_next(mut self: Pin<&mut Self>, cx: &mut Context) -> Poll<Option<Self::Item>> {
func (ui *untilImported[H, N, Header, Blocked, M]) pollNext() (bool, *Blocked, error) {
	// 		// We are using a `this` variable in order to allow multiple simultaneous mutable borrow to
	// 		// `self`.
	// 		let this = &mut *self;

	// 		loop {
	// 			match StreamExt::poll_next_unpin(&mut this.incoming_messages, cx) {
	// 				Poll::Ready(None) => return Poll::Ready(None),
	// 				Poll::Ready(Some(input)) => {
	// 					// new input: schedule wait of any parts which require
	// 					// blocks to be known.
	// 					match M::needs_waiting(input, &this.status_check)? {
	// 						DiscardWaitOrReady::Discard => {},
	// 						DiscardWaitOrReady::Wait(items) => {
	// 							for (target_hash, target_number, wait) in items {
	// 								this.pending
	// 									.entry(target_hash)
	// 									.or_insert_with(|| (target_number, Instant::now(), Vec::new()))
	// 									.2
	// 									.push(wait)
	// 							}
	// 						},
	// 						DiscardWaitOrReady::Ready(item) => this.ready.push_back(item),
	// 					}

	// 					if let Some(metrics) = &mut this.metrics {
	// 						metrics.waiting_messages_inc();
	// 					}
	// 				},
	// 				Poll::Pending => break,
	// 			}
	// 		}
incoming:
	for {
		select {
		case b, ok := <-ui.incomingMessages:
			if !ok {
				return true, nil, nil
			}
			// new input: schedule wait of any parts which require
			// blocks to be known.
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

	// 		loop {
	// 			match StreamExt::poll_next_unpin(&mut this.import_notifications, cx) {
	// 				Poll::Ready(None) => return Poll::Ready(None),
	// 				Poll::Ready(Some(notification)) => {
	// 					// new block imported. queue up all messages tied to that hash.
	// 					if let Some((_, _, messages)) = this.pending.remove(&notification.hash) {
	// 						let canon_number = *notification.header.number();
	// 						let ready_messages =
	// 							messages.into_iter().filter_map(|m| m.wait_completed(canon_number));

	// 						this.ready.extend(ready_messages);
	// 					}
	// 				},
	// 				Poll::Pending => break,
	// 			}
	// 		}
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

	// 		let mut update_interval = false;
	// 		while let Poll::Ready(Some(Ok(()))) = this.check_pending.poll_next_unpin(cx) {
	// 			update_interval = true;
	// 		}
	var updateInterval bool
	select {
	case <-ui.checkPending:
		updateInterval = true
	default:
	}

	if updateInterval {
		// 			let mut known_keys = Vec::new();
		knownKeys := make([]HashNumber[H, N], 0)
		// 			for (&block_hash, &mut (block_number, ref mut last_log, ref v)) in
		// 				this.pending.iter_mut()
		// 			{
		for blockHash, e := range ui.pending {
			// 				if let Some(number) = this.status_check.block_number(block_hash)? {
			// 					known_keys.push((block_hash, number));
			// 				} else {
			number, err := ui.statusCheck.Number(blockHash)
			if err != nil {
				return true, nil, err
			}
			if number != nil {
				knownKeys = append(knownKeys, HashNumber[H, N]{Hash: blockHash, Number: *number})
			} else {
				// 					let next_log = *last_log + LOG_PENDING_INTERVAL;
				nextLog := e.LastLog.Add(logPendingInterval)
				// 					if Instant::now() >= next_log {
				if time.Now().After(nextLog) {
					// 						debug!(
					// 							target: LOG_TARGET,
					// 							"Waiting to import block {} before {} {} messages can be imported. \
					// 							Requesting network sync service to retrieve block from. \
					// 							Possible fork?",
					// 							block_hash,
					// 							v.len(),
					// 							this.identifier,
					// 						);
					logger.Debugf(
						"Waiting to import block %s before %d %s messages can be imported. Requesting network sync service to retrieve block from. Possible fork?",
						blockHash, len(e.Wait), ui.identifier)

					// NOTE: when sending an empty vec of peers the
					// underlying should make a best effort to sync the
					// block from any peers it knows about.
					// 						this.block_sync_requester.set_sync_fork_request(
					// 							vec![],
					// 							block_hash,
					// 							block_number,
					// 						);
					ui.blockSyncRequester.SetSyncForkRequest(
						nil,
						blockHash,
						e.BlockNumber,
					)
					// 						*last_log = next_log;
					e.LastLog = nextLog
					ui.pending[blockHash] = e
				}
			}
		}

		// 			for (known_hash, canon_number) in known_keys {
		// 				if let Some((_, _, pending_messages)) = this.pending.remove(&known_hash) {
		// 					let ready_messages =
		// 						pending_messages.into_iter().filter_map(|m| m.wait_completed(canon_number));

		// 					this.ready.extend(ready_messages);
		// 				}
		// 			}
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

	// 		if let Some(ready) = this.ready.pop_front() {
	// 			if let Some(metrics) = &mut this.metrics {
	// 				metrics.waiting_messages_dec();
	// 			}
	// 			return Poll::Ready(Some(Ok(ready)))
	// 		}
	if ui.ready.Len() > 0 {
		ready := ui.ready.PopFront()
		return true, &ready, nil
	}

	//		if this.import_notifications.is_done() && this.incoming_messages.is_done() {
	//			Poll::Ready(None)
	//		} else {
	//			Poll::Pending
	//		}
	//	}
	return false, nil, nil
}

func warnAuthorityWrongTarget[H runtime.Hash](hash H, id primitives.AuthorityID) {
	logger.Warnf("Authority %s signed GRANDPA message with wrong block number for hash %s", id, hash)
}

type signedMessage[H runtime.Hash, N runtime.Number] struct {
	primitives.SignedMessage[H, N]
}

// impl<Block: BlockT> BlockUntilImported<Block> for SignedMessage<Block::Header> {
// 	type Blocked = Self;

// fn needs_waiting<BlockStatus: BlockStatusT<Block>>(
//
//		msg: Self::Blocked,
//		status_check: &BlockStatus,
//	) -> Result<DiscardWaitOrReady<Block, Self, Self::Blocked>, Error> {
func (sm signedMessage[H, N]) NeedsWaiting(
	msg signedMessage[H, N],
	statusCheck BlockStatus[H, N],
) (discardWaitOrReady, error) {
	// 		let (&target_hash, target_number) = msg.target();
	target := msg.Target()
	targetHash := target.Hash
	targetNumber := target.Number

	// 		if let Some(number) = status_check.block_number(target_hash)? {
	// 			if number != target_number {
	// 				warn_authority_wrong_target(target_hash, msg.id);
	// 				return Ok(DiscardWaitOrReady::Discard)
	// 			} else {
	// 				return Ok(DiscardWaitOrReady::Ready(msg))
	// 			}
	// 		}
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
	// 		Ok(DiscardWaitOrReady::Wait(vec![(target_hash, target_number, msg)]))
	return wait[H, N, signedMessage[H, N]]{
		{
			TargetHash:   targetHash,
			TargetNumber: targetNumber,
			Wait:         msg,
		},
	}, nil
}

// fn wait_completed(self, canon_number: NumberFor<Block>) -> Option<Self::Blocked> {
func (sm signedMessage[H, N]) WaitCompleted(canonNumber N) *signedMessage[H, N] {
	// 		let (&target_hash, target_number) = self.target();
	target := sm.Target()
	targetHash := target.Hash
	targetNumber := target.Number
	// 		if canon_number != target_number {
	// 			warn_authority_wrong_target(target_hash, self.id);

	//			None
	//		} else {
	//			Some(self)
	//		}
	//	}
	if canonNumber != targetNumber {
		warnAuthorityWrongTarget(targetHash, sm.ID)
		return nil
	}
	return &sm
}

// /// Helper type definition for the stream which waits until vote targets for
// /// signed messages are imported.
// pub(crate) type UntilVoteTargetImported<Block, BlockStatus, BlockSyncRequester, I> = UntilImported<
//
//	Block,
//	BlockStatus,
//	BlockSyncRequester,
//	I,
//	SignedMessage<<Block as BlockT>::Header>,
//
// >;
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
