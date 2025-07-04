// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statementdistribution

import (
	"bytes"
	"fmt"
	"slices"
	"time"

	"github.com/ChainSafe/gossamer/dot/parachain/network-bridge/messages"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/libp2p/go-libp2p/core/peer"
)

type origin byte

const (
	cluster origin = iota
	unspecified
)

// candidateIdentifier is an identifier for a candidate.
//
// In the [requestManager], we are requesting candidates for which we have no information other
// than the candidate hash and statements signed by validators. It is possible for
// validators for multiple groups to abuse this lack of information: until we actually
// get the preimage of this candidate we cannot confirm anything other than the candidate hash.
type candidateIdentifier struct {
	// The relay-parent this candidate is ostensibly under.
	relayParent common.Hash
	// The hash of the candidate.
	candidateHash parachaintypes.CandidateHash
	// The index of the group claiming to be assigned to the candidate's para.
	groupIndex parachaintypes.GroupIndex
}

func (ci candidateIdentifier) String() string {
	return fmt.Sprintf(
		"relayParent=%s candidateHash=%s groupIndex=%d",
		ci.relayParent,
		ci.candidateHash,
		ci.groupIndex,
	)
}

func (ci candidateIdentifier) compare(other candidateIdentifier) int {
	if comp := bytes.Compare(ci.relayParent[:], other.relayParent[:]); comp != 0 {
		return comp
	}

	if comp := bytes.Compare(ci.candidateHash.Value[:], other.candidateHash.Value[:]); comp != 0 {
		return comp
	}

	return int(ci.groupIndex - other.groupIndex)
}

type candidateIdentifierSet map[candidateIdentifier]struct{}

func (s candidateIdentifierSet) insert(id candidateIdentifier) {
	s[id] = struct{}{}
}

func (s candidateIdentifierSet) contains(id candidateIdentifier) bool { //nolint:unused
	_, ok := s[id]
	return ok
}

func (s candidateIdentifierSet) retain(keep func(candidateIdentifier) bool) {
	for id := range s {
		if !keep(id) {
			delete(s, id)
		}
	}
}

// requestProperties is used in target selection and validation of a request.
type requestProperties struct { //nolint:unused
	// A mask for limiting the statements the response is allowed to contain.
	// The mask has `OR` semantics: statements by validators corresponding to bits
	// in the mask are not desired. It also returns the required backing threshold
	// for the candidate.
	unwantedMask parachaintypes.StatementFilter

	// The required backing threshold, if any. If this is non-nil, then requests will only
	// be made to peers which can provide enough statements to back the candidate, when
	// taking into account the `unwantedMask`, and a response will only be validated
	// in the case of those statements.
	//
	// If this is nil, it is assumed that only the candidate itself is needed.
	backingThreshold *int
}

type taggedResponse struct { //nolint:unused
	identifier    candidateIdentifier
	requestedPeer peer.ID
	props         requestProperties
	// payload is AttestedCandidateResponse, maybe ReqRespResult should be generic over the response type?
	response chan messages.ReqRespResult
}

type unhandledResponse struct { //nolint:unused
	response taggedResponse
}

type priority struct {
	origin   origin
	attempts uint
}

func (p priority) compare(other priority) int {
	if p.origin != other.origin {
		return int(p.origin - other.origin)
	}

	return int(p.attempts - other.attempts)
}

// requestedCandidate represents a pending request.
type requestedCandidate struct {
	priority priority
	knownBy  []peer.ID // polkadot-sdk uses VecDeque<PeerId>
	// Has the request been sent out and a response not yet received?
	inFlight bool
	// The timestamp for the next time we should retry, if the response failed.
	nextRetryTime *time.Time
}

func (rc requestedCandidate) isPending() bool { //nolint:unused
	if rc.inFlight {
		return false
	}

	if rc.nextRetryTime != nil && time.Now().Before(*rc.nextRetryTime) {
		return false
	}

	return true
}

type priorityCandidatePair struct {
	priority
	candidateIdentifier
}

