package prospectiveparachains

import (
	"bytes"
	"container/list"
	"fmt"
	"iter"
	"slices"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/tidwall/btree"
)

type candidateState byte

const (
	seconded candidateState = iota
	backed
)

// forkSelectionRule does a normal comparison between 2 candidate hashes
// and returns -1 if the first hash is lower than the second one meaning that
// the first hash will be chosen as the best candidate.
func forkSelectionRule(hash1, hash2 parachaintypes.CandidateHash) int {
	return bytes.Compare(hash1.Value[:], hash2.Value[:])
}

// candidateEntry represents a candidate in the candidateStorage
type candidateEntry struct {
	candidateHash      parachaintypes.CandidateHash
	parentHeadDataHash common.Hash
	outputHeadDataHash common.Hash
	relayParent        common.Hash
	candidate          *prospectiveCandidate
	state              candidateState
}

func newCandidateEntry(
	candidateHash parachaintypes.CandidateHash,
	candidate parachaintypes.CommittedCandidateReceipt,
	persistedValidationData parachaintypes.PersistedValidationData,
	state candidateState,
) (*candidateEntry, error) {
	pvdHash, err := persistedValidationData.Hash()
	if err != nil {
		return nil, fmt.Errorf("hashing persisted validation data: %w", err)
	}

	if pvdHash != candidate.Descriptor.PersistedValidationDataHash {
		return nil, errPersistedValidationDataMismatch
	}

	parentHeadDataHash, err := persistedValidationData.ParentHead.Hash()
	if err != nil {
		return nil, fmt.Errorf("hashing parent head data: %w", err)
	}

	outputHeadDataHash, err := candidate.Commitments.HeadData.Hash()
	if err != nil {
		return nil, fmt.Errorf("hashing output head data: %w", err)
	}

	if parentHeadDataHash == outputHeadDataHash {
		return nil, errZeroLengthCycle
	}

	return &candidateEntry{
		candidateHash:      candidateHash,
		parentHeadDataHash: parentHeadDataHash,
		outputHeadDataHash: outputHeadDataHash,
		relayParent:        candidate.Descriptor.RelayParent,
		state:              state,
		candidate: &prospectiveCandidate{
			Commitments:             candidate.Commitments,
			PersistedValidationData: persistedValidationData,
			PoVHash:                 candidate.Descriptor.PovHash,
			ValidationCodeHash:      candidate.Descriptor.ValidationCodeHash,
		},
	}, nil
}

// candidateStorage is an utility for storing candidates and information about them such as
// their relay-parents and their backing states. This does not assume any restriction on whether
// or not candidates form a chain. Useful for storing all kinds of candidates.
type candidateStorage struct {
	byParentHead    map[common.Hash]map[parachaintypes.CandidateHash]struct{}
	byOutputHead    map[common.Hash]map[parachaintypes.CandidateHash]struct{}
	byCandidateHash map[parachaintypes.CandidateHash]*candidateEntry
}

func (c *candidateStorage) clone() *candidateStorage {
	clone := newCandidateStorage()

	for parentHead, candidates := range c.byParentHead {
		clone.byParentHead[parentHead] = make(map[parachaintypes.CandidateHash]struct{})
		for candidateHash := range candidates {
			clone.byParentHead[parentHead][candidateHash] = struct{}{}
		}
	}

	for outputHead, candidates := range c.byOutputHead {
		clone.byOutputHead[outputHead] = make(map[parachaintypes.CandidateHash]struct{})
		for candidateHash := range candidates {
			clone.byOutputHead[outputHead][candidateHash] = struct{}{}
		}
	}

	for candidateHash, entry := range c.byCandidateHash {
		clone.byCandidateHash[candidateHash] = &candidateEntry{
			candidateHash:      entry.candidateHash,
			parentHeadDataHash: entry.parentHeadDataHash,
			outputHeadDataHash: entry.outputHeadDataHash,
			relayParent:        entry.relayParent,
			candidate:          entry.candidate,
			state:              entry.state,
		}
	}

	return clone
}

func newCandidateStorage() *candidateStorage {
	return &candidateStorage{
		byParentHead:    make(map[common.Hash]map[parachaintypes.CandidateHash]struct{}),
		byOutputHead:    make(map[common.Hash]map[parachaintypes.CandidateHash]struct{}),
		byCandidateHash: make(map[parachaintypes.CandidateHash]*candidateEntry),
	}
}

func (c *candidateStorage) addPendingAvailabilityCandidate(
	candidateHash parachaintypes.CandidateHash,
	candidate parachaintypes.CommittedCandidateReceipt,
	persistedValidationData parachaintypes.PersistedValidationData,
) error {
	entry, err := newCandidateEntry(candidateHash, candidate, persistedValidationData, backed)
	if err != nil {
		return err
	}

	if err := c.addCandidateEntry(entry); err != nil {
		return fmt.Errorf("adding candidate entry: %w", err)
	}

	return nil
}

// Len return the number of stored candidate
func (c *candidateStorage) len() int {
	return len(c.byCandidateHash)
}

