package prospectiveparachains

import (
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

// ProspectiveParachainsMessage Messages sent to the Prospective Parachains subsystem.
type ProspectiveParachainsMessage interface {
	isProspectiveParachainsMessage()
}

// IntroduceSecondedCandidate Inform the Prospective Parachains Subsystem of a new seconded candidate.
// The response is false if the candidate was rejected by prospective parachains,
// true otherwise (if it was accepted or already present)
type IntroduceSecondedCandidate struct {
	IntroduceSecondedCandidateRequest
	Response chan bool
}

func (IntroduceSecondedCandidate) isProspectiveParachainsMessage() {}

// IntroduceSecondedCandidateRequest Request introduction of a seconded candidate into the prospective parachains
// subsystem.
type IntroduceSecondedCandidateRequest struct {
	// The para-id of the candidate.
	CandidateParaID parachaintypes.ParaID
	// The candidate receipt itself.
	CandidateReceipt parachaintypes.CommittedCandidateReceipt
	// The persisted validation data of the candidate.
	PersistedValidationData parachaintypes.PersistedValidationData
}

// CandidateBacked Inform the Prospective Parachains Subsystem that a previously introduced candidate
// has been backed. This requires that the candidate was successfully introduced in
// the past.
type CandidateBacked struct {
	ParaId        parachaintypes.ParaID
	CandidateHash parachaintypes.CandidateHash
}

func (CandidateBacked) isProspectiveParachainsMessage() {}

// GetBackableCandidates Try getting requested quantity of backable candidate hashes along with their relay parents
// for the given parachain, under the given relay-parent hash, which is a descendant of the given ancestors.
// Timed out ancestors should not be included in the collection.
// RequestedQty should represent the number of scheduled cores of this ParaId.
// A timed out ancestor frees the cores of all of its descendants, so if there's a hole in the
// supplied ancestor path, we'll get candidates that backfill those timed out slots first. It
// may also return less/no candidates, if there aren't enough backable candidates recorded.
type GetBackableCandidates struct {
	RelayParentHash common.Hash
	ParaId          parachaintypes.ParaID
	RequestedQty    uint32
	Ancestors       Ancestors
	Response        chan []parachaintypes.CandidateHashAndRelayParent
}

func (GetBackableCandidates) isProspectiveParachainsMessage() {}

// Ancestors A collection of ancestor candidates of a parachain.
type Ancestors map[parachaintypes.CandidateHash]struct{}

// GetHypotheticalMembership Get the hypothetical or actual membership of candidates with the given properties
// under the specified active leave's fragment chain.
//
// For each candidate, we return a vector of leaves where the candidate is present or could be
// added. "Could be added" either means that the candidate can be added to the chain right now
// or could be added in the future (we may not have its ancestors yet).
// Note that even if we think it could be added in the future, we may find out that it was
// invalid, as time passes.
// If an active leaf is not in the vector, it means that there's no
// chance this candidate will become valid under that leaf in the future.
//
// If `RragmentChainRelayParent` in the request is not `nil`, the return vector can only
// contain this relay parent (or none).
type GetHypotheticalMembership struct {
	HypotheticalMembershipRequest HypotheticalMembershipRequest
	Response                      chan []HypotheticalMembershipResponseItem
}

func (GetHypotheticalMembership) isProspectiveParachainsMessage() {}

type HypotheticalMembershipResponseItem struct {
	HypotheticalCandidate  parachaintypes.HypotheticalCandidate
	HypotheticalMembership HypotheticalMembership
}

// HypotheticalMembershipRequest Request specifying which candidates are either already included
// or might become included in fragment chain under a given active leaf (or any active leaf if
// `FragmentChainRelayParent` is `nil`).
type HypotheticalMembershipRequest struct {
	// Candidates, in arbitrary order, which should be checked for
	// hypothetical/actual membership in fragment chains.
	Candidates []parachaintypes.HypotheticalCandidate
	// Either a specific fragment chain to check, otherwise all.
	FragmentChainRelayParent *common.Hash
}

// HypotheticalMembership Indicates the relay-parents whose fragment chain a candidate
// is present in or can be added in (right now or in the future).
type HypotheticalMembership []common.Hash

// GetMinimumRelayParents Get the minimum accepted relay-parent number for each para in the fragment chain
// for the given relay-chain block hash.
//
// That is, if the block hash is known and is an active leaf, this returns the
// minimum relay-parent block number in the same branch of the relay chain which
// is accepted in the fragment chain for each para-id.
//
// If the block hash is not an active leaf, this will return an empty vector.
//
// Para-IDs which are omitted from this list can be assumed to have no
// valid candidate relay-parents under the given relay-chain block hash.
//
// Para-IDs are returned in no particular order.
type GetMinimumRelayParents struct {
	RelayChainBlockHash common.Hash
	Sender              chan []ParaIDBlockNumber
}

type ParaIDBlockNumber struct {
	ParaId      parachaintypes.ParaID
	BlockNumber parachaintypes.BlockNumber
}

func (GetMinimumRelayParents) isProspectiveParachainsMessage() {}

// GetProspectiveValidationData Get the validation data of some prospective candidate. The candidate doesn't need
// to be part of any fragment chain, but this only succeeds if the parent head-data and
// relay-parent are part of the `CandidateStorage` (meaning that it's a candidate which is
// part of some fragment chain or which prospective-parachains predicted will become part of
// some fragment chain).
type GetProspectiveValidationData struct {
	ProspectiveValidationDataRequest
	Sender chan parachaintypes.PersistedValidationData
}

// ProspectiveValidationDataRequest A request for the persisted validation data stored in the prospective
// parachains subsystem.
type ProspectiveValidationDataRequest struct {
	// The para-id of the candidate.
	ParaId parachaintypes.ParaID
	// The relay-parent of the candidate.
	CandidateRelayParent common.Hash
	// The parent head-data.
	ParentHeadData ParentHeadData
}

// ParentHeadData The parent head-data hash with optional data itself.
type ParentHeadData interface {
	isParentHeadData()
}

// ParentHeadDataHash Parent head-data hash.
type ParentHeadDataHash common.Hash

func (ParentHeadDataHash) isParentHeadData() {}

type ParentHeadDataWithHash struct {
	// This will be provided for collations with elastic scaling enabled.
	Data parachaintypes.HeadData
	// Parent head-data hash.
	Hash ParentHeadDataHash
}

func (ParentHeadDataWithHash) isParentHeadData() {}