type priorityCandidatePairs []priorityCandidatePair

func (p *priorityCandidatePairs) binarySearch(priority priority, candidateIdentifier candidateIdentifier) (int, bool) {
	target := priorityCandidatePair{priority, candidateIdentifier}

	return slices.BinarySearchFunc(
		*p,
		target,
		func(pair priorityCandidatePair, target priorityCandidatePair) int {
			if comp := pair.priority.compare(target.priority); comp != 0 {
				return comp
			}
			return pair.candidateIdentifier.compare(target.candidateIdentifier)
		},
	)
}

func (p *priorityCandidatePairs) remove(index int) {
	*p = slices.Delete(*p, index, index+1)
}

func (p *priorityCandidatePairs) insert(prio priority, id candidateIdentifier, index int) {
	*p = slices.Insert(*p, index, priorityCandidatePair{prio, id})
}

func (p *priorityCandidatePairs) retain(keep func(priorityCandidatePair) bool) {
	n := 0
	for _, pair := range *p {
		if keep(pair) {
			(*p)[n] = pair
			n++
		}
	}
	*p = (*p)[:n]
}

// entry is used for manipulating a [requestedCandidate] stored in [requestManager] state.
type entry struct {
	prevIndex  int
	identifier candidateIdentifier
	byPriority *priorityCandidatePairs // using a pointer to enable append/delete ops on the slice
	requested  *requestedCandidate
}

// addPeer adds a peer to the set of known peers.
func (e *entry) addPeer(p peer.ID) {
	if !slices.Contains(e.requested.knownBy, p) {
		e.requested.knownBy = append(e.requested.knownBy, p)
	}
}

// setClusterPriority notes that the candidate is required for the cluster.
// Calling this method mutates the state of the [requestManager] object from which this entry was obtained by calling
// requestManager.getOrInsert().
func (e *entry) setClusterPriority() {
	e.requested.priority.origin = cluster

	_ = insertOrUpdatePriority(
		e.byPriority,
		&e.prevIndex,
		e.identifier,
		e.requested.priority,
	)
}

type requestManager struct {
	requests map[candidateIdentifier]*requestedCandidate
	// sorted by priority.
	byPriority priorityCandidatePairs
	// all unique identifiers for the candidate.
	uniqueIdentifiers map[parachaintypes.CandidateHash]candidateIdentifierSet
}

func newRequestManager() *requestManager {
	return &requestManager{
		requests:          make(map[candidateIdentifier]*requestedCandidate),
		byPriority:        make(priorityCandidatePairs, 0),
		uniqueIdentifiers: make(map[parachaintypes.CandidateHash]candidateIdentifierSet),
	}
}

// getOrInsert gets an [entry] for mutating a request and inserts it if the
// manager doesn't store this request already.
func (rm *requestManager) getOrInsert(
	relayParent common.Hash,
	candidateHash parachaintypes.CandidateHash,
	groupIndex parachaintypes.GroupIndex,
) *entry {
	identifier := candidateIdentifier{relayParent, candidateHash, groupIndex}
	var priorityIndex int

	candidate, knownRequest := rm.requests[identifier]
	if !knownRequest {
		candidate = &requestedCandidate{
			priority:      priority{origin: unspecified, attempts: 0},
			knownBy:       make([]peer.ID, 0),
			inFlight:      false,
			nextRetryTime: nil,
		}

		rm.requests[identifier] = candidate

		if uniques, ok := rm.uniqueIdentifiers[candidateHash]; !ok {
			rm.uniqueIdentifiers[candidateHash] = candidateIdentifierSet{
				identifier: {},
			}
		} else {
			uniques.insert(identifier)
		}

		priorityIndex = insertOrUpdatePriority(
			&rm.byPriority,
			nil,
			identifier,
			candidate.priority,
		)
	} else {
		var found bool
		priorityIndex, found = rm.byPriority.binarySearch(candidate.priority, identifier)
		if !found {
			panic("requested candidates always have a priority entry; qed")
		}
	}

	return &entry{
		prevIndex:  priorityIndex,
		identifier: identifier,
		byPriority: &rm.byPriority,
		requested:  candidate,
	}
}