// addCandidateEntry inserts a new entry in the storage map, where the candidate hash
// is the key and the *candidateEntry is the value, also it create other links, the
// parent head hash points to the candidate hash also the output head hash points to the
// candidate hash
func (c *candidateStorage) addCandidateEntry(candidate *candidateEntry) error {
	_, ok := c.byCandidateHash[candidate.candidateHash]
	if ok {
		return errCandidateAlreadyKnown
	}

	// updates the reference parent hash -> candidate
	// we don't check the `ok` value since the key can
	// exists in the map but pointing to a nil map
	setOfCandidates := c.byParentHead[candidate.parentHeadDataHash]
	if setOfCandidates == nil {
		setOfCandidates = make(map[parachaintypes.CandidateHash]struct{})
	}
	setOfCandidates[candidate.candidateHash] = struct{}{}
	c.byParentHead[candidate.parentHeadDataHash] = setOfCandidates

	// udpates the reference output hash -> candidate
	setOfCandidates = c.byOutputHead[candidate.outputHeadDataHash]
	if setOfCandidates == nil {
		setOfCandidates = make(map[parachaintypes.CandidateHash]struct{})
	}
	setOfCandidates[candidate.candidateHash] = struct{}{}
	c.byOutputHead[candidate.outputHeadDataHash] = setOfCandidates

	c.byCandidateHash[candidate.candidateHash] = candidate
	return nil
}

// removeCandidate removes the candidate entry from the storage based on candidateHash
// it also removes the parent head hash entry that points to candidateHash and
// removes the output head hash entry that points to candidateHash
func (c *candidateStorage) removeCandidate(candidateHash parachaintypes.CandidateHash) {
	entry, ok := c.byCandidateHash[candidateHash]
	if !ok {
		return
	}

	delete(c.byCandidateHash, candidateHash)

	if setOfCandidates, ok := c.byParentHead[entry.parentHeadDataHash]; ok {
		delete(setOfCandidates, candidateHash)
		if len(setOfCandidates) == 0 {
			delete(c.byParentHead, entry.parentHeadDataHash)
		}
	}

	if setOfCandidates, ok := c.byOutputHead[entry.outputHeadDataHash]; ok {
		delete(setOfCandidates, candidateHash)
		if len(setOfCandidates) == 0 {
			delete(c.byOutputHead, entry.outputHeadDataHash)
		}
	}
}

func (c *candidateStorage) markBacked(candidateHash parachaintypes.CandidateHash) {
	entry, ok := c.byCandidateHash[candidateHash]
	if !ok {
		logger.Tracef("candidate not found while marking as backed")
	}

	entry.state = backed
}

func (c *candidateStorage) headDataByHash(hash common.Hash) *parachaintypes.HeadData {
	// first, search for candidates outputting this head data and extract the head data
	// from their commitments if they exist.
	// otherwise, search for candidates building upon this head data and extract the
	// head data from their persisted validation data if they exist.

	if setOfCandidateHashes, ok := c.byOutputHead[hash]; ok {
		for candidateHash := range setOfCandidateHashes {
			if candidate, ok := c.byCandidateHash[candidateHash]; ok {
				return &candidate.candidate.Commitments.HeadData
			}
		}
	}

	if setOfCandidateHashes, ok := c.byParentHead[hash]; ok {
		for candidateHash := range setOfCandidateHashes {
			if candidate, ok := c.byCandidateHash[candidateHash]; ok {
				return &candidate.candidate.PersistedValidationData.ParentHead
			}
		}
	}

	return nil
}

func (c *candidateStorage) possibleBackedParaChildren(parentHeadHash common.Hash) iter.Seq[*candidateEntry] {
	return func(yield func(*candidateEntry) bool) {
		seqOfCandidateHashes, ok := c.byParentHead[parentHeadHash]
		if !ok {
			return
		}

		for candidateHash := range seqOfCandidateHashes {
			if entry, ok := c.byCandidateHash[candidateHash]; ok && entry.state == backed {
				if !yield(entry) {
					return
				}
			}
		}
	}
}

// pendingAvailability is a candidate on-chain but pending availability, for special
// treatment in the `scope`
type pendingAvailability struct {
	candidateHash parachaintypes.CandidateHash
	relayParent   relayChainBlockInfo
}

// The scope of a fragment chain
type scope struct {
	// the relay parent we're currently building on top of
	relayParent relayChainBlockInfo
	// the other relay parents candidates are allowed to build upon,
	// mapped by the block number
	ancestors *btree.Map[parachaintypes.BlockNumber, relayChainBlockInfo]
	// the other relay parents candidates are allowed to build upon,
	// mapped by hash
	ancestorsByHash map[common.Hash]relayChainBlockInfo
	// candidates pending availability at this block
	pendingAvailability []*pendingAvailability
	// the base constraints derived from the latest included candidate
	baseConstraints *parachaintypes.Constraints
	// equal to `max_candidate_depth`
	maxDepth uint
}

