package fragmentchain

import (
	"fmt"
	"iter"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	inclusionemulator "github.com/ChainSafe/gossamer/dot/parachain/util/inclusion-emulator"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/tidwall/btree"
)

type CandidateState int

const (
	Seconded CandidateState = iota
	Backed
)

// CandidateEntry represents a candidate into the CandidateStorage
// TODO: Should CandidateEntry implements `HypotheticalOrConcreteCandidate`
type CandidateEntry struct {
	candidateHash      parachaintypes.CandidateHash
	parentHeadDataHash common.Hash
	outputHeadDataHash common.Hash
	relayParent        common.Hash
	// TODO: this is under a Arc<ProspectiveCandidate> smart pointer, should we
	// have that here? maybe some specialized struct that protects the underlying data?
	candidate inclusionemulator.ProspectiveCandidate
	state     CandidateState
}

func (c *CandidateEntry) Hash() parachaintypes.CandidateHash {
	return c.candidateHash
}

func NewCandidateEntry(
	candidateHash parachaintypes.CandidateHash,
	candidate parachaintypes.CommittedCandidateReceipt,
	persistedValidationData parachaintypes.PersistedValidationData,
	state CandidateState,
) (*CandidateEntry, error) {
	pvdHash, err := persistedValidationData.Hash()
	if err != nil {
		return nil, fmt.Errorf("while hashing persisted validation data: %w", err)
	}

	if pvdHash != candidate.Descriptor.PersistedValidationDataHash {
		return nil, ErrPersistedValidationDataMismatch
	}

	parendHeadDataHash, err := persistedValidationData.ParentHead.Hash()
	if err != nil {
		return nil, fmt.Errorf("while hashing parent head data: %w", err)
	}

	outputHeadDataHash, err := candidate.Commitments.HeadData.Hash()
	if err != nil {
		return nil, fmt.Errorf("while hashing output head data: %w", err)
	}

	if parendHeadDataHash == outputHeadDataHash {
		return nil, ErrCandidateEntryZeroLengthCycle
	}

	return &CandidateEntry{
		candidateHash:      candidateHash,
		parentHeadDataHash: parendHeadDataHash,
		outputHeadDataHash: outputHeadDataHash,
		relayParent:        candidate.Descriptor.RelayParent,
		state:              state,
		candidate: inclusionemulator.ProspectiveCandidate{
			Commitments:             candidate.Commitments,
			PersistedValidationData: persistedValidationData,
			PoVHash:                 candidate.Descriptor.PovHash,
			ValidationCodeHash:      candidate.Descriptor.ValidationCodeHash,
		},
	}, nil
}

// CandidateStorage is an utility for storing candidates and information about them such as
// their relay-parents and their backing states. This does not assume any restriction on whether
// or not candidates form a chain. Useful for storing all kinds of candidates.
type CandidateStorage struct {
	byParentHead    map[common.Hash]map[parachaintypes.CandidateHash]any
	byOutputHead    map[common.Hash]map[parachaintypes.CandidateHash]any
	byCandidateHash map[parachaintypes.CandidateHash]*CandidateEntry
}

func (c *CandidateStorage) AddPendingAvailabilityCandidate(
	candidateHash parachaintypes.CandidateHash,
	candidate parachaintypes.CommittedCandidateReceipt,
	persistedValidationData parachaintypes.PersistedValidationData,
) error {
	entry, err := NewCandidateEntry(candidateHash, candidate, persistedValidationData, Backed)
	if err != nil {
		return err
	}

	return c.addCandidateEntry(entry)
}

// Len return the number of stored candidate
func (c *CandidateStorage) Len() uint {
	return uint(len(c.byCandidateHash))
}

func (c *CandidateStorage) addCandidateEntry(candidate *CandidateEntry) error {
	_, ok := c.byCandidateHash[candidate.candidateHash]
	if ok {
		return ErrCandidateAlradyKnown
	}

	// updates the reference parent hash -> candidate
	// we don't check the `ok` value since the key can
	// exists in the map but pointing to a nil hashset
	setOfCandidates := c.byParentHead[candidate.parentHeadDataHash]
	if setOfCandidates == nil {
		setOfCandidates = make(map[parachaintypes.CandidateHash]any)
	}
	setOfCandidates[candidate.candidateHash] = struct{}{}
	c.byParentHead[candidate.parentHeadDataHash] = setOfCandidates

	// udpates the reference output hash -> candidate
	setOfCandidates = c.byOutputHead[candidate.outputHeadDataHash]
	if setOfCandidates == nil {
		setOfCandidates = make(map[parachaintypes.CandidateHash]any)
	}
	setOfCandidates[candidate.candidateHash] = struct{}{}
	c.byOutputHead[candidate.outputHeadDataHash] = setOfCandidates

	return nil
}

