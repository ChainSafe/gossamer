package util

import (
	"errors"
	"fmt"
	"slices"
	"time"

	prospectiveparachain "github.com/ChainSafe/gossamer/dot/parachain/prospective-parachains/messages"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime"
)

// Always aim to retain 1 block before the active leaves.
const minimumRetainLength parachaintypes.BlockNumber = 2

var errLeafAlreadyKnown = errors.New("leaf was already known")

// NewBackingImplicitView creates a new backing implicit view with the given runtime instance
func NewBackingImplicitView(blockState BlockState, collatingFor *parachaintypes.ParaID) *BackingImplicitView {
	return &BackingImplicitView{
		leaves:           make(map[common.Hash]activeLeafPruningInfo),
		blockInfoStorage: make(map[common.Hash]blockInfo),
		collatingFor:     collatingFor,
		blockState:       blockState,
	}
}

// BackingImplicitView represents the implicit view of the relay chain derived from
// the immediate view. It is composed of active leaves, block information storage,
// and the minimum relay-parents allowed for candidates of various parachains at
// those leaves.
type BackingImplicitView struct {
	// leaves is a map of active leaves, where the key is the leaf hash and
	// the value is the pruning information for that leaf.
	leaves           map[common.Hash]activeLeafPruningInfo
	blockInfoStorage map[common.Hash]blockInfo
	// collatingFor indicates whether the node is a collator and is only interested in the allowed relay parents
	// of a single ParaID.
	// If this is non-nil, the node operates as a collator, and the prospective-parachains are no longer queried.
	collatingFor *parachaintypes.ParaID
	// blockState provides access to block storage
	blockState BlockState
}

// Leaves returns the active leaves in the implicit view.
func (view *BackingImplicitView) Leaves() []common.Hash {
	leaves := make([]common.Hash, 0, len(view.leaves))
	for leaf := range view.leaves {
		leaves = append(leaves, leaf)
	}
	return leaves
}

// ContainsLeaf checks if the implicit view contains a specific leaf hash.
func (view *BackingImplicitView) ContainsLeaf(leafHash common.Hash) bool {
	_, ok := view.leaves[leafHash]
	return ok
}

// AllAllowedRelayParents returns all allowed relay-parents in the view with no particular order.
//
// IMPORTANT: not all blocks are guaranteed to be allowed for some leaves, it may
// happen that a block info is only kept in the view storage because of a retaining rule.
//
// For getting relay-parents that are valid for parachain candidates use `KnownAllowedRelayParentsUnder` method.
func (view *BackingImplicitView) AllAllowedRelayParents() []common.Hash {
	allRelayParents := make([]common.Hash, 0, len(view.blockInfoStorage))
	for relayParent := range view.blockInfoStorage {
		allRelayParents = append(allRelayParents, relayParent)
	}
	return allRelayParents
}

// ActivateLeaf activates a leaf in the implicit view.
// This will request the minimum relay parents for the leaf and will load headers in the
// ancestry of the leaf as needed. These are the 'implicit ancestors' of the leaf.
//
// To maximise reuse of outdated leaves, it's best to activate new leaves before
// deactivating old ones.
func (view *BackingImplicitView) ActivateLeaf(leafHash common.Hash, subsystemToOverseer chan<- any) error {
	if _, exists := view.leaves[leafHash]; exists {
		return fmt.Errorf("%w: %s", errLeafAlreadyKnown, leafHash)
	}

	fetched, err := view.fetchFreshLeafAndInsertAncestry(leafHash, subsystemToOverseer)
	if err != nil {
		return err
	}

	// Retain at least `minimumRetainLength` blocks in storage.
	// This helps to avoid Chain API calls when activating leaves in the same chain.
	retainMinimum := min(
		fetched.minimumAncestorNumber,
		max(fetched.leafNumber-minimumRetainLength, 0),
	)
	// store the leaf in the active leaves map
	view.leaves[leafHash] = activeLeafPruningInfo{retainMinimum: retainMinimum}
	return nil
}

