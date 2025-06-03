package grandpa

import (
	"sync"

	"github.com/ChainSafe/gossamer/internal/client/api"
	"github.com/ChainSafe/gossamer/internal/client/api/utils"
	client_common "github.com/ChainSafe/gossamer/internal/client/consensus/common"
	shareddata "github.com/ChainSafe/gossamer/internal/client/consensus/common/shared-data"
	"github.com/ChainSafe/gossamer/internal/primitives/consensus/common"
	"github.com/ChainSafe/gossamer/internal/primitives/consensus/grandpa"
	"github.com/ChainSafe/gossamer/internal/primitives/crypto/hashing"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	forktree "github.com/ChainSafe/gossamer/internal/utils/fork-tree"
)

// enum AppliedChanges<H, N> {
type importAppliedChanges interface {
	needsJustification() bool
}

// Standard(bool), // true if the change is ready to be applied (i.e. it's a root)
type importAppliedChangesStandard bool

// Forced(NewAuthoritySet<H, N>),
type importAppliedChangesForced[H runtime.Hash, N runtime.Number] newAuthoritySet[H, N]

// None,
type importAppliedChangesNone struct{}

func (importAppliedChangesStandard) needsJustification() bool {
	return true
}
func (importAppliedChangesForced[H, N]) needsJustification() bool {
	return false
}
func (importAppliedChangesNone) needsJustification() bool {
	return false
}

// }

type justInCase[H runtime.Hash, N runtime.Number] struct {
	old AuthoritySet[H, N]
	shareddata.SharedDataLocked[AuthoritySet[H, N]]
}

// struct PendingSetChanges<Block: BlockT> {
type pendingSetChanges[
	H runtime.Hash,
	N runtime.Number,
	Hasher runtime.Hasher[H],
	Header runtime.Header[N, H],
] struct {
	// just_in_case: Option<(
	//
	//	AuthoritySet<Block::Hash, NumberFor<Block>>,
	//	SharedDataLockedUpgradable<AuthoritySet<Block::Hash, NumberFor<Block>>>,
	//
	// )>,
	justInCase *justInCase[H, N]
	// applied_changes: AppliedChanges<Block::Hash, NumberFor<Block>>,
	appliedChanges importAppliedChanges
	// do_pause: bool,
	doPause bool
}

// / A block-import handler for GRANDPA.
// /
// / This scans each imported block for signals of changing authority set.
// / If the block being imported enacts an authority set change then:
// / - If the current authority set is still live: we import the block
// / - Otherwise, the block must include a valid justification.
// /
// / When using GRANDPA, the block import worker should be using this block import
// / object.im
// pub struct GrandpaBlockImport<Backend, Block: BlockT, Client, SC> {
type GrandpaBlockImport[
	H runtime.Hash,
	N runtime.Number,
	Hasher runtime.Hasher[H],
	Header runtime.Header[N, H],
	E runtime.Extrinsic,
] struct {
	// inner: Arc<Client>,
	inner ClientForGrandpa[H, N, Hasher, Header, E]
	// justification_import_period: u32,
	justificationImportPeriod uint32
	// select_chain: SC,
	selectChain common.SelectChain[H, N, Header]
	// authority_set: SharedAuthoritySet<Block::Hash, NumberFor<Block>>,
	authoritySet *SharedAuthoritySet[H, N]
	// send_voter_commands: TracingUnboundedSender<VoterCommand<Block::Hash, NumberFor<Block>>>,
	sendVoterCommands chan voterCommand
	// authority_set_hard_forks:
	//
	//	Mutex<HashMap<Block::Hash, PendingChange<Block::Hash, NumberFor<Block>>>>,
	authoritySetHardForks    map[H]PendingChange[H, N]
	authoritySetHardForksMtx sync.Mutex
	//
	// justification_sender: GrandpaJustificationSender<Block>,
	justificationSender GrandpaJustificationSender[H, N, Header]
	// telemetry: Option<TelemetryHandle>,
	// TODO: telemetry
	// _phantom: PhantomData<Backend>,
}