// newScopeWithAncestors defines a new scope, all arguments are straightforward
// except ancestors. Ancestor should be in reverse order, starting with the parent
// of the relayParent, and proceeding backwards in block number decrements of 1.
// Ancestors not following these conditions will be rejected.
//
// This function will only consume ancestors up to the `MinRelayParentNumber` of the
// `baseConstraints`.
//
// Only ancestor whose children have the same session id as the relay parent's children
// should be provided. It is allowed to provide 0 ancestors.
func newScopeWithAncestors(
	relayParent relayChainBlockInfo,
	baseConstraints *parachaintypes.Constraints,
	pendingAvailability []*pendingAvailability,
	maxDepth uint,
	ancestors []relayChainBlockInfo,
) (*scope, error) {
	ancestorsMap := btree.NewMap[parachaintypes.BlockNumber, relayChainBlockInfo](100)
	ancestorsByHash := make(map[common.Hash]relayChainBlockInfo)

	prev := relayParent.Number
	for _, ancestor := range ancestors {
		if prev == 0 {
			return nil, errUnexpectedAncestor{number: ancestor.Number, prev: prev}
		}

		if ancestor.Number != prev-1 {
			return nil, errUnexpectedAncestor{number: ancestor.Number, prev: prev}
		}

		if prev == baseConstraints.MinRelayParentNumber {
			break
		}

		prev = ancestor.Number
		ancestorsByHash[ancestor.Hash] = ancestor
		ancestorsMap.Set(ancestor.Number, ancestor)
	}

	return &scope{
		relayParent:         relayParent,
		baseConstraints:     baseConstraints,
		pendingAvailability: pendingAvailability,
		maxDepth:            maxDepth,
		ancestors:           ancestorsMap,
		ancestorsByHash:     ancestorsByHash,
	}, nil
}

// earliestRelayParent gets the earliest relay-parent allowed in the scope of the fragment chain.
func (s *scope) earliestRelayParent() relayChainBlockInfo {
	if iter := s.ancestors.Iter(); iter.Next() {
		return iter.Value()
	}
	return s.relayParent
}

// Ancestor gets the relay ancestor of the fragment chain by hash.
func (s *scope) ancestor(hash common.Hash) *relayChainBlockInfo {
	if hash == s.relayParent.Hash {
		return &s.relayParent
	}

	if blockInfo, ok := s.ancestorsByHash[hash]; ok {
		return &blockInfo
	}

	return nil
}

// Whether the candidate in question is one pending availability in this scope.
func (s *scope) getPendingAvailability(candidateHash parachaintypes.CandidateHash) *pendingAvailability {
	for _, c := range s.pendingAvailability {
		if c.candidateHash == candidateHash {
			return c
		}
	}
	return nil
}

// Fragment node is a node that belongs to a `BackedChain`. It holds constraints based on
// the ancestors in the chain
type fragmentNode struct {
	fragment                *Fragment
	candidateHash           parachaintypes.CandidateHash
	cumulativeModifications *constraintModifications
	parentHeadDataHash      common.Hash
	outputHeadDataHash      common.Hash
}

func (f *fragmentNode) relayParent() common.Hash {
	return f.fragment.RelayParent().Hash
}

// newCandidateEntryFromFragment creates a candidate entry from a fragment, we dont need
// to perform the checks done in `newCandidateEntry` since a `fragmentNode` always comes
// from a `candidateEntry`
func newCandidateEntryFromFragment(node *fragmentNode) *candidateEntry {
	return &candidateEntry{
		candidateHash:      node.candidateHash,
		parentHeadDataHash: node.parentHeadDataHash,
		outputHeadDataHash: node.outputHeadDataHash,
		candidate:          node.fragment.Candidate(),
		relayParent:        node.relayParent(),
		// a fragment node is always backed
		state: backed,
	}
}

// backedChain is a chain of backed/backable candidates
// Includes candidates pending availability and candidates which may be backed on-chain
type backedChain struct {
	// holds the candidate chain
	chain []*fragmentNode

	// index from parent head data to the candidate that has that head data as parent
	// only contains the candidates present in the `chain`
	byParentHead map[common.Hash]parachaintypes.CandidateHash

	// index from head data hash to the candidate hash outputting that head data
	// only contains the candidates present in the `chain`
	byOutputHead map[common.Hash]parachaintypes.CandidateHash

	// a set of candidate hashes in the `chain`
	candidates map[parachaintypes.CandidateHash]struct{}
}

func newBackedChain() *backedChain {
	return &backedChain{
		chain:        make([]*fragmentNode, 0),
		byParentHead: make(map[common.Hash]parachaintypes.CandidateHash),
		byOutputHead: make(map[common.Hash]parachaintypes.CandidateHash),
		candidates:   make(map[parachaintypes.CandidateHash]struct{}),
	}
}

func (bc *backedChain) push(candidate *fragmentNode) {
	bc.candidates[candidate.candidateHash] = struct{}{}
	bc.byParentHead[candidate.parentHeadDataHash] = candidate.candidateHash
	bc.byOutputHead[candidate.outputHeadDataHash] = candidate.candidateHash
	bc.chain = append(bc.chain, candidate)
}

func (bc *backedChain) clear() []*fragmentNode {
	bc.byParentHead = make(map[common.Hash]parachaintypes.CandidateHash)
	bc.byOutputHead = make(map[common.Hash]parachaintypes.CandidateHash)
	bc.candidates = make(map[parachaintypes.CandidateHash]struct{})

	oldChain := bc.chain
	bc.chain = nil
	return oldChain
}