func (c *CandidateStorage) removeCandidate(candidateHash parachaintypes.CandidateHash) {
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

func (c *CandidateStorage) markBacked(candidateHash parachaintypes.CandidateHash) {
	entry, ok := c.byCandidateHash[candidateHash]
	if !ok {
		fmt.Println("candidate not found while marking as backed")
	}

	entry.state = Backed
	fmt.Println("candidate marked as backed")
}

func (c *CandidateStorage) contains(candidateHash parachaintypes.CandidateHash) bool {
	_, ok := c.byCandidateHash[candidateHash]
	return ok
}

// candidates returns an iterator over references to the stored candidates, in arbitrary order.
func (c *CandidateStorage) candidates() iter.Seq[*CandidateEntry] {
	return func(yield func(*CandidateEntry) bool) {
		for _, entry := range c.byCandidateHash {
			if !yield(entry) {
				return
			}
		}
	}
}

func (c *CandidateStorage) headDataByHash(hash common.Hash) *parachaintypes.HeadData {
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

func (c *CandidateStorage) possibleBackedParaChildren(parentHeadHash common.Hash) iter.Seq[*CandidateEntry] {
	return func(yield func(*CandidateEntry) bool) {
		seqOfCandidateHashes, ok := c.byParentHead[parentHeadHash]
		if !ok {
			return
		}

		for candidateHash := range seqOfCandidateHashes {
			if entry, ok := c.byCandidateHash[candidateHash]; ok && entry.state == Backed {
				if !yield(entry) {
					return
				}
			}
		}
	}
}

// PendindAvailability is a candidate on-chain but pending availability, for special
// treatment in the `Scope`
type PendindAvailability struct {
	CandidateHash parachaintypes.CandidateHash
	RelayParent   inclusionemulator.RelayChainBlockInfo
}

// The scope of a fragment chain
type Scope struct {
	// the relay parent we're currently building on top of
	relayParent inclusionemulator.RelayChainBlockInfo
	// the other relay parents candidates are allowed to build upon,
	// mapped by the block number
	ancestors *btree.Map[uint, inclusionemulator.RelayChainBlockInfo]
	// the other relay parents candidates are allowed to build upon,
	// mapped by hash
	ancestorsByHash map[common.Hash]inclusionemulator.RelayChainBlockInfo
	// candidates pending availability at this block
	pendindAvailability []*PendindAvailability
	// the base constraints derived from the latest included candidate
	baseConstraints parachaintypes.Constraints
	// equal to `max_candidate_depth`
	maxDepth uint
}

// NewScopeWithAncestors defines a new scope, all arguments are straightforward
// expect ancestors. Ancestor should be in reverse order, starting with the parent
// of the relayParent, and proceeding backwards in block number decrements of 1.
// Ancestors not following these conditions will be rejected.
//
// This function will only consume ancestors up to the `MinRelayParentNumber` of the
// `baseConstraints`.
//
// Only ancestor whose children have the same session id as the relay parent's children
// should be provided. It is allowed to provide 0 ancestors.
func NewScopeWithAncestors(
	relayParent inclusionemulator.RelayChainBlockInfo,
	baseConstraints parachaintypes.Constraints,
	pendingAvailability []*PendindAvailability,
	maxDepth uint,
	ancestors iter.Seq[inclusionemulator.RelayChainBlockInfo],
) (*Scope, error) {
	ancestorsMap := btree.NewMap[uint, inclusionemulator.RelayChainBlockInfo](100)
	ancestorsByHash := make(map[common.Hash]inclusionemulator.RelayChainBlockInfo)

	prev := relayParent.Number
	for ancestor := range ancestors {
		if prev == 0 {
			return nil, ErrUnexpectedAncestor{Number: ancestor.Number, Prev: prev}
		}

		if ancestor.Number != prev-1 {
			return nil, ErrUnexpectedAncestor{Number: ancestor.Number, Prev: prev}
		}

		if prev == baseConstraints.MinRelayParentNumber {
			break
		}

		prev = ancestor.Number
		ancestorsByHash[ancestor.Hash] = ancestor
		ancestorsMap.Set(ancestor.Number, ancestor)
	}

	return &Scope{
		relayParent:         relayParent,
		baseConstraints:     baseConstraints,
		pendindAvailability: pendingAvailability,
		maxDepth:            maxDepth,
		ancestors:           ancestorsMap,
		ancestorsByHash:     ancestorsByHash,
	}, nil
}

// EarliestRelayParent gets the earliest relay-parent allowed in the scope of the fragment chain.
func (s *Scope) EarliestRelayParent() inclusionemulator.RelayChainBlockInfo {
	if iter := s.ancestors.Iter(); iter.Next() {
		return iter.Value()
	}
	return s.relayParent
}

// Ancestor gets the relay ancestor of the fragment chain by hash.
func (s *Scope) Ancestor(hash common.Hash) *inclusionemulator.RelayChainBlockInfo {
	if hash == s.relayParent.Hash {
		return &s.relayParent
	}

	if blockInfo, ok := s.ancestorsByHash[hash]; ok {
		return &blockInfo
	}

	return nil
}

// Whether the candidate in question is one pending availability in this scope.
func (s *Scope) GetPendingAvailability(candidateHash parachaintypes.CandidateHash) *PendindAvailability {
	for _, c := range s.pendindAvailability {
		if c.CandidateHash == candidateHash {
			return c
		}
	}
	return nil
}

type FragmentNode struct {
	fragment                inclusionemulator.Fragment
	candidateHash           parachaintypes.CandidateHash
	cumulativeModifications inclusionemulator.ConstraintModifications
	parentHeadDataHash      common.Hash
	outputHeadDataHash      common.Hash
}

// AnswerMinimumRelayParentsRequest checks the requested relay parents
// and returns those that have associated candidates stored.
func (c *CandidateStorage) AnswerMinimumRelayParentsRequest(requestedRelayParents []common.Hash) ([]common.Hash, error) {
	// Return an error if no relay parents are provided in the request
	if len(requestedRelayParents) == 0 {
		return nil, fmt.Errorf("no relay parents requested")
	}

	// Slice to store valid relay parents
	validRelayParents := make([]common.Hash, 0)

	// Iterate through the requested relay parents
	for _, relayParent := range requestedRelayParents {
		// Check if the relay parent has any associated candidates
		if candidates, exists := c.byParentHead[relayParent]; exists && len(candidates) > 0 {
			validRelayParents = append(validRelayParents, relayParent)
		}
	}

	// Return the list of valid relay parents
	return validRelayParents, nil
}