// Deactivate a leaf in the view. This prunes any outdated implicit ancestors as well.
// Returns hashes of blocks pruned from storage.
func (view *BackingImplicitView) DeactivateLeaf(leafHash common.Hash) []common.Hash {
	// Check if the leaf exists in the active leaves map
	if _, exists := view.leaves[leafHash]; !exists {
		return nil
	}

	// Remove the leaf from the active leaves map
	delete(view.leaves, leafHash)

	// Determine the minimum retain block number across all remaining leaves
	minimumBlockNumber := ^parachaintypes.BlockNumber(0) // Max value for BlockNumber
	for _, leafInfo := range view.leaves {
		if leafInfo.retainMinimum < minimumBlockNumber {
			minimumBlockNumber = leafInfo.retainMinimum
		}
	}

	// Prune everything before the minimum out of all leaves,
	// pruning absolutely everything if there are no leaves (empty view)

	// prune blocks with blockNumber < minimum
	var pruned []common.Hash
	for hash, info := range view.blockInfoStorage {
		if info.blockNumber < minimumBlockNumber {
			delete(view.blockInfoStorage, hash)
			pruned = append(pruned, hash)
		}
	}
	return pruned
}

// KnownAllowedRelayParentsUnder returns known, allowed relay-parents that are valid for parachain candidates
// which could be backed in a child of a given block for a given para ID.
//
// This is expressed as a contiguous slice of relay-chain block hashes which may
// include the provided block hash itself.
//
// If `paraID` is nil, this returns all valid relay-parents across all paras for the leaf.
func (view *BackingImplicitView) KnownAllowedRelayParentsUnder(
	blockHash common.Hash, paraID *parachaintypes.ParaID) []common.Hash {
	blockInfo, exists := view.blockInfoStorage[blockHash]
	if !exists {
		return nil
	}

	return blockInfo.allowedRelayParents.allowedFor(paraID, blockInfo.blockNumber)
}

func (view *BackingImplicitView) fetchFreshLeafAndInsertAncestry(
	leafHash common.Hash, subsystemToOverseer chan<- any) (*fetchSummary, error) {
	// Get the block header directly from block state
	blockHeader, err := view.blockState.GetHeader(leafHash)
	if err != nil {
		return nil, fmt.Errorf("getting block header for leaf %s: %w", leafHash, err)
	}

	minRelayParents, err := view.findMinRelayParents(leafHash, blockHeader, subsystemToOverseer)
	if err != nil {
		return nil, fmt.Errorf("getting min relay parents for leaf %s: %w", leafHash, err)
	}

	minAncestorNumber := ^parachaintypes.BlockNumber(0)
	if len(minRelayParents) > 0 {
		for _, paraIDBlockNumber := range minRelayParents {
			if paraIDBlockNumber.BlockNumber < minAncestorNumber {
				minAncestorNumber = paraIDBlockNumber.BlockNumber
			}
		}
	} else {
		minAncestorNumber = parachaintypes.BlockNumber(blockHeader.Number)
	}

	expectedAncestryLen := max(blockHeader.Number-uint(minAncestorNumber), 0) + 1
	ancestry := make([]common.Hash, 0, expectedAncestryLen)
	ancestry = append(ancestry, leafHash)

	if blockHeader.Number > 0 {
		nextAncestorNumber := parachaintypes.BlockNumber(blockHeader.Number - 1)

		ancestorHashes, err := view.fetchAncestorsUpToMinBlockNumber(
			blockHeader.ParentHash, nextAncestorNumber, minAncestorNumber)
		if err != nil {
			return nil, err
		}

		ancestry = append(ancestry, ancestorHashes...)
	}

	// Create allowed relay parents for the leaf block
	allowed := &allowedRelayParents{
		minimumRelayParent:            make(map[parachaintypes.ParaID]parachaintypes.BlockNumber),
		allowedRelayParentsContiguous: ancestry,
	}

	// Store minimum relay parents
	for _, parent := range minRelayParents {
		allowed.minimumRelayParent[parent.ParaId] = parent.BlockNumber
	}

	// Store the leaf block info
	view.blockInfoStorage[leafHash] = blockInfo{
		blockNumber:         parachaintypes.BlockNumber(blockHeader.Number),
		parentHash:          blockHeader.ParentHash,
		allowedRelayParents: allowed,
	}

	return &fetchSummary{
		minimumAncestorNumber: minAncestorNumber,
		leafNumber:            parachaintypes.BlockNumber(blockHeader.Number),
	}, nil
}