//	impl<Backend, Block: BlockT, Client, SC> GrandpaBlockImport<Backend, Block, Client, SC> {
//		pub(crate) fn new(
//			inner: Arc<Client>,
//			justification_import_period: u32,
//			select_chain: SC,
//			authority_set: SharedAuthoritySet<Block::Hash, NumberFor<Block>>,
//			send_voter_commands: TracingUnboundedSender<VoterCommand<Block::Hash, NumberFor<Block>>>,
//			authority_set_hard_forks: Vec<(SetId, PendingChange<Block::Hash, NumberFor<Block>>)>,
//			justification_sender: GrandpaJustificationSender<Block>,
//			telemetry: Option<TelemetryHandle>,
//		) -> GrandpaBlockImport<Backend, Block, Client, SC> {
func newGrandpaBlockImport[
	H runtime.Hash,
	N runtime.Number,
	Hasher runtime.Hasher[H],
	Header runtime.Header[N, H],
	E runtime.Extrinsic,
](
	inner ClientForGrandpa[H, N, Hasher, Header, E],
	justificationImportPeriod uint32,
	selectChain common.SelectChain[H, N, Header],
	sharedAuthoritySet *SharedAuthoritySet[H, N],
	sendVoterCommands chan voterCommand,
	authoritySetHardForks []struct {
		SetID
		PendingChange[H, N]
	},
	justificationSender GrandpaJustificationSender[H, N, Header],
	// TODO: telemetry
) *GrandpaBlockImport[H, N, Hasher, Header, E] {
	// check for and apply any forced authority set hard fork that applies
	// to the *current* authority set.
	// 		if let Some((_, change)) = authority_set_hard_forks
	// 			.iter()
	// 			.find(|(set_id, _)| *set_id == authority_set.set_id())
	// 		{
	// 			authority_set.inner().current_authorities = change.next_authorities.clone();
	// 		}
	for _, hardFork := range authoritySetHardForks {
		setID := hardFork.SetID
		if setID == SetID(sharedAuthoritySet.SetID()) {
			authoritySet, unlock := sharedAuthoritySet.inner.DataMut()
			// authoritySet.mtx.Lock()
			authoritySet.CurrentAuthorities = hardFork.PendingChange.NextAuthorities
			// authoritySet.mtx.Unlock()
			unlock()
		}
	}

	// index authority set hard forks by block hash so that they can be used
	// by any node syncing the chain and importing a block hard fork
	// authority set changes.
	// 		let authority_set_hard_forks = authority_set_hard_forks
	// 			.into_iter()
	// 			.map(|(_, change)| (change.canon_hash, change))
	// 			.collect::<HashMap<_, _>>();
	var authoritySetHardForksMap = make(map[H]PendingChange[H, N])
	for _, hardFork := range authoritySetHardForks {
		authoritySetHardForksMap[hardFork.PendingChange.CanonHash] = hardFork.PendingChange
	}

	// check for and apply any forced authority set hard fork that apply to
	// any *pending* standard changes, checking by the block hash at which
	// they were announced.
	// 		{
	// 			let mut authority_set = authority_set.inner();

	// 			authority_set.pending_standard_changes =
	// 				authority_set.pending_standard_changes.clone().map(&mut |hash, _, original| {
	// 					authority_set_hard_forks.get(hash).cloned().unwrap_or(original)
	// 				});
	// 		}
	// authoritySet.mtx.Lock()
	// authSet := &authoritySet.inner
	authoritySet, unlock := sharedAuthoritySet.inner.DataMut()

	authoritySet.PendingStandardChanges = forktree.Map(authoritySet.PendingStandardChanges, func(hash H, _ N, original PendingChange[H, N]) PendingChange[H, N] {
		if change, ok := authoritySetHardForksMap[hash]; ok {
			return change
		}
		return original
	})
	// authoritySet.mtx.Unlock()
	unlock()

	// 		GrandpaBlockImport {
	// 			inner,
	// 			justification_import_period,
	// 			select_chain,
	// 			authority_set,
	// 			send_voter_commands,
	// 			authority_set_hard_forks: Mutex::new(authority_set_hard_forks),
	// 			justification_sender,
	// 			telemetry,
	// 			_phantom: PhantomData,
	// 		}
	return &GrandpaBlockImport[H, N, Hasher, Header, E]{
		inner:                     inner,
		justificationImportPeriod: justificationImportPeriod,
		selectChain:               selectChain,
		authoritySet:              sharedAuthoritySet,
		sendVoterCommands:         sendVoterCommands,
		authoritySetHardForks:     authoritySetHardForksMap,
		justificationSender:       justificationSender,
	}
}