func (bc *backedChain) revertToParentHash(parentHeadDataHash common.Hash) []*fragmentNode {
	foundIndex := -1

	for i := 0; i < len(bc.chain); i++ {
		node := bc.chain[i]

		if foundIndex != -1 {
			delete(bc.byParentHead, node.parentHeadDataHash)
			delete(bc.byOutputHead, node.outputHeadDataHash)
			delete(bc.candidates, node.candidateHash)
		} else if node.outputHeadDataHash == parentHeadDataHash {
			foundIndex = i
		}
	}

	if foundIndex != -1 {
		// drain the elements from the found index until
		// the end of the slice and return them
		removed := make([]*fragmentNode, len(bc.chain)-(foundIndex+1))
		copy(removed, bc.chain[foundIndex+1:])
		bc.chain = slices.Delete(bc.chain, foundIndex+1, len(bc.chain))

		return removed
	}

	return nil
}

// this is a fragment chain specific to an active leaf. It holds the current
// best backable candidate chain, as well as potential candidates which could
// become connected to the chain in the future or which could even overwrite
// the existing chain
type fragmentChain struct {
	// the current scope, which dictates the on-chain operating constraints that
	// all future candidates must ad-here to.
	scope *scope

	// the current best chain of backable candidates. It only contains candidates
	// which build on top of each other and which have reached the backing quorum.
	// In the presence of potential forks, this chain will pick a fork according to
	// the `forkSelectionRule`
	bestChain *backedChain

	// the potential candidate storage. Contains candidates which are not yet part of
	// the `chain` but may become in the future. These can form any tree shape as well
	// as contain unconnected candidates for which we don't know the parent.
	unconnected *candidateStorage
}

// newFragmentChain createa a new fragment chain with the given scope and populates it with
// the candidates pending availability
func newFragmentChain(scope *scope, candidatesPendingAvailability *candidateStorage) *fragmentChain {
	fragmentChain := &fragmentChain{
		scope:       scope,
		bestChain:   newBackedChain(),
		unconnected: newCandidateStorage(),
	}

	// we only need to populate the best backable chain. Candidates pending availability
	// must form a chain with the latest included head.
	fragmentChain.populateChain(candidatesPendingAvailability)
	return fragmentChain
}

// populateFromPrevious populates the `fragmentChain` given the new candidates pending
// availability and the optional previous fragment chain (of the previous relay parent)
func (f *fragmentChain) populateFromPrevious(prevFragmentChain *fragmentChain) {
	prevStorage := prevFragmentChain.unconnected.clone()
	for _, candidate := range prevFragmentChain.bestChain.chain {
		// if they used to be pending availability, dont add them. This is fine because:
		// - if they still are pending availability, they have already been added to
		// the new storage
		// - if they were included, no point in keeping them
		//
		// This cannot happen for the candidates in the unconnected storage. The pending
		// availability candidates will always be part of the best chain
		pending := prevFragmentChain.scope.getPendingAvailability(candidate.candidateHash)
		if pending == nil {
			_ = prevStorage.addCandidateEntry(newCandidateEntryFromFragment(candidate))
		}
	}

	// first populate the best backable chain
	f.populateChain(prevStorage)

	// now that we picked the best backable chain, trim the forks generated by candidates
	// which are not present in the best chain
	f.trimUneligibleForks(prevStorage, nil)

	// finally, keep any candidates which haven't been trimmed but still have potential
	f.populateUnconnectedPotentialCandidates(prevStorage)
}

func (f *fragmentChain) bestChainLen() int {
	return len(f.bestChain.chain)
}

func (f *fragmentChain) containsUnconnectedCandidate(candidateHash parachaintypes.CandidateHash) bool { //nolint:unused
	_, ok := f.unconnected.byCandidateHash[candidateHash]
	return ok
}

// bestChainVec returns a vector of the chain's candidate hashes, in-order.
func (f *fragmentChain) bestChainVec() (hashes []parachaintypes.CandidateHash) {
	hashes = make([]parachaintypes.CandidateHash, len(f.bestChain.chain))
	for idx, node := range f.bestChain.chain {
		hashes[idx] = node.candidateHash
	}
	return hashes
}

func (f *fragmentChain) isCandidateBacked(hash parachaintypes.CandidateHash) bool { //nolint:unused
	if _, ok := f.bestChain.candidates[hash]; ok {
		return true
	}

	candidate := f.unconnected.byCandidateHash[hash]
	return candidate != nil && candidate.state == backed
}

// candidateBacked marks a candidate as backed. This can trigger a recreation of the best backable chain.
func (f *fragmentChain) candidateBacked(newlyBackedCandidate parachaintypes.CandidateHash) {
	// already backed
	if _, ok := f.bestChain.candidates[newlyBackedCandidate]; ok {
		return
	}

	candidateEntry, ok := f.unconnected.byCandidateHash[newlyBackedCandidate]
	if !ok {
		// candidate is not in unconnected storage
		return
	}

	parentHeadDataHash := candidateEntry.parentHeadDataHash

	f.unconnected.markBacked(newlyBackedCandidate)

	if !f.revertTo(parentHeadDataHash) {
		// if nothing was reverted, there is nothing we can do for now
		return
	}

	prevStorage := f.unconnected.clone()
	f.unconnected = newCandidateStorage()

	f.populateChain(prevStorage)
	f.trimUneligibleForks(prevStorage, &parentHeadDataHash)
	f.populateUnconnectedPotentialCandidates(prevStorage)
}