// fetchAncestorsUpToMinBlockNumber fetches ancestor blocks up to minBlockNumber if we're not at genesis
// Returns the ancestry hashes and any error encountered
func (view *BackingImplicitView) fetchAncestorsUpToMinBlockNumber(
	nextAncestorHash common.Hash,
	nextAncestorNumber parachaintypes.BlockNumber,
	minBlockNumber parachaintypes.BlockNumber,
) ([]common.Hash, error) {
	var ancestry []common.Hash

	// Fetch and store ancestors until we reach minBlockNumber
	for nextAncestorNumber >= minBlockNumber {
		var parentHash common.Hash

		// Check if block info is already in storage
		if info, exists := view.blockInfoStorage[nextAncestorHash]; exists {
			parentHash = info.parentHash
		} else {
			// Load header and insert into block storage
			header, err := view.blockState.GetHeader(nextAncestorHash)
			if err != nil {
				return nil, fmt.Errorf("getting header for ancestor %s: %w", nextAncestorHash, err)
			}

			parentHash = header.ParentHash

			// Store block info
			view.blockInfoStorage[nextAncestorHash] = blockInfo{
				blockNumber:         nextAncestorNumber,
				parentHash:          header.ParentHash,
				allowedRelayParents: nil, // No relay parents for non-leaf blocks
			}
		}

		ancestry = append(ancestry, nextAncestorHash)

		// Stop at genesis
		if nextAncestorNumber == 0 {
			break
		}

		nextAncestorNumber--
		nextAncestorHash = parentHash
	}

	return ancestry, nil
}

// findMinRelayParents finds the minimum relay parents for a given leaf hash.
// If the node is a collator, bypass prospective-parachains. We're only interested in the one paraid
func (view *BackingImplicitView) findMinRelayParents(
	leafHash common.Hash, blockHeader *types.Header, subsystemToOverseer chan<- any,
) ([]prospectiveparachain.ParaIDBlockNumber, error) {
	paraID := view.collatingFor

	if paraID == nil {
		return view.fetchMinRelayParentsFromProspectiveParachains(leafHash, subsystemToOverseer)
	}

	minBlock, err := view.fetchMinRelayParentsForCollator(leafHash, parachaintypes.BlockNumber(blockHeader.Number))
	if err != nil {
		return nil, fmt.Errorf("getting min relay parent for collator %s: %w", leafHash, err)
	}

	minRelayParent := prospectiveparachain.ParaIDBlockNumber{ParaId: *paraID, BlockNumber: *minBlock}
	return []prospectiveparachain.ParaIDBlockNumber{minRelayParent}, nil
}

// Request the min relay parents from prospective-parachains.
func (*BackingImplicitView) fetchMinRelayParentsFromProspectiveParachains(
	leafHash common.Hash,
	subsystemToOverseer chan<- any,
) ([]prospectiveparachain.ParaIDBlockNumber, error) {
	getMin := prospectiveparachain.GetMinimumRelayParents{
		RelayChainBlockHash: leafHash,
		Sender:              make(chan []prospectiveparachain.ParaIDBlockNumber),
	}

	subsystemToOverseer <- getMin
	select {
	case result := <-getMin.Sender:
		return result, nil
	case <-time.After(5 * time.Second):
		return nil, fmt.Errorf("timeout while waiting for relay parents for leaf %s", leafHash)
	}
}