// / Checks the given header for a consensus digest signalling a **standard** scheduled change and
// / extracts it.
// pub fn find_scheduled_change<B: BlockT>(
//
//	header: &B::Header,
//
// ) -> Option<ScheduledChange<NumberFor<B>>> {
func FindScheduledChange[
	H runtime.Hash,
	N runtime.Number,
	Header runtime.Header[N, H],
](
	header Header,
) *grandpa.ScheduledChange[N] {
	// 	let id = OpaqueDigestItemId::Consensus(&GRANDPA_ENGINE_ID);
	id := runtime.OpaqueDigestItemIDConsensus(grandpa.GrandpaEngineID)

	// 	let filter_log = |log: ConsensusLog<NumberFor<B>>| match log {
	// 		ConsensusLog::ScheduledChange(change) => Some(change),
	// 		_ => None,
	// 	};
	filterLog := func(log grandpa.ConsensusLog) *grandpa.ScheduledChange[N] {
		scheduledChange, ok := log.(grandpa.ConensusLogScheduledChange[N])
		if !ok {
			return nil
		}
		orig := grandpa.ScheduledChange[N](scheduledChange)
		return &orig
	}
	// find the first consensus digest with the right ID which converts to
	// the right kind of consensus log.
	// header.digest().convert_first(|l| l.try_to(id).and_then(filter_log))
	for _, log := range header.Digest().Logs {
		logVDT := runtime.DigestItemTryTo[grandpa.ConsensusLogVDT[N]](log, id)
		if logVDT == nil {
			continue
		}
		val, err := logVDT.Value()
		if err != nil {
			continue
		}
		log := val.(grandpa.ConsensusLog)
		sc := filterLog(log)
		if sc != nil {
			return sc
		}
	}
	return nil
}

type ForcedChange[H runtime.Hash, N runtime.Number] struct {
	MedianLastFinalized N
	grandpa.ScheduledChange[N]
}

// / Checks the given header for a consensus digest signalling a **forced** scheduled change and
// / extracts it.
// pub fn find_forced_change<B: BlockT>(
//
//	header: &B::Header,
//
// ) -> Option<(NumberFor<B>, ScheduledChange<NumberFor<B>>)> {
func FindForcedChange[
	H runtime.Hash,
	N runtime.Number,
	Header runtime.Header[N, H],
](
	header Header,
) *ForcedChange[H, N] {
	// 	let id = OpaqueDigestItemId::Consensus(&GRANDPA_ENGINE_ID);
	id := runtime.OpaqueDigestItemIDConsensus(grandpa.GrandpaEngineID)

	// 	let filter_log = |log: ConsensusLog<NumberFor<B>>| match log {
	// 		ConsensusLog::ForcedChange(delay, change) => Some((delay, change)),
	// 		_ => None,
	// 	};
	filterLog := func(log grandpa.ConsensusLog) *ForcedChange[H, N] {
		forcedChange, ok := log.(grandpa.ConsensusLogForcedChange[N])
		if !ok {
			return nil
		}
		return &ForcedChange[H, N]{
			MedianLastFinalized: forcedChange.Delay,
			ScheduledChange:     forcedChange.ScheduledChange,
		}
	}

	// 	// find the first consensus digest with the right ID which converts to
	// 	// the right kind of consensus log.
	// 	header.digest().convert_first(|l| l.try_to(id).and_then(filter_log))
	for _, log := range header.Digest().Logs {
		logVDT := runtime.DigestItemTryTo[grandpa.ConsensusLogVDT[N]](log, id)
		if logVDT == nil {
			continue
		}
		val, err := logVDT.Value()
		if err != nil {
			continue
		}
		log := val.(grandpa.ConsensusLog)
		fc := filterLog(log)
		if fc != nil {
			return fc
		}
	}
	return nil
}

