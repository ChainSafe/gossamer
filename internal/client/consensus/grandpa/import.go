package grandpa

import (
	"fmt"
	"sync"

	"github.com/ChainSafe/gossamer/internal/client/api"
	"github.com/ChainSafe/gossamer/internal/client/api/utils"
	client_common "github.com/ChainSafe/gossamer/internal/client/consensus/common"
	shareddata "github.com/ChainSafe/gossamer/internal/client/consensus/common/shared-data"
	"github.com/ChainSafe/gossamer/internal/primitives/blockchain"
	"github.com/ChainSafe/gossamer/internal/primitives/consensus/common"
	"github.com/ChainSafe/gossamer/internal/primitives/consensus/grandpa"
	"github.com/ChainSafe/gossamer/internal/primitives/crypto/hashing"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/internal/primitives/storage"
	forktree "github.com/ChainSafe/gossamer/internal/utils/fork-tree"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

type importAppliedChanges interface {
	needsJustification() bool
}

type importAppliedChangesStandard bool // true if the change is ready to be applied (i.e. it's a root)

type importAppliedChangesForced[H runtime.Hash, N runtime.Number] newAuthoritySet[H, N]

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

type justInCase[H runtime.Hash, N runtime.Number] struct {
	old AuthoritySet[H, N]
	shareddata.SharedDataLocked[AuthoritySet[H, N]]
}

type pendingSetChanges[
	H runtime.Hash,
	N runtime.Number,
	Hasher runtime.Hasher[H],
	Header runtime.Header[N, H],
] struct {
	justInCase     *justInCase[H, N]
	appliedChanges importAppliedChanges
	doPause        bool
}

// revert the pending set change explicitly.
func (pendingSetChanges[H, N, Hasher, Header]) revert() {}

func (psc *pendingSetChanges[H, N, Hasher, Header]) defuse() (importAppliedChanges, bool) {
	psc.justInCase = nil
	appliedChanges := psc.appliedChanges
	psc.appliedChanges = importAppliedChangesNone{}
	return appliedChanges, psc.doPause
}

func (psc *pendingSetChanges[H, N, Hasher, Header]) drop() {
	if psc.justInCase != nil {
		jic := psc.justInCase
		psc.justInCase = nil
		oldSet := jic.old
		locked := jic.SharedDataLocked
		*locked.MutRef() = oldSet
		defer locked.Unlock()
	}
}

// A block-import handler for GRANDPA.
//
// This scans each imported block for signals of changing authority set.
// If the block being imported enacts an authority set change then:
// - If the current authority set is still live: we import the block
// - Otherwise, the block must include a valid justification.
//
// When using GRANDPA, the block import worker should be using this block import object.
type GrandpaBlockImport[
	H runtime.Hash,
	N runtime.Number,
	Hasher runtime.Hasher[H],
	Header runtime.Header[N, H],
	E runtime.Extrinsic,
] struct {
	inner                     ClientForGrandpa[H, N, Hasher, Header, E]
	justificationImportPeriod uint32
	selectChain               common.SelectChain[H, N, Header]
	authoritySet              *SharedAuthoritySet[H, N]
	sendVoterCommands         chan voterCommand
	authoritySetHardForks     map[H]PendingChange[H, N]
	authoritySetHardForksMtx  sync.Mutex
	justificationSender       GrandpaJustificationSender[H, N, Header]
	// TODO: telemetry
}

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
	var authoritySetHardForksMap = make(map[H]PendingChange[H, N])
	for _, hardFork := range authoritySetHardForks {
		authoritySetHardForksMap[hardFork.PendingChange.CanonHash] = hardFork.PendingChange
	}

	// check for and apply any forced authority set hard fork that apply to
	// any *pending* standard changes, checking by the block hash at which
	// they were announced.
	authoritySet, unlock := sharedAuthoritySet.inner.DataMut()

	authoritySet.PendingStandardChanges = forktree.Map(
		authoritySet.PendingStandardChanges,
		func(hash H, _ N, original PendingChange[H, N]) PendingChange[H, N] {
			if change, ok := authoritySetHardForksMap[hash]; ok {
				return change
			}
			return original
		},
	)
	unlock()

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

// Checks the given header for a consensus digest signalling a **standard** scheduled change and
// extracts it.
func FindScheduledChange[
	H runtime.Hash,
	N runtime.Number,
	Header runtime.Header[N, H],
](
	header Header,
) *grandpa.ScheduledChange[N] {
	id := runtime.OpaqueDigestItemIDConsensus(grandpa.GrandpaEngineID)

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

// Checks the given header for a consensus digest signalling a **forced** scheduled change and
// extracts it.
func FindForcedChange[
	H runtime.Hash,
	N runtime.Number,
	Header runtime.Header[N, H],
](
	header Header,
) *ForcedChange[H, N] {
	id := runtime.OpaqueDigestItemIDConsensus(grandpa.GrandpaEngineID)

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

	// find the first consensus digest with the right ID which converts to
	// the right kind of consensus log.
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

func (gbi *GrandpaBlockImport[H, N, Hasher, Header, E]) checkNewChange(
	header Header,
	hash H,
) *PendingChange[H, N] {
	// check for forced authority set hard forks
	gbi.authoritySetHardForksMtx.Lock()
	change, ok := gbi.authoritySetHardForks[hash]
	gbi.authoritySetHardForksMtx.Unlock()
	if ok {
		return &change
	}

	// check for forced change.
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
	scheduled := FindScheduledChange[H, N](header)
	return &PendingChange[H, N]{
		NextAuthorities: scheduled.NextAuthorities,
		Delay:           scheduled.Delay,
		CanonHeight:     header.Number(),
		CanonHash:       hash,
		DelayKind:       delayKindFinalized{},
	}
}

type innerGuard[H runtime.Hash, N runtime.Number] struct {
	old   *AuthoritySet[H, N]
	guard *shareddata.SharedDataLocked[AuthoritySet[H, N]]
}

func (ig *innerGuard[H, N]) asMut() *AuthoritySet[H, N] {
	if ig.guard == nil {
		panic("guard is nil; only taken on deconstruction; qed")
	}
	return ig.guard.MutRef()
}

func (ig *innerGuard[H, N]) setOld(old AuthoritySet[H, N]) {
	if ig.old == nil {
		// ignore "newer" old changes.
		ig.old = &old
	}
}

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

func (ig *innerGuard[H, N]) drop() {
	guard := ig.guard
	ig.guard = nil
	old := ig.old
	ig.old = nil
	if guard != nil && old != nil {
		*ig.guard.MutRef() = *ig.old
	}
}

func (gbi *GrandpaBlockImport[H, N, Hasher, Header, E]) makeAuthoritiesChanges(
	block *client_common.BlockImportParams[H, N, E, Header],
	hash H,
	initialSync bool,
) (*pendingSetChanges[H, N, Hasher, Header], error) {
	// when we update the authorities, we need to hold the lock
	// until the block is written to prevent a race if we need to restore
	// the old authority set on error or panic.
	number := block.Header.Number()
	maybeChange := gbi.checkNewChange(block.Header, hash)

	// returns a function for checking whether a block is a descendent of another
	// consistent with querying client directly after importing the block.
	parentHash := block.Header.ParentHash()
	isDescendentOf := utils.IsDescendantOf(gbi.inner, &utils.HashParent[H]{Hash: hash, Parent: parentHash})
	locked := gbi.authoritySet.inner.Locked()
	defer locked.Unlock()
	guard := innerGuard[H, N]{
		old:   nil,
		guard: &locked,
	}
	defer guard.drop()

	// whether to pause the old authority set -- happens after import
	// of a forced change block.
	var doPause bool

	// add any pending changes.
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

	var appliedChanges importAppliedChanges
	forcedChangeSet, err := guard.asMut().applyForcedChanges(
		hash,
		number,
		isDescendentOf,
		initialSync,
	)
	if err != nil {
		return nil, err
	}

	if forcedChangeSet != nil {
		medianLastFinalizedNumber := forcedChangeSet.median
		newSet := forcedChangeSet.set
		setID, newAuthorities := newSet.current()

		// we will use the median last finalized number as a hint
		// for the canon block the new authority set should start
		// with. we use the minimum between the median and the local
		// best finalized block.
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
			panic("the given block number is less or equal than the current best finalized number; " +
				"current best finalized number must exist in chain; qed.")
		}

		newAuthoritySet := newAuthoritySet[H, N]{
			CanonNumber: canonNumber,
			CanonHash:   *canonHash,
			SetID:       grandpa.SetID(setID),
			Authorities: newAuthorities,
		}
		old := guard.asMut().Clone()
		guard.setOld(old)
		*guard.asMut() = newSet

		appliedChanges = importAppliedChangesForced[H, N](newAuthoritySet)
	} else {
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
	justInCaseConsumed := guard.consume()
	if justInCaseConsumed != nil {
		authorities := &justInCaseConsumed.SharedDataLocked
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
		err := updateAuthoritySet(authorities.Data(), authoritiesChange, func(insertions []api.KeyValue) error {
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
		if err != nil {
			return nil, err
		}
	}

	var jic *justInCase[H, N]
	if justInCaseConsumed != nil {
		jic = &justInCase[H, N]{
			old:              justInCaseConsumed.old,
			SharedDataLocked: justInCaseConsumed.SharedDataLocked,
		}
	}

	return &pendingSetChanges[H, N, Hasher, Header]{
		justInCase:     jic,
		appliedChanges: appliedChanges,
		doPause:        doPause,
	}, nil
}

// Read current set id form a given state.
func (gbi *GrandpaBlockImport[H, N, Hasher, Header, E]) currentSetID(hash H) (grandpa.SetID, error) {
	runtimeVersion, err := gbi.inner.RuntimeAPI().Version(hash)
	if err != nil {
		return 0, err
	}

	var GrandpaID = [8]byte{237, 153, 197, 172, 178, 94, 237, 245}
	apiVersion := runtimeVersion.APIVersion(GrandpaID)

	if apiVersion != nil && *apiVersion < 3 {
		// The new API is not supported in this runtime. Try reading directly from storage.
		// This code may be removed once warp sync to an old runtime is no longer needed.
		for _, prefix := range []string{"GrandpaFinality", "Grandpa"} {
			k0 := hashing.Twox128([]byte(prefix))
			k1 := hashing.Twox128([]byte("CurrentSetId"))
			k := k0[:]
			k = append(k, k1[:]...)
			id, _ := gbi.inner.Storage(hash, storage.StorageKey(k))
			if id != nil {
				var setID grandpa.SetID
				err := scale.Unmarshal(*id, &setID)
				if err == nil {
					return setID, nil
				}
			}
		}
		return 0, fmt.Errorf("unable to retrieve current set id")
	} else {
		setID, err := gbi.inner.RuntimeAPI().CurrentSetID(hash)
		if err != nil {
			return 0, err
		}
		return setID, nil
	}
}

// Import whole new state and reset authority set.
func (gbi *GrandpaBlockImport[H, N, Hasher, Header, E]) importState(
	block *client_common.BlockImportParams[H, N, E, Header],
) (client_common.ImportResult, error) {
	hash := block.GetPostHash()
	number := block.Header.Number()

	// Force imported state finality.
	block.Finalized = true
	importResult, err := gbi.inner.ImportBlock(block)
	if err == nil {
		switch importResult := importResult.(type) {
		case client_common.ImportResultImported:
			aux := client_common.ImportedAux(importResult)
			// We've just imported a new state. We trust the sync module has verified
			// finality proofs and that the state is correct and final.
			// So we can read the authority list and set id from the state.
			gbi.authoritySetHardForksMtx.Lock()
			gbi.authoritySetHardForks = make(map[H]PendingChange[H, N])
			gbi.authoritySetHardForksMtx.Unlock()
			authorities, err := gbi.inner.RuntimeAPI().GrandpaAuthorities(hash)
			if err != nil {
				return nil, err
			}
			setID, err := gbi.currentSetID(hash)
			if err != nil {
				return nil, err
			}
			authoritySet, err := NewAuthoritySet[H, N](
				authorities,
				uint64(setID),
				forktree.NewForkTree[H, N, PendingChange[H, N]](),
				[]PendingChange[H, N]{},
				AuthoritySetChanges[N]{},
			)
			if err != nil {
				return nil, err
			}

			locked := gbi.authoritySet.inner.Locked()
			*locked.MutRef() = authoritySet.Clone()
			defer locked.Unlock()

			err = updateAuthoritySet(
				locked.Data(),
				nil,
				func(insertions []api.KeyValue) error {
					return gbi.inner.InsertAux(insertions, nil)
				},
			)
			if err != nil {
				return nil, err
			}
			newSet := newAuthoritySet[H, N]{
				CanonNumber: number,
				CanonHash:   hash,
				SetID:       setID,
				Authorities: authorities,
			}
			gbi.sendVoterCommands <- voterCommandChangeAuthorities[H, N](newSet)
			return client_common.ImportResultImported(aux), nil
		case client_common.ImportResultAlreadyInChain,
			client_common.ImportResultKnownBad,
			client_common.ImportResultMissingState,
			client_common.ImportResultUnknownParent:
			//			Ok(r) => Ok(r),
			return importResult, nil
		default:
			panic("unreachable")
		}

	} else {
		return nil, err
	}
}

func (gbi *GrandpaBlockImport[H, N, Hasher, Header, E]) ImportBlock(
	block *client_common.BlockImportParams[H, N, E, Header],
) (client_common.ImportResult, error) {
	hash := block.GetPostHash()
	number := block.Header.Number()

	// early exit if block already in chain, otherwise the check for
	// authority changes will error when trying to re-import a change block
	status, err := gbi.inner.Status(hash)
	if err != nil {
		return nil, err
	}
	if status == blockchain.BlockStatusInChain {
		// Strip justifications when re-importing an existing block.
		block.Justifications = nil
		return gbi.inner.ImportBlock(block)
	}

	if block.WithState() {
		return gbi.importState(block)
	}

	if number <= gbi.inner.Info().FinalizedNumber {
		// Importing an old block. Just save justifications and authority set changes
		if gbi.checkNewChange(block.Header, hash) != nil {
			if block.Justifications == nil {
				return nil, fmt.Errorf("justification required when importing an old block with authority set change")
			}
			locked := gbi.authoritySet.inner.Locked()
			authoritySet := locked.MutRef()
			authoritySet.AuthoritySetChanges.insert(number)
			err := updateAuthoritySet(
				*authoritySet,
				nil,
				func(insertions []api.KeyValue) error {
					converted := make([]api.AuxDataOperation, len(insertions))
					for i, kv := range insertions {
						converted[i] = api.AuxDataOperation{
							Key:  kv.Key,
							Data: kv.Value,
						}
					}
					block.Auxiliary = append(block.Auxiliary, converted...)
					return nil
				},
			)
			if err != nil {
				return nil, err
			}
		}
		return gbi.inner.ImportBlock(block)
	}

	// on initial sync we will restrict logging under info to avoid spam.
	initialSync := block.Origin == common.NetworkInitialSyncBlockOrigin

	pendingChanges, err := gbi.makeAuthoritiesChanges(block, hash, initialSync)
	if err != nil {
		return nil, err
	}
	defer pendingChanges.drop()

	// we don't want to finalize on inner.ImportBlock
	justifications := block.Justifications
	block.Justifications = nil
	importResult, err := gbi.inner.ImportBlock(block)

	if err != nil {
		logger.Debugf("Restoring old authority set after block import error: %s", err)
		pendingChanges.revert()
		return nil, err
	}
	var importedAux client_common.ImportedAux
	switch importResult := importResult.(type) {
	case client_common.ImportResultImported:
		importedAux = client_common.ImportedAux(importResult)
	default:
		logger.Debugf("Restoring old authority set after block import result: %v", importResult)
		pendingChanges.revert()
		return importResult, nil
	}

	appliedChanges, doPause := pendingChanges.defuse()

	// Send the pause signal after import but BEFORE sending a `ChangeAuthorities` message.
	if doPause {
		gbi.sendVoterCommands <- voterCommandPause("Forced change scheduled after inactivity")
	}

	needsJustification := appliedChanges.needsJustification()

	switch appliedChanges := appliedChanges.(type) {
	case importAppliedChangesForced[H, N]:
		// NOTE: when we do a force change we are "discrediting" the old set so we
		// ignore any justifications from them. this block may contain a justification
		// which should be checked and imported below against the new authority
		// triggered by this forced change. the new grandpa voter will start at the
		// last median finalized block (which is before the block that enacts the
		// change), full nodes syncing the chain will not be able to successfully
		// import justifications for those blocks since their local authority set view
		// is still of the set before the forced change was enacted, still after #1867
		// they should import the block and discard the justification, and they will
		// then request a justification from sync if it's necessary (which they should
		// then be able to successfully validate).
		gbi.sendVoterCommands <- voterCommandChangeAuthorities[H, N](newAuthoritySet[H, N](appliedChanges))
		// we must clear all pending justifications requests, presumably they won't be
		// finalized hence why this forced changes was triggered
		importedAux.ClearJustificationRequests = true

	case importAppliedChangesStandard:
		// this is a standard change, we don't apply it yet, but we will send a
		// we can't apply this change yet since there are other dependent changes that we
		// need to apply first, drop any justification that might have been provided with
		// the block to make sure we request them from `sync` which will ensure they'll be
		// applied in-order.
		justifications = nil
	default:
	}
	var grandpaJustification *runtime.EncodedJustification
	if justifications != nil {
		grandpaJustification = justifications.IntoJustification(grandpa.GrandpaEngineID)
	}

	if grandpaJustification != nil {
		if shouldProcessJustification(
			gbi.inner,
			gbi.justificationImportPeriod,
			number,
			needsJustification,
		) {
			err := gbi.importJustification(
				hash,
				number,
				runtime.Justification{
					ConsensusEngineID:    grandpa.GrandpaEngineID,
					EncodedJustification: *grandpaJustification,
				},
				needsJustification,
				initialSync,
			)

			if err != nil {
				if needsJustification {
					logger.Debugf(
						("Requesting justification from peers due to imported block #%d that enacts authority set" +
							"change with invalid justification: %s"), number, err)
					importedAux.BadJustification = true
					importedAux.NeedsJustification = true
				}
			}
		} else {
			logger.Debugf("Ignoring unnecessary justification for block #%d", number)
		}
	} else {
		if needsJustification {
			logger.Debugf(
				"Imported unjustified block #%d that enacts authority set change, waiting for finality for enactment.",
				number)
			importedAux.NeedsJustification = true
		}
	}

	return client_common.ImportResultImported(importedAux), nil
}

func (gbi *GrandpaBlockImport[H, N, Hasher, Header, E]) CheckBlock(
	block client_common.BlockCheckParams[H, N],
) (client_common.ImportResult, error) {
	return gbi.inner.CheckBlock(block)
}

// Import a block justification and finalize the block.
//
// If enactsChange is set to true, then finalizing this block *must*
// enact an authority set change, the function will panic otherwise.
func (gbi *GrandpaBlockImport[H, N, Hasher, Header, E]) importJustification(
	hash H,
	number N,
	justification runtime.Justification,
	enactsChange bool,
	initialSync bool,
) error {
	if justification.ConsensusEngineID != grandpa.GrandpaEngineID {
		// TODO: the import queue needs to be refactored to be able dispatch to the correct
		// JustificationImport instance based on ConsensusEngineID, or we need to build a
		// justification import pipeline similar to what we do for BlockImport. In the
		// meantime we'll just drop the justification, since this is only used for BEEFY which
		// is still WIP.
		return nil
	}

	just, err := DecodeGrandpaJustificationVerifyFinalizes[H, N, Hasher, Header](
		justification.EncodedJustification,
		HashNumber[H, N]{Hash: hash, Number: number},
		gbi.authoritySet.SetID(),
		gbi.authoritySet.CurrentAuthorities(),
	)

	if err != nil {
		return err
	}

	err = finalizeBlock(
		gbi.inner,
		gbi.authoritySet,
		nil,
		hash,
		number,
		justificationOrCommitJustification[H, N, Header]{just},
		initialSync,
		&gbi.justificationSender,
	)
	if err != nil {
		_, ok := err.(voterCommand)
		if ok {
			l := logger.Infof
			if initialSync {
				l = logger.Debugf
			}
			l("👴 Imported justification for block #%d that triggers command %s, signalling voter.", number, err)

			// send the command to the voter
			gbi.sendVoterCommands <- err.(voterCommand)
		} else {
			return err
		}
	} else {
		if enactsChange {
			panic("returns Ok when no authority set change should be enacted; qed;")
		}
	}

	return nil
}