// removeFor removes all pending requests for the given candidate.
func (rm *requestManager) removeFor(candidate parachaintypes.CandidateHash) { //nolint:unused
	identifiers, ok := rm.uniqueIdentifiers[candidate]
	if !ok {
		return
	}

	rm.byPriority.retain(func(pair priorityCandidatePair) bool {
		return identifiers.contains(pair.candidateIdentifier)
	})

	for id := range identifiers {
		delete(rm.requests, id)
	}
}

// removeByRelayParent removes pending requests based on relay-parent.
func (rm *requestManager) removeByRelayParent(relayParent common.Hash) {
	candidateHashes := make(map[parachaintypes.CandidateHash]struct{})

	// Remove from byPriority and requests.
	rm.byPriority.retain(func(pair priorityCandidatePair) bool {
		id := pair.candidateIdentifier
		retain := relayParent != id.relayParent
		if !retain {
			delete(rm.requests, id)
			candidateHashes[id.candidateHash] = struct{}{}
		}
		return retain
	})

	// Remove from uniqueIdentifiers.
	for candidateHash := range candidateHashes {
		uniqueIds, ok := rm.uniqueIdentifiers[candidateHash]
		if ok {
			uniqueIds.retain(func(id candidateIdentifier) bool {
				return relayParent != id.relayParent
			})

			if len(uniqueIds) == 0 {
				delete(rm.uniqueIdentifiers, candidateHash)
			}
		}
	}

	logger.Debugf("Requests remaining after cleanup: %d", len(rm.byPriority))
}

// hasPendingRequests returns true if there are pending requests that are dispatchable.
func (rm *requestManager) hasPendingRequests() bool { //nolint:unused
	for _, request := range rm.requests {
		if request.isPending() {
			return true
		}
	}
	return false
}

// nextRetryTime returns the time at which the next request to be retried will be ready.
func (rm *requestManager) nextRetryTime() *time.Time { //nolint:unused
	var next *time.Time

	for _, request := range rm.requests {
		if request.inFlight || request.nextRetryTime == nil {
			continue
		}

		if next == nil || request.nextRetryTime.Before(*next) {
			next = request.nextRetryTime
		}
	}

	return next
}

// TODO: replace with implementation (#4378)
type responseManager interface { //nolint:unused
	incoming() *unhandledResponse
	len() int
	push(taggedResponse, peer.ID)
	isSendingTo(peer.ID) bool
}