// impl<BE, Block: BlockT, Client, SC> GrandpaBlockImport<BE, Block, Client, SC>
// where
//
//	NumberFor<Block>: finality_grandpa::BlockNumberOps,
//	BE: Backend<Block>,
//	Client: ClientForGrandpa<Block, BE>,
//	Client::Api: GrandpaApi<Block>,
//	for<'a> &'a Client: BlockImport<Block, Error = ConsensusError>,
//
//	{
//		// check for a new authority set change.
//		fn check_new_change(
//			&self,
//			header: &Block::Header,
//			hash: Block::Hash,
//		) -> Option<PendingChange<Block::Hash, NumberFor<Block>>> {
func (gbi *GrandpaBlockImport[H, N, Hasher, Header, E]) checkNewChange(
	header Header,
	hash H,
) *PendingChange[H, N] {
	// check for forced authority set hard forks
	// 		if let Some(change) = self.authority_set_hard_forks.lock().get(&hash) {
	// 			return Some(change.clone())
	// 		}
	gbi.authoritySetHardForksMtx.Lock()
	change, ok := gbi.authoritySetHardForks[hash]
	gbi.authoritySetHardForksMtx.Unlock()
	if ok {
		return &change
	}

	// check for forced change.
	// 		if let Some((median_last_finalized, change)) = find_forced_change::<Block>(header) {
	// 			return Some(PendingChange {
	// 				next_authorities: change.next_authorities,
	// 				delay: change.delay,
	// 				canon_height: *header.number(),
	// 				canon_hash: hash,
	// 				delay_kind: DelayKind::Best { median_last_finalized },
	// 			})
	// 		}
	fc := FindForcedChange[H, N](header)
	if fc != nil {
		return &PendingChange[H, N]{
			NextAuthorities: fc.ScheduledChange.NextAuthorities,
			Delay:           fc.ScheduledChange.Delay,
			CanonHeight:     header.Number(),
			CanonHash:       hash,
			DelayKind:       delayKindBest[N]{MedianLastFinalized: fc.MedianLastFinalized},
		}
	}

	// check normal scheduled change.
	// 		let change = find_scheduled_change::<Block>(header)?;
	// 		Some(PendingChange {
	// 			next_authorities: change.next_authorities,
	// 			delay: change.delay,
	// 			canon_height: *header.number(),
	// 			canon_hash: hash,
	// 			delay_kind: DelayKind::Finalized,
	// 		})
	scheduled := FindScheduledChange[H, N](header)
	return &PendingChange[H, N]{
		NextAuthorities: scheduled.NextAuthorities,
		Delay:           scheduled.Delay,
		CanonHeight:     header.Number(),
		CanonHash:       hash,
		DelayKind:       delayKindFinalized{},
	}
}

//	struct InnerGuard<'a, H, N> {
//		old: Option<AuthoritySet<H, N>>,
//		guard: Option<SharedDataLocked<'a, AuthoritySet<H, N>>>,
//	}
type innerGuard[H runtime.Hash, N runtime.Number] struct {
	old   *AuthoritySet[H, N]
	guard *shareddata.SharedDataLocked[AuthoritySet[H, N]]
}