// canAddCandidateAsPotential checks if this candidate could be added in the future
func (f *fragmentChain) canAddCandidateAsPotential(entry *candidateEntry) error {
	candidateHash := entry.candidateHash

	_, existsInCandidateStorage := f.unconnected.byCandidateHash[candidateHash]
	_, existsInBestChain := f.bestChain.candidates[candidateHash]
	if existsInBestChain || existsInCandidateStorage {
		return errCandidateAlreadyKnown
	}

	return f.checkPotential(entry)
}

// tryAddingSecondedCandidate tries to add a candidate as a seconded candidate, if the
// candidate has potential. It will never be added to the chain directly in the seconded
// state, it will only be part of the unconnected storage
func (f *fragmentChain) tryAddingSecondedCandidate(entry *candidateEntry) error { //nolint:unused
	if entry.state == backed {
		return errIntroduceBackedCandidate
	}

	err := f.canAddCandidateAsPotential(entry)
	if err != nil {
		return err
	}

	return f.unconnected.addCandidateEntry(entry)
}

// getHeadDataByHash tries to get the full head data associated with this hash
func (f *fragmentChain) getHeadDataByHash(headDataHash common.Hash) (*parachaintypes.HeadData, error) { //nolint:unused
	reqParent := f.scope.baseConstraints.RequiredParent
	reqParentHash, err := reqParent.Hash()
	if err != nil {
		return nil, fmt.Errorf("while hashing required parent: %w", err)
	}
	if reqParentHash == headDataHash {
		return &reqParent, nil
	}

	hasHeadDataInChain := false
	if _, ok := f.bestChain.byParentHead[headDataHash]; ok {
		hasHeadDataInChain = true
	} else if _, ok := f.bestChain.byOutputHead[headDataHash]; ok {
		hasHeadDataInChain = true
	}

	if hasHeadDataInChain {
		for _, candidate := range f.bestChain.chain {
			if candidate.parentHeadDataHash == headDataHash {
				headData := candidate.
					fragment.
					Candidate().
					PersistedValidationData.
					ParentHead
				return &headData, nil
			} else if candidate.outputHeadDataHash == headDataHash {
				headData := candidate.fragment.Candidate().Commitments.HeadData
				return &headData, nil
			} else {
				continue
			}
		}
	}

	return f.unconnected.headDataByHash(headDataHash), nil
}

type candidateAndRelayParent struct {
	candidateHash   parachaintypes.CandidateHash
	realyParentHash common.Hash
}

// findBackableChain selects `count` candidates after the given `ancestors` which
// can be backed on chain next. The intention of the `ancestors` is to allow queries
// on the basis of one or more candidates which were previously pending availability
// becoming available or candidates timing out
func (f *fragmentChain) findBackableChain(
	ancestors map[parachaintypes.CandidateHash]struct{}, count uint32) []*candidateAndRelayParent {
	if count == 0 {
		return nil
	}

	basePos := f.findAncestorPath(ancestors)

	actualEndIdx := min(basePos+int(count), len(f.bestChain.chain))
	res := make([]*candidateAndRelayParent, 0, actualEndIdx-basePos)

	for _, elem := range f.bestChain.chain[basePos:actualEndIdx] {
		// only supply candidates which are not yet pending availability.
		// `ancestors` should have already contained them, but check just in case
		if pending := f.scope.getPendingAvailability(elem.candidateHash); pending == nil {
			res = append(res, &candidateAndRelayParent{
				candidateHash:   elem.candidateHash,
				realyParentHash: elem.relayParent(),
			})
		} else {
			break
		}
	}

	return res
}

// findAncestorPath tries to orders the ancestors into a viable path from root to the last one.
// stops when the ancestors are all used or when a node in the chain is not present in the
// ancestors set. Returns the index in the chain were the search stopped
func (f *fragmentChain) findAncestorPath(ancestors map[parachaintypes.CandidateHash]struct{}) int {
	if len(f.bestChain.chain) == 0 {
		return 0
	}

	for idx, candidate := range f.bestChain.chain {
		_, ok := ancestors[candidate.candidateHash]
		if !ok {
			return idx
		}
		delete(ancestors, candidate.candidateHash)
	}

	// this means that we found the entire chain in the ancestor set. There wont be
	// anything left to back.
	return len(f.bestChain.chain)
}

// earliestRelayParent returns the earliest relay parent a new candidate can have in order
// to be added to the chain right now. This is the relay parent of the latest candidate in
// the chain. The value returned may not be valid if we want to add a candidate pending
// availability, which may have a relay parent which is out of scope, special handling
// is needed in that case.
func (f *fragmentChain) earliestRelayParent() *relayChainBlockInfo {
	if len(f.bestChain.chain) > 0 {
		lastCandidate := f.bestChain.chain[len(f.bestChain.chain)-1]
		info := f.scope.ancestor(lastCandidate.relayParent())
		if info != nil {
			return info
		}

		// if the relay parent is out of scope AND it is in the chain
		// it must be a candidate pending availability
		pending := f.scope.getPendingAvailability(lastCandidate.candidateHash)
		if pending == nil {
			return nil
		}

		return &pending.relayParent
	}

	earliest := f.scope.earliestRelayParent()
	return &earliest
}