// fetchMinRelayParentsForCollator calculates the minimum number of relay parents required
// for a collator based on the given leaf block and its ancestors. It determines this by
// checking session indices of ancestor blocks until finding one with a different session
// or reaching the minimum required depth.
func (view *BackingImplicitView) fetchMinRelayParentsForCollator(
	leafHash common.Hash,
	leafNumber parachaintypes.BlockNumber,
) (*parachaintypes.BlockNumber, error) {
	blockState := view.blockState

	instance, err := blockState.GetRuntime(leafHash)
	if err != nil {
		return nil, fmt.Errorf("getting runtime instance for leaf %s: %w", leafHash, err)
	}

	requiredSession, err := instance.ParachainHostSessionIndexForChild()
	if err != nil {
		return nil, fmt.Errorf("getting session index for leaf %s: %w", leafHash, err)
	}

	schedulingLookahead, err := instance.ParachainHostSchedulingLookAhead()
	if err != nil {
		return nil, fmt.Errorf("getting scheduling lookahead for leaf %s: %w", leafHash, err)
	}

	ancestors, err := FetchAncestorBlockHashes(leafHash, schedulingLookahead, blockState)
	if err != nil {
		return nil, fmt.Errorf("getting ancestor block hashes for leaf %s: %w", leafHash, err)
	}

	// Start with current leaf number as minimum relay parents
	relayParentsRequired := leafNumber

	// Traverse ancestors until we find a block with different session or reach minimum depth
	for _, ancestorHash := range ancestors {
		ancestorSession, err := requestSessionIndexForChild(ancestorHash, blockState)
		if err != nil {
			return nil, err // error already wrapped
		}

		// Break if we find a block from a different session
		if ancestorSession == nil || *ancestorSession != requiredSession {
			break
		}

		// Decrement required relay parents and break if we've reached minimum
		relayParentsRequired--
		if relayParentsRequired == 0 {
			break
		}
	}

	return &relayParentsRequired, nil
}

type activeLeafPruningInfo struct {
	// The minimum block in the same branch of the relay-chain that should be preserved.
	retainMinimum parachaintypes.BlockNumber
}

type blockInfo struct {
	blockNumber parachaintypes.BlockNumber

	// allowedRelayParents represents the implicit views of previous relay parents
	// that were active leaves. This is useful for understanding the views of peers
	// in the network, which may not always be in perfect synchrony with our own view.
	//
	// If a peer is ahead of us with a new leaf, we may not recognise the block hash,
	// and there's nothing we can do. However, if a peer is behind us, retaining
	// information about previous leaves' implicit views allows us to continue sending
	// relevant messages to them until they catch up.
	allowedRelayParents *allowedRelayParents
	parentHash          common.Hash
}

// allowedRelayParents represents the minimum relay parents implicitly relative to a particular block.
type allowedRelayParents struct {
	// minimumRelayParent maps ParaID to the minimum block number of relay parents.
	// This is only populated for active leaves, so it will be empty for blocks
	// that have never been witnessed as active leaves.
	minimumRelayParent map[parachaintypes.ParaID]parachaintypes.BlockNumber

	// allowedRelayParentsContiguous contains the ancestry in descending order,
	// starting from the block hash itself down to and including the minimum of `minimumRelayParent`.
	allowedRelayParentsContiguous []common.Hash
}

// allowedFor returns the allowed relay parents for a given parachain ID and base block number.
func (a *allowedRelayParents) allowedFor(
	paraID *parachaintypes.ParaID, baseNumber parachaintypes.BlockNumber) []common.Hash {
	if paraID == nil {
		return a.allowedRelayParentsContiguous
	}

	minBlockNumber, exists := a.minimumRelayParent[*paraID]
	if !exists || baseNumber < minBlockNumber {
		return nil
	}

	diff := int(baseNumber - minBlockNumber)

	// difference of 0 should lead to slice len of 1
	sliceLen := min(diff+1, len(a.allowedRelayParentsContiguous))
	return a.allowedRelayParentsContiguous[:sliceLen]
}

type fetchSummary struct {
	minimumAncestorNumber parachaintypes.BlockNumber
	leafNumber            parachaintypes.BlockNumber
}

// BlockState interface defines the methods needed for block operations
type BlockState interface {
	// GetRuntime returns runtime instance for a given block hash.
	GetRuntime(blockHash common.Hash) (instance runtime.Instance, err error)
	// GetHeader returns a block header for a given hash
	GetHeader(hash common.Hash) (*types.Header, error)
}

func requestSessionIndexForChild(
	relayParent common.Hash, blockState BlockState,
) (*parachaintypes.SessionIndex, error) {
	instance, err := blockState.GetRuntime(relayParent)
	if err != nil {
		return nil, fmt.Errorf("getting runtime instance for block %s: %w", relayParent, err)
	}

	sessionIndex, err := instance.ParachainHostSessionIndexForChild()
	if err != nil {
		return nil, fmt.Errorf("getting session index for block %s: %w", relayParent, err)
	}

	return &sessionIndex, nil
}