//	impl<'a, H, N> InnerGuard<'a, H, N> {
//		fn as_mut(&mut self) -> &mut AuthoritySet<H, N> {
//			self.guard.as_mut().expect("only taken on deconstruction; qed")
//		}
func (ig *innerGuard[H, N]) asMut() *AuthoritySet[H, N] {
	if ig.guard == nil {
		panic("guard is nil; only taken on deconstruction; qed")
	}
	return ig.guard.MutRef()
}

//	fn set_old(&mut self, old: AuthoritySet<H, N>) {
//		if self.old.is_none() {
//			// ignore "newer" old changes.
//			self.old = Some(old);
//		}
//	}
func (ig *innerGuard[H, N]) setOld(old AuthoritySet[H, N]) {
	if ig.old == nil {
		// ignore "newer" old changes.
		ig.old = &old
	}
}

//		fn consume(
//			mut self,
//		) -> Option<(AuthoritySet<H, N>, SharedDataLocked<'a, AuthoritySet<H, N>>)> {
//			self.old
//				.take()
//				.map(|old| (old, self.guard.take().expect("only taken on deconstruction; qed")))
//		}
//	}
type consumed[H runtime.Hash, N runtime.Number] struct {
	old AuthoritySet[H, N]
	shareddata.SharedDataLocked[AuthoritySet[H, N]]
}

func (ig *innerGuard[H, N]) consume() *consumed[H, N] {
	old := ig.old
	ig.old = nil
	if old == nil {
		return nil
	}
	if ig.guard == nil {
		panic("guard is nil; only taken on deconstruction; qed")
	}
	return &consumed[H, N]{
		old:              *old,
		SharedDataLocked: *ig.guard,
	}
}

//	impl<'a, H, N> Drop for InnerGuard<'a, H, N> {
//		fn drop(&mut self) {
//			if let (Some(mut guard), Some(old)) = (self.guard.take(), self.old.take()) {
//				*guard = old;
//			}
//		}
//	}
func (ig *innerGuard[H, N]) drop() {
	// if let (Some(mut guard), Some(old)) = (self.guard.take(), self.old.take()) {
	// 	*guard = old;
	// }
	guard := ig.guard
	ig.guard = nil
	old := ig.old
	ig.old = nil
	if guard != nil && old != nil {
		*ig.guard.MutRef() = *ig.old
	}
}