// earliestRelayParentPendingAvailability returns the earliest relay parent a potential
// candidate may have for it to ever be added to the chain. This is the relay parent of
// the last candidate pending availability or the earliest relay parent in scope.
func (f *fragmentChain) earliestRelayParentPendingAvailability() *relayChainBlockInfo {
	for i := len(f.bestChain.chain) - 1; i >= 0; i-- {
		candidate := f.bestChain.chain[i]
		if pending := f.scope.getPendingAvailability(candidate.candidateHash); pending != nil {
			return &pending.relayParent
		}
	}
	earliest := f.scope.earliestRelayParent()
	return &earliest
}

// populateUnconnectedPotentialCandidates populates the unconnected potential candidate storage
// starting from a previous storage
func (f *fragmentChain) populateUnconnectedPotentialCandidates(oldStorage *candidateStorage) {
	for _, candidate := range oldStorage.byCandidateHash {
		// sanity check, all pending availability candidates should be already present
		// in the chain
		if pending := f.scope.getPendingAvailability(candidate.candidateHash); pending != nil {
			continue
		}

		// we can just use the error to check if we can add
		// or not an entry since an error can legitimately
		// happen when pruning stale candidates.
		err := f.canAddCandidateAsPotential(candidate)
		if err == nil {
			_ = f.unconnected.addCandidateEntry(candidate)
		}
	}
}

func (f *fragmentChain) checkPotential(candidate *candidateEntry) error {
	relayParent := candidate.relayParent
	parentHeadHash := candidate.parentHeadDataHash

	// trivial 0-length cycle
	if candidate.outputHeadDataHash == parentHeadHash {
		return errZeroLengthCycle
	}

	// Check if the relay parent is in scope
	relayParentInfo := f.scope.ancestor(relayParent)
	if relayParentInfo == nil {
		return errRelayParentNotInScope{
			relayParentA: relayParent,
			relayParentB: f.scope.earliestRelayParent().Hash,
		}
	}

	// Check if the relay parent moved backwards from the latest candidate pending availability
	earliestRPOfPendingAvailability := f.earliestRelayParentPendingAvailability()
	if relayParentInfo.Number < earliestRPOfPendingAvailability.Number {
		return errRelayParentPrecedesCandidatePendingAvailability{
			relayParentA: relayParentInfo.Hash,
			relayParentB: earliestRPOfPendingAvailability.Hash,
		}
	}

	// If it's a fork with a backed candidate in the current chain
	if otherCandidateHash, ok := f.bestChain.byParentHead[parentHeadHash]; ok {
		if f.scope.getPendingAvailability(otherCandidateHash) != nil {
			// Cannot accept a fork with a candidate pending availability
			return errForkWithCandidatePendingAvailability{candidateHash: otherCandidateHash}
		}

		// If the candidate is backed and in the current chain, accept only a candidate
		// according to the fork selection rule
		if forkSelectionRule(otherCandidateHash, candidate.candidateHash) == -1 {
			return errForkChoiceRule{candidateHash: otherCandidateHash}
		}
	}

	// Try seeing if the parent candidate is in the current chain or if it is the latest
	// included candidate. If so, get the constraints the candidate must satisfy
	var constraints *parachaintypes.Constraints
	var maybeMinRelayParentNumber *parachaintypes.BlockNumber

	requiredParentHash, err := f.scope.baseConstraints.RequiredParent.Hash()
	if err != nil {
		return fmt.Errorf("while hashing required parent: %w", err)
	}

	if parentCandidateHash, ok := f.bestChain.byOutputHead[parentHeadHash]; ok {
		var parentCandidate *fragmentNode

		for _, c := range f.bestChain.chain {
			if c.candidateHash == parentCandidateHash {
				parentCandidate = c
				break
			}
		}

		if parentCandidate == nil {
			return errParentCandidateNotFound
		}

		var err error
		constraints, err = applyModifications(
			f.scope.baseConstraints,
			parentCandidate.cumulativeModifications)
		if err != nil {
			return errComputeConstraints{modificationErr: err}
		}

		if ancestor := f.scope.ancestor(parentCandidate.relayParent()); ancestor != nil {
			maybeMinRelayParentNumber = &ancestor.Number
		}
	} else if requiredParentHash == parentHeadHash {
		// It builds on the latest included candidate
		constraints = f.scope.baseConstraints.Clone()
	} else {
		// If the parent is not yet part of the chain, there's nothing else we can check for now
		return nil
	}

	// Check for cycles or invalid tree transitions
	if err := f.checkCyclesOrInvalidTree(candidate.outputHeadDataHash); err != nil {
		return err
	}

	// Check against constraints if we have a full concrete candidate
	_, err = checkAgainstConstraints(
		relayParentInfo,
		constraints,
		candidate.candidate.Commitments,
		candidate.candidate.ValidationCodeHash,
		candidate.candidate.PersistedValidationData,
	)
	if err != nil {
		return errCheckAgainstConstraints{fragmentValidityErr: err}
	}

	if relayParentInfo.Number < constraints.MinRelayParentNumber {
		return errRelayParentMovedBackwards
	}

	if maybeMinRelayParentNumber != nil && relayParentInfo.Number < *maybeMinRelayParentNumber {
		return errRelayParentMovedBackwards
	}

	return nil
}