// PathsViaRelayParent returns all paths from each leaf to the last block in state containing the relay parent.
// If no paths exist the function will return an empty slice.
func (view *BackingImplicitView) PathsViaRelayParent(relayParent common.Hash) [][]common.Hash {
	if len(view.leaves) == 0 {
		// No active leaves so the view should be empty. Don't return any paths.
		return nil
	}

	if _, exists := view.blockInfoStorage[relayParent]; !exists {
		// relayParent is not in the view - don't return any paths
		return nil
	}

	// Find all paths from each leaf to relayParent
	var paths [][]common.Hash
	for leaf := range view.leaves {
		path := make([]common.Hash, 0)
		currentLeaf := leaf
		visited := make(map[common.Hash]struct{})
		pathContainsTarget := false

		// Start from the leaf and traverse all known blocks
		for {
			if _, wasVisited := visited[currentLeaf]; wasVisited {
				// There is a cycle - abandon this path
				break
			}

			if info, exists := view.blockInfoStorage[currentLeaf]; exists {
				// currentLeaf is a known block - add it to the path and mark it as visited
				path = append(path, currentLeaf)
				visited[currentLeaf] = struct{}{}

				// currentLeaf is the target relayParent. Mark the path so that it's included in the result
				if currentLeaf == relayParent {
					pathContainsTarget = true
				}

				// update currentLeaf with the parent
				currentLeaf = info.parentHash
			} else {
				// path is complete
				if pathContainsTarget {
					// we want the path ordered from oldest to newest so reverse it
					slices.Reverse(path)
					paths = append(paths, path)
				}
				break
			}
		}
	}

	return paths
}

// Information about a relay-chain block, to be used when calling this module from prospective
// parachains.
type BlockInfoProspectiveParachains struct {
	// The hash of the relay-chain block.
	Hash common.Hash
	// The hash of the parent relay-chain block.
	ParentHash common.Hash
	// The number of the relay-chain block.
	Number parachaintypes.BlockNumber
	// The storage root of the relay-chain block.
	StorageRoot common.Hash
}

// ActivateLeafFromProspectiveParachains activates a leaf in the view. To be used by the
// prospective parachains subsystem only. This function must be used instead of ActivateLeaf
// when called from prospective-parachains subsystem to prevent a circular deadlock:
// ActivateLeaf sends a message to prospective-parachains and waits for response.
// If prospective-parachains is the caller, it will wait forever for the response.
//
// No-op for known leaves
func (view *BackingImplicitView) ActivateLeafFromProspectiveParachains(
	leaf *BlockInfoProspectiveParachains,
	ancestors []*BlockInfoProspectiveParachains,
) {
	if _, exists := view.leaves[leaf.Hash]; exists {
		return
	}

	ancestorsLen := len(ancestors)

	var retainMinimum parachaintypes.BlockNumber
	if ancestorsLen > 0 {
		retainMinimum = ancestors[ancestorsLen-1].Number
	}

	// Retain at least `minimumRetainLength` blocks in storage.
	// This helps to avoid Chain API calls when activating leaves in the same chain.
	retainMinimum = min(retainMinimum, max(leaf.Number-minimumRetainLength, 0))

	view.leaves[leaf.Hash] = activeLeafPruningInfo{retainMinimum: retainMinimum}

	allowed := &allowedRelayParents{
		// minimumRelayParent is nil since when called from prospective-parachains,
		// the minimum relay parent data is already available and won't be queried from this view
		minimumRelayParent:            nil,
		allowedRelayParentsContiguous: make([]common.Hash, 0, ancestorsLen),
	}

	for _, ancestor := range ancestors {
		view.blockInfoStorage[ancestor.Hash] = blockInfo{
			blockNumber:         ancestor.Number,
			parentHash:          ancestor.ParentHash,
			allowedRelayParents: nil, // No relay parents for non-leaf blocks
		}

		allowed.allowedRelayParentsContiguous = append(allowed.allowedRelayParentsContiguous, ancestor.Hash)
	}

	view.blockInfoStorage[leaf.Hash] = blockInfo{
		blockNumber:         leaf.Number,
		parentHash:          leaf.ParentHash,
		allowedRelayParents: allowed,
	}

	view.leaves[leaf.Hash] = activeLeafPruningInfo{}
}