// fn make_authorities_changes(
//
//	&self,
//	block: &mut BlockImportParams<Block>,
//	hash: Block::Hash,
//	initial_sync: bool,
//
// ) -> Result<PendingSetChanges<Block>, ConsensusError> {
func (gbi *GrandpaBlockImport[H, N, Hasher, Header, E]) makeAuthoritiesChanges(
	block *client_common.BlockImportParams[H, N, E, Header],
	hash H,
	initialSync bool,
) (*pendingSetChanges[H, N, Hasher, Header], error) {
	// when we update the authorities, we need to hold the lock
	// until the block is written to prevent a race if we need to restore
	// the old authority set on error or panic.

	// 		let number = *(block.header.number());
	// 		let maybe_change = self.check_new_change(&block.header, hash);
	number := block.Header.Number()
	maybeChange := gbi.checkNewChange(block.Header, hash)

	// returns a function for checking whether a block is a descendent of another
	// consistent with querying client directly after importing the block.
	// 		let parent_hash = *block.header.parent_hash();
	// 		let is_descendent_of = is_descendent_of(&*self.inner, Some((hash, parent_hash)));
	parentHash := block.Header.ParentHash()
	isDescendentOf := utils.IsDescendantOf(gbi.inner, &utils.HashParent[H]{Hash: hash, Parent: parentHash})
	// 		let mut guard = InnerGuard { guard: Some(self.authority_set.inner_locked()), old: None };
	locked := gbi.authoritySet.inner.Locked()
	defer locked.Unlock()
	guard := innerGuard[H, N]{
		old:   nil,
		guard: &locked,
	}

	// whether to pause the old authority set -- happens after import
	// of a forced change block.
	// 		let mut do_pause = false;
	var doPause bool

	// add any pending changes.
	// 		if let Some(change) = maybe_change {
	// 			let old = guard.as_mut().clone();
	// 			guard.set_old(old);

	// 			if let DelayKind::Best { .. } = change.delay_kind {
	// 				do_pause = true;
	// 			}

	// 			guard
	// 				.as_mut()
	// 				.add_pending_change(change, &is_descendent_of)
	// 				.map_err(|e| ConsensusError::ClientImport(e.to_string()))?;
	// 		}
	if maybeChange != nil {
		change := *maybeChange
		old := guard.asMut().Clone()
		guard.setOld(old)

		if _, ok := change.DelayKind.(delayKindBest[N]); ok {
			doPause = true
		}

		err := guard.asMut().addPendingChange(change, isDescendentOf)
		if err != nil {
			return nil, err
		}
	}

	// 		let applied_changes = {
	// 			let forced_change_set = guard
	// 				.as_mut()
	// 				.apply_forced_changes(
	// 					hash,
	// 					number,
	// 					&is_descendent_of,
	// 					initial_sync,
	// 					self.telemetry.clone(),
	// 				)
	// 				.map_err(|e| ConsensusError::ClientImport(e.to_string()))
	// 				.map_err(ConsensusError::from)?;
	var appliedChanges importAppliedChanges
	forcedChangeSet, err := guard.asMut().applyForcedChanges(
		hash,
		number,
		isDescendentOf,
	)
	if err != nil {
		return nil, err
	}

	// 			if let Some((median_last_finalized_number, new_set)) = forced_change_set {
	if forcedChangeSet != nil {
		medianLastFinalizedNumber := forcedChangeSet.median
		newSet := forcedChangeSet.set
		// 				let new_authorities = {
		// 					let (set_id, new_authorities) = new_set.current();
		setID, newAuthorities := newSet.current()

		// we will use the median last finalized number as a hint
		// for the canon block the new authority set should start
		// with. we use the minimum between the median and the local
		// best finalized block.
		// 					let best_finalized_number = self.inner.info().finalized_number;
		// 					let canon_number = best_finalized_number.min(median_last_finalized_number);
		// 					let canon_hash = self.inner.hash(canon_number)
		// 							.map_err(|e| ConsensusError::ClientImport(e.to_string()))?
		// 							.expect(
		// 								"the given block number is less or equal than the current best finalized number; \
		// 								 current best finalized number must exist in chain; qed."
		// 							);
		bestFinalizedNumber := gbi.inner.Info().FinalizedNumber
		canonNumber := bestFinalizedNumber
		if medianLastFinalizedNumber < bestFinalizedNumber {
			canonNumber = medianLastFinalizedNumber
		}
		canonHash, err := gbi.inner.Hash(canonNumber)
		if err != nil {
			return nil, err
		}
		if canonHash == nil {
			panic("the given block number is less or equal than the current best finalized number; current best finalized number must exist in chain; qed.")
		}

		// 					NewAuthoritySet {
		// 						canon_number,
		// 						canon_hash,
		// 						set_id,
		// 						authorities: new_authorities.to_vec(),
		// 					}
		// 				};
		// 				let old = ::std::mem::replace(guard.as_mut(), new_set);
		// 				guard.set_old(old);
		newAuthoritySet := newAuthoritySet[H, N]{
			CanonNumber: canonNumber,
			CanonHash:   *canonHash,
			SetID:       grandpa.SetID(setID),
			Authorities: newAuthorities,
		}
		old := guard.asMut().Clone()
		guard.setOld(old)
		*guard.asMut() = newSet

		// 				AppliedChanges::Forced(new_authorities)
		appliedChanges = importAppliedChangesForced[H, N](newAuthoritySet)
	} else {
		// 				let did_standard = guard
		// 					.as_mut()
		// 					.enacts_standard_change(hash, number, &is_descendent_of)
		// 					.map_err(|e| ConsensusError::ClientImport(e.to_string()))
		// 					.map_err(ConsensusError::from)?;

		// 				if let Some(root) = did_standard {
		// 					AppliedChanges::Standard(root)
		// 				} else {
		// 					AppliedChanges::None
		// 				}
		// 			}
		didStandard, err := guard.asMut().EnactsStandardChange(hash, number, isDescendentOf)
		if err != nil {
			return nil, err
		}

		if didStandard != nil {
			appliedChanges = importAppliedChangesStandard(*didStandard)
		} else {
			appliedChanges = importAppliedChangesNone{}
		}
	}

	// consume the guard safely and write necessary changes.
	// 		let just_in_case = guard.consume();
	justInCaseConsumed := guard.consume()
	// 		if let Some((_, ref authorities)) = just_in_case {
	if justInCaseConsumed != nil {
		authorities := &justInCaseConsumed.SharedDataLocked
		// 			let authorities_change = match applied_changes {
		// 				AppliedChanges::Forced(ref new) => Some(new),
		// 				AppliedChanges::Standard(_) => None, // the change isn't actually applied yet.
		// 				AppliedChanges::None => None,
		// 			};
		var authoritiesChange *newAuthoritySet[H, N]
		switch appliedChanges := appliedChanges.(type) {
		case importAppliedChangesForced[H, N]:
			newSet := newAuthoritySet[H, N](appliedChanges)
			authoritiesChange = &newSet
		case importAppliedChangesStandard:
			// the change isn't actually applied yet.
		case importAppliedChangesNone:
			// no change
		default:
			panic("unreachable")
		}
		// 			crate::aux_schema::update_authority_set::<Block, _, _>(
		// 				authorities,
		// 				authorities_change,
		// 				|insert| {
		// 					block
		// 						.auxiliary
		// 						.extend(insert.iter().map(|(k, v)| (k.to_vec(), Some(v.to_vec()))))
		// 				},
		// 			);
		updateAuthoritySet(authorities.Data(), authoritiesChange, func(insertions []api.KeyValue) error {
			converted := make([]api.AuxDataOperation, len(insertions))
			for i, kv := range insertions {
				converted[i] = api.AuxDataOperation{
					Key:  kv.Key,
					Data: kv.Value,
				}
			}
			block.Auxiliary = append(block.Auxiliary, converted...)
			return nil
		})
	}

	// 		let just_in_case = just_in_case.map(|(o, i)| (o, i.release_mutex()));
	var jic *justInCase[H, N]
	if justInCaseConsumed != nil {
		jic = &justInCase[H, N]{
			old:              justInCaseConsumed.old,
			SharedDataLocked: justInCaseConsumed.SharedDataLocked,
		}
	}

	// 		Ok(PendingSetChanges { just_in_case, applied_changes, do_pause })
	return &pendingSetChanges[H, N, Hasher, Header]{
		justInCase:     jic,
		appliedChanges: appliedChanges,
		doPause:        doPause,
	}, nil
}