// trimUneligibleForks once the backable chain was populated, trim the forks generated by candidate
// hashes which are not present in the best chain. Fan this out into a full breadth-first search. If
// starting point is not nil then start the search from the candidates having this parent head hash.
func (f *fragmentChain) trimUneligibleForks(storage *candidateStorage, startingPoint *common.Hash) {
	type queueItem struct {
		hash         common.Hash
		hasPotential bool
	}

	queue := list.New()

	// start out with the candidates in the chain. They are all valid candidates.
	if startingPoint != nil {
		queue.PushBack(queueItem{hash: *startingPoint, hasPotential: true})
	} else {
		if len(f.bestChain.chain) == 0 {
			reqParentHeadHash, err := f.scope.baseConstraints.RequiredParent.Hash()
			if err != nil {
				panic(fmt.Sprintf("while hashing required parent: %s", err.Error()))
			}

			queue.PushBack(queueItem{hash: reqParentHeadHash, hasPotential: true})
		} else {
			for _, candidate := range f.bestChain.chain {
				queue.PushBack(queueItem{hash: candidate.parentHeadDataHash, hasPotential: true})
			}
		}
	}

	// to make sure that cycles dont make us loop forever, keep track
	// of the visited parent head hashes
	visited := map[common.Hash]struct{}{}

	for queue.Len() > 0 {
		// queue.PopFront()
		parent := queue.Remove(queue.Front()).(queueItem)
		visited[parent.hash] = struct{}{}

		children, ok := storage.byParentHead[parent.hash]
		if !ok {
			continue
		}

		// cannot remove while iterating so store them here temporarily
		var toRemove []parachaintypes.CandidateHash

		for childHash := range children {
			child, ok := storage.byCandidateHash[childHash]
			if !ok {
				continue
			}

			// already visited this child. either is a cycle or multipath that lead
			// to the same candidate. either way, stop this branch to avoid looping
			// forever
			if _, ok = visited[child.outputHeadDataHash]; ok {
				continue
			}

			// only keep a candidate if its full ancestry was already kept as potential
			// and this candidate itself has potential
			if parent.hasPotential && f.checkPotential(child) == nil {
				queue.PushBack(queueItem{hash: child.outputHeadDataHash, hasPotential: true})
			} else {
				// otherwise, remove this candidate and continue looping for its children
				// but mark the parent's potential as false. we only want to remove its children.
				toRemove = append(toRemove, childHash)
				queue.PushBack(queueItem{hash: child.outputHeadDataHash, hasPotential: false})
			}
		}

		for _, hash := range toRemove {
			storage.removeCandidate(hash)
		}
	}
}

type possibleChild struct {
	fragment           *Fragment
	candidateHash      parachaintypes.CandidateHash
	outputHeadDataHash common.Hash
	parentHeadDataHash common.Hash
}