// nextRequest yields the next request to dispatch, if there is any.
//
// This function accepts two closures as an argument.
//
// The first closure is used to gather information about the desired
// properties of a response, which is used to select targets and validate
// the response later on.
//
// The second closure is used to determine the specific advertised
// statements by a peer, to be compared against the mask and backing
// threshold and returns `None` if the peer is no longer connected.
func (rm *requestManager) nextRequest( //nolint:unused
	responseManager responseManager,
	requestProps func(candidateIdentifier) *requestProperties,
	peerAdvertised func(candidateIdentifier, peer.ID) *parachaintypes.StatementFilter,
) *messages.OutgoingRequest {
	// The number of parallel requests a node can answer is limited by
	// `MAX_PARALLEL_ATTESTED_CANDIDATE_REQUESTS`, however there is no
	// need for the current node to limit itself to the same amount of
	// requests, because the requests are going to different nodes anyways.
	// While looking at https://github.com/paritytech/polkadot-sdk/issues/3314,
	// found out that these requests take around 100ms to fulfil, so it
	// would make sense to try to request things as early as we can, given
	// we would need to request it for each candidate, around 25 right now
	// on kusama.
	if responseManager.len() >= 2*messages.MaxParallelAttestedCandidateRequests {
		return nil
	}

	var res *messages.OutgoingRequest

	type idWithIndex struct {
		id    candidateIdentifier
		index int
	}
	var cleanupOutdated []idWithIndex

	for i, pair := range rm.byPriority {
		id := pair.candidateIdentifier
		entry, ok := rm.requests[id]
		if !ok {
			logger.Errorf("Missing entry for priority queue member identifier=%s", id.String())
			continue
		}

		if !entry.isPending() {
			continue
		}

		props := requestProps(id)
		if props == nil {
			cleanupOutdated = append(cleanupOutdated, idWithIndex{id, i})
			continue
		}

		target := findRequestTargetWithUpdate(
			&entry.knownBy,
			id,
			*props,
			peerAdvertised,
			responseManager,
		)
		if target == nil {
			continue
		}

		logger.Debugf("Issuing candidate request candidateHash=%s peer=%s", id.candidateHash.String(), *target)

		res = messages.NewOutgoingRequest(
			parachaintypes.PeerID(*target),
			&messages.AttestedCandidateRequest{
				CandidateHash: id.candidateHash,
				Mask:          props.unwantedMask,
			},
		)

		responseManager.push(
			taggedResponse{
				identifier:    id,
				requestedPeer: *target,
				props:         *props,
				response:      res.Result,
			},
			*target,
		)

		break
	}

	for _, idIdx := range cleanupOutdated {
		rm.byPriority.remove(idIdx.index)
		delete(rm.requests, idIdx.id)

		identifiers, ok := rm.uniqueIdentifiers[idIdx.id.candidateHash]
		if ok {
			delete(identifiers, idIdx.id)
			if len(identifiers) == 0 {
				delete(rm.uniqueIdentifiers, idIdx.id.candidateHash)
			}
		}
	}

	return res
}

func insertOrUpdatePriority(
	prioritySorted *priorityCandidatePairs,
	prevIndex *int,
	candidateIdentifier candidateIdentifier,
	newPriority priority,
) int {
	if prevIndex != nil {
		// GIGO: this behaves strangely if prev-index is not for the
		// expected identifier.
		if (*prioritySorted)[*prevIndex].priority == newPriority {
			// unchanged.
			return *prevIndex
		}
		prioritySorted.remove(*prevIndex)
	}

	index, found := prioritySorted.binarySearch(newPriority, candidateIdentifier)
	if !found {
		prioritySorted.insert(newPriority, candidateIdentifier, index)
	}

	return index
}

// findRequestTargetWithUpdate finds a valid request target, returning nil if none exists.
// Cleans up disconnected peers and places the returned peer at the back of the queue.
// The method mutates the `knownBy` argument.
func findRequestTargetWithUpdate( //nolint:unused
	knownBy *[]peer.ID,
	candidateIdentifier candidateIdentifier,
	props requestProperties,
	peerAdvertised func(candidateIdentifier, peer.ID) *parachaintypes.StatementFilter,
	responseManager responseManager,
) *peer.ID {
	if knownBy == nil {
		logger.Debugf("Unexpected call to findRequestTargetWithUpdate() with nil knownBy argument")
		return nil
	}

	type peerWithIndex struct {
		peer  peer.ID
		index int
	}
	var target *peerWithIndex
	var prune []int

	for i, p := range *knownBy {
		// If we are already sending to that peer, skip for now
		if responseManager.isSendingTo(p) {
			continue
		}

		filter := peerAdvertised(candidateIdentifier, p)
		if filter == nil {
			prune = append(prune, i)
			continue
		}

		filter.MaskSeconded(props.unwantedMask.SecondedInGroup)
		filter.MaskValid(props.unwantedMask.ValidatedInGroup)
		if secondedAndSufficient(filter, props.backingThreshold) {
			target = &peerWithIndex{p, i}
			break
		}
	}

	for _, idx := range prune {
		*knownBy = slices.Delete(*knownBy, idx, idx+1)
	}

	if target != nil {
		idx := target.index - len(prune)
		*knownBy = slices.Delete(*knownBy, idx, idx+1)
		return &target.peer
	}

	return nil
}