// /// Read current set id form a given state.
// fn current_set_id(&self, hash: Block::Hash) -> Result<SetId, ConsensusError> {
func (gbi *GrandpaBlockImport[H, N, Hasher, Header, E]) currentSetID(hash H) (SetID, error) {
	// 		let runtime_version = self.inner.runtime_api().version(hash).map_err(|e| {
	// 			ConsensusError::ClientImport(format!(
	// 				"Unable to retrieve current runtime version. {}",
	// 				e
	// 			))
	// 		})?;
	runtimeVersion, err := gbi.inner.RuntimeAPI().Version(hash)
	if err != nil {
		return 0, err
	}

	var GrandpaID = [8]byte{237, 153, 197, 172, 178, 94, 237, 245}
	apiVersion := runtimeVersion.APIVersion(GrandpaID)

	// 		if runtime_version
	// 			.api_version(&<dyn GrandpaApi<Block>>::ID)
	// 			.map_or(false, |v| v < 3)
	// 		{
	if apiVersion != nil && *apiVersion < 3 {
		// The new API is not supported in this runtime. Try reading directly from storage.
		// This code may be removed once warp sync to an old runtime is no longer needed.
		// 			for prefix in ["GrandpaFinality", "Grandpa"] {
		for _, prefix := range []string{"GrandpaFinality", "Grandpa"} {
			// 				let k = [
			// 					sp_crypto_hashing::twox_128(prefix.as_bytes()),
			// 					sp_crypto_hashing::twox_128(b"CurrentSetId"),
			// 				]
			// 				.concat();
			k0 := hashing.Twox128([]byte(prefix))
			k1 := hashing.Twox128([]byte("CurrentSetId"))
			k := k0[:]
			k = append(k, k1[:]...)
			// 				if let Ok(Some(id)) =
			// 					self.inner.storage(hash, &sc_client_api::StorageKey(k.to_vec()))
			// 				{
			// 					if let Ok(id) = SetId::decode(&mut id.0.as_ref()) {
			// 						return Ok(id)
			// 					}
			// 				}
			id, err := gbi.inner.Storage(hash, runtime.StorageKey(k))
		}
		// 			Err(ConsensusError::ClientImport("Unable to retrieve current set id.".into()))
	} else {
		// 			self.inner
		// 				.runtime_api()
		// 				.current_set_id(hash)
		// 				.map_err(|e| ConsensusError::ClientImport(e.to_string()))
	}
	panic("unimpl")
}