// populateChain populates the fragment chain with candidates from the supplied `candidateStorage`.
// Can be called by the `newFragmentChain` or when backing a new candidate. When this is called
// it may cause the previous chain to be completely erased or it may add more than one candidate
func (f *fragmentChain) populateChain(storage *candidateStorage) {
	var cumulativeModifications *constraintModifications
	if len(f.bestChain.chain) > 0 {
		lastCandidate := f.bestChain.chain[len(f.bestChain.chain)-1]
		cumulativeModifications = lastCandidate.cumulativeModifications.Clone()
	} else {
		cumulativeModifications = NewConstraintModificationsIdentity()
	}

	earliestRelayParent := f.earliestRelayParent()
	if earliestRelayParent == nil {
		return
	}

	for len(f.bestChain.chain) < int(f.scope.maxDepth)+1 {
		childConstraints, err := applyModifications(
			f.scope.baseConstraints, cumulativeModifications)
		if err != nil {
			logger.Warnf("failed to apply modifications: %s", err.Error())
			break
		}

		requiredHeadHash, err := childConstraints.RequiredParent.Hash()
		if err != nil {
			panic(fmt.Sprintf("failed while hashing required parent: %s", err.Error()))
		}

		possibleChildren := make([]*possibleChild, 0)
		// select the few possible backed/backable children which can be added to the chain right now
		for candidateEntry := range storage.possibleBackedParaChildren(requiredHeadHash) {
			// only select a candidate if:
			// 1. it does not introduce a fork or a cycle
			// 2. parent hash is correct
			// 3. relay parent does not move backwards
			// 4. all non-pending-availability candidates have relay-parent in the scope
			// 5. candidate outputs fulfil constraints

			var relayParent *relayChainBlockInfo
			var minRelayParent parachaintypes.BlockNumber

			pending := f.scope.getPendingAvailability(candidateEntry.candidateHash)
			if pending != nil {
				relayParent = &pending.relayParent
				if len(f.bestChain.chain) == 0 {
					minRelayParent = pending.relayParent.Number
				} else {
					minRelayParent = earliestRelayParent.Number
				}
			} else {
				info := f.scope.ancestor(candidateEntry.relayParent)
				if info == nil {
					continue
				}

				relayParent = info
				minRelayParent = earliestRelayParent.Number
			}

			if err := f.checkCyclesOrInvalidTree(candidateEntry.outputHeadDataHash); err != nil {
				logger.Warnf("failed while checking cycle or invalid tree: %s", err.Error())
				continue
			}

			// require: candidates dont move backwards and only pending availability
			// candidates can be out-of-scope.
			//
			// earliest relay parent can be before the
			if relayParent.Number < minRelayParent {
				// relay parent moved backwards
				continue
			}

			// don't add candidates if they're already present in the chain
			// this can never happen, as candidates can only be duplicated
			// if there's a cycle and we shouldnt have allowed for a cycle
			// to be chained
			if _, ok := f.bestChain.candidates[candidateEntry.candidateHash]; ok {
				continue
			}

			constraints := childConstraints.Clone()
			if pending != nil {
				// overwrite for candidates pending availability as a special-case
				constraints.MinRelayParentNumber = pending.relayParent.Number
			}

			fragment, err := NewFragment(relayParent, constraints, candidateEntry.candidate)
			if err != nil {
				logger.Warnf("failed to create fragment: %s", err.Error())
				continue
			}

			possibleChildren = append(possibleChildren, &possibleChild{
				fragment:           fragment,
				candidateHash:      candidateEntry.candidateHash,
				outputHeadDataHash: candidateEntry.outputHeadDataHash,
				parentHeadDataHash: candidateEntry.parentHeadDataHash,
			})
		}

		if len(possibleChildren) == 0 {
			break
		}

		// choose the best candidate
		bestCandidate := slices.MinFunc(possibleChildren, func(fst, snd *possibleChild) int {
			// always pick a candidate pending availability as best.
			if f.scope.getPendingAvailability(fst.candidateHash) != nil {
				return -1
			} else if f.scope.getPendingAvailability(snd.candidateHash) != nil {
				return 1
			} else {
				return forkSelectionRule(fst.candidateHash, snd.candidateHash)
			}
		})

		// remove the candidate from storage
		storage.removeCandidate(bestCandidate.candidateHash)

		// update the cumulative constraint modifications
		cumulativeModifications.Stack(bestCandidate.fragment.ConstraintModifications())

		// update the earliest relay parent
		earliestRelayParent = &relayChainBlockInfo{
			Hash:        bestCandidate.fragment.RelayParent().Hash,
			Number:      bestCandidate.fragment.RelayParent().Number,
			StorageRoot: bestCandidate.fragment.RelayParent().StorageRoot,
		}

		node := &fragmentNode{
			fragment:                bestCandidate.fragment,
			candidateHash:           bestCandidate.candidateHash,
			parentHeadDataHash:      bestCandidate.parentHeadDataHash,
			outputHeadDataHash:      bestCandidate.outputHeadDataHash,
			cumulativeModifications: cumulativeModifications.Clone(),
		}

		// add the candidate to the chain now
		f.bestChain.push(node)
	}
}

// checkCyclesOrInvalidTree checks whether a candidate outputting this head data would
// introduce a cycle or multiple paths to the same state. Trivial 0-length cycles are
// checked  in `newCandidateEntry`.
func (f *fragmentChain) checkCyclesOrInvalidTree(outputHeadDataHash common.Hash) error {
	// this should catch a cycle where this candidate would point back to the parent
	// of some candidate in the chain
	_, ok := f.bestChain.byParentHead[outputHeadDataHash]
	if ok {
		return errCycle
	}

	// multiple paths to the same state, which cannot happen for a chain
	_, ok = f.bestChain.byOutputHead[outputHeadDataHash]
	if ok {
		return errMultiplePaths
	}

	return nil
}

// revertTo reverts the best backable chain so that the last candidate will be one outputting the given
// `parent_head_hash`. If the `parent_head_hash` is exactly the required parent of the base
// constraints (builds on the latest included candidate), revert the entire chain.
// Return false if we couldn't find the parent head hash
func (f *fragmentChain) revertTo(parentHeadDataHash common.Hash) bool {
	var removedItems []*fragmentNode = nil

	requiredParentHash, err := f.scope.baseConstraints.RequiredParent.Hash()
	if err != nil {
		panic(fmt.Sprintf("failed while hashing required parent: %s", err.Error()))
	}

	if requiredParentHash == parentHeadDataHash {
		removedItems = f.bestChain.clear()
	}

	if _, ok := f.bestChain.byOutputHead[parentHeadDataHash]; removedItems == nil && ok {
		removedItems = f.bestChain.revertToParentHash(parentHeadDataHash)
	}

	if removedItems == nil {
		return false
	}

	// Even if it's empty, we need to return true, because we'll be able to add a new candidate
	// to the chain.
	for _, node := range removedItems {
		_ = f.unconnected.addCandidateEntry(newCandidateEntryFromFragment(node))
	}

	return true
}