// /// Import whole new state and reset authority set.
// async fn import_state(
//
//	&self,
//	mut block: BlockImportParams<Block>,
//
// ) -> Result<ImportResult, ConsensusError> {
func (gbi *GrandpaBlockImport[H, N, Hasher, Header, E]) importState(
	block *client_common.BlockImportParams[H, N, E, Header],
) (client_common.ImportResult, error) {
	// 		let hash = block.post_hash();
	// 		let number = *block.header.number();
	// 		// Force imported state finality.
	// 		block.finalized = true;
	// 		let import_result = (&*self.inner).import_block(block).await;
	// 		match import_result {
	// 			Ok(ImportResult::Imported(aux)) => {
	// 				// We've just imported a new state. We trust the sync module has verified
	// 				// finality proofs and that the state is correct and final.
	// 				// So we can read the authority list and set id from the state.
	// 				self.authority_set_hard_forks.lock().clear();
	// 				let authorities = self
	// 					.inner
	// 					.runtime_api()
	// 					.grandpa_authorities(hash)
	// 					.map_err(|e| ConsensusError::ClientImport(e.to_string()))?;
	// 				let set_id = self.current_set_id(hash)?;
	// 				let authority_set = AuthoritySet::new(
	// 					authorities.clone(),
	// 					set_id,
	// 					fork_tree::ForkTree::new(),
	// 					Vec::new(),
	// 					AuthoritySetChanges::empty(),
	// 				)
	// 				.ok_or_else(|| ConsensusError::ClientImport("Invalid authority list".into()))?;
	// 				*self.authority_set.inner_locked() = authority_set.clone();

	//				crate::aux_schema::update_authority_set::<Block, _, _>(
	//					&authority_set,
	//					None,
	//					|insert| self.inner.insert_aux(insert, []),
	//				)
	//				.map_err(|e| ConsensusError::ClientImport(e.to_string()))?;
	//				let new_set =
	//					NewAuthoritySet { canon_number: number, canon_hash: hash, set_id, authorities };
	//				let _ = self
	//					.send_voter_commands
	//					.unbounded_send(VoterCommand::ChangeAuthorities(new_set));
	//				Ok(ImportResult::Imported(aux))
	//			},
	//			Ok(r) => Ok(r),
	//			Err(e) => Err(ConsensusError::ClientImport(e.to_string())),
	//		}
	//	}
	panic("unimpl")
}
