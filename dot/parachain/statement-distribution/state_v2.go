// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

//nolint:unused
package statementdistribution

import (
	"github.com/ChainSafe/gossamer/dot/parachain/grid"
	"github.com/ChainSafe/gossamer/dot/parachain/network-bridge/events"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	parachainutil "github.com/ChainSafe/gossamer/dot/parachain/util"
	validationprotocol "github.com/ChainSafe/gossamer/dot/parachain/validation-protocol"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/keystore"
)

// requestManager defines the interface that manages
// outgoing requests
type requestManager interface {
	removeByRelayParent(rp common.Hash)
}

// skipcq:SCC-U1000
type perRelayParentState struct {
	localValidator       *localValidatorState
	statementStore       *statementStore
	secondingLimit       uint
	session              parachaintypes.SessionIndex
	transposedClaimQueue parachaintypes.TransposedClaimQueue
	groupsPerPara        map[parachaintypes.ParaID][]parachaintypes.GroupIndex
	disabledValidators   map[parachaintypes.ValidatorIndex]struct{}
	assignmentsPerGroup  map[parachaintypes.GroupIndex][]parachaintypes.ParaID
}

func (p *perRelayParentState) activeValidatorState() *activeValidatorState {
	if p.localValidator != nil {
		return p.localValidator.active
	}

	return nil
}

// isDisabled returns `true` if the given validator is disabled in the context of the relay parent.
func (p *perRelayParentState) isDisabled(vIdx parachaintypes.ValidatorIndex) bool {
	_, ok := p.disabledValidators[vIdx]
	return ok
}

func (p *perRelayParentState) disabledBitmask(group []parachaintypes.ValidatorIndex) (parachaintypes.BitVec, error) {
	disableBm := make([]bool, len(group))
	for idx, v := range group {
		disableBm[idx] = p.isDisabled(v)
	}

	bm, err := parachaintypes.NewBitVec(disableBm)
	if err != nil {
		logger.Criticalf("cannot create a bitvec: %s", err.Error())
	}

	return bm, err
}

type localValidatorState struct {
	gridTracker *gridTracker
	active      *activeValidatorState // skipcq:SCC-U1000
}

// skipcq:SCC-U1000
type activeValidatorState struct {
	index          parachaintypes.ValidatorIndex
	groupIndex     parachaintypes.GroupIndex
	assignments    []parachaintypes.ParaID
	clusterTracker *clusterTracker // TODO: use cluster tracker implementation (#4713)
}

// skipcq:SCC-U1000
type perSessionState struct {
	sessionInfo *parachaintypes.SessionInfo
	groups      *groups
	authLookup  map[parachaintypes.AuthorityDiscoveryID]parachaintypes.ValidatorIndex
	gridView    *sessionTopologyView

	// when localValidator is nil means it is inactive
	localValidator     *parachaintypes.ValidatorIndex
	allowV2Descriptors bool
}

func newPerSessionState(
	sessionInfo *parachaintypes.SessionInfo,
	keystore keystore.Keystore,
	backingThreshold uint32,
	allowV2Descriptor bool,
) *perSessionState {
	authlookup := make(map[parachaintypes.AuthorityDiscoveryID]parachaintypes.ValidatorIndex)
	for idx, ad := range sessionInfo.DiscoveryKeys {
		authlookup[ad] = parachaintypes.ValidatorIndex(idx)
	}

	var localValidator *parachaintypes.ValidatorIndex
	validatorPk, validatorIdx := parachainutil.SigningKeyAndIndex(sessionInfo.Validators, keystore)
	if validatorPk != nil {
		localValidator = &validatorIdx
	}

	return &perSessionState{
		sessionInfo:        sessionInfo,
		groups:             newGroups(sessionInfo.ValidatorGroups, backingThreshold),
		authLookup:         authlookup,
		localValidator:     localValidator,
		allowV2Descriptors: allowV2Descriptor,
		gridView:           nil,
	}
}

// supplyTopology sets the topology for the session and updates the local validator
// Note: we use the local index rather than the `perSessionState.localValidator` as the
// former may be not nil when the latter is nil, due to the set of nodes in
// discovery being a superset of the active validators for consensus.
// skipcq:SCC-U1000
func (s *perSessionState) supplyTopology(topology *grid.SessionGridTopology, localIdx *parachaintypes.ValidatorIndex) {
	gridView, err := buildSessionTopology(
		s.sessionInfo.ValidatorGroups,
		topology,
		localIdx,
	)
	if err != nil {
		logger.Errorf("building sessionTopologyView for validator index %d: %s", localIdx, err)
		return
	}

	s.gridView = gridView

	logger.Infof(
		"Node uses the following topology indices: "+
			"index_in_gossip_topology: %d, index_in_parachain_auths: %d",
		localIdx, s.localValidator)
}

// isNotValidator returns `true` if local is neither active or inactive validator node.
//
// returns `false` if session topology is not known yet.
func (s *perSessionState) isNotValidator() bool {
	return s.gridView != nil && s.localValidator == nil
}

// skipcq:SCC-U1000
type peerState struct {
	view            parachaintypes.View
	protocolVersion validationprotocol.ValidationVersion // skipcq:SCC-U1000
	implicitView    map[common.Hash]struct{}
	discoveryIds    *map[parachaintypes.AuthorityDiscoveryID]struct{}
}

// updateView returns a vector of implicit relay-parents which weren't previously part of the view.
func (p *peerState) updateView(newView parachaintypes.View,
	localImplicitView parachainutil.ImplicitView) []common.Hash {
	nextImplicit := make(map[common.Hash]struct{})
	for _, h := range newView.Heads {
		for _, n := range localImplicitView.KnownAllowedRelayParentsUnder(h, nil) {
			nextImplicit[n] = struct{}{}
		}
	}

	freshImplicit := make([]common.Hash, 0, len(nextImplicit))
	for n := range nextImplicit {
		_, ok := p.implicitView[n]
		if !ok {
			freshImplicit = append(freshImplicit, n)
		}
	}

	p.view = newView
	p.implicitView = nextImplicit

	return freshImplicit
}

// reconcileActiveLeaf attempts to reconcile the view with new information about the
// implicit relay parents under an active leaf.
func (p *peerState) reconcileActiveLeaf(leafHash common.Hash, implicit []common.Hash) []common.Hash {
	if !p.view.Contains(leafHash) {
		return nil
	}

	v := make([]common.Hash, 0, len(implicit))
	for _, h := range implicit {
		if _, ok := p.implicitView[h]; !ok {
			p.implicitView[h] = struct{}{}
			v = append(v, h)
		}
	}

	return v
}

// knowsRelayParent returns true if the peer knows the relay-parent either implicitly or explicitly.
func (p *peerState) knowsRelayParent(relayParent common.Hash) bool {
	_, implicit := p.implicitView[relayParent]
	return implicit || p.view.Contains(relayParent)
}

// isAuthority returns true if the peer is an authority with the given AuthorityDiscoveryID.
func (p *peerState) isAuthority(authorityID parachaintypes.AuthorityDiscoveryID) bool {
	if p.discoveryIds == nil {
		return false
	}
	_, ok := (*p.discoveryIds)[authorityID]
	return ok
}

// iterKnownDiscoveryIDs returns a slice of known AuthorityDiscoveryIDs for the peer.
func (p *peerState) iterKnownDiscoveryIDs() []parachaintypes.AuthorityDiscoveryID {
	if p.discoveryIds == nil {
		return nil
	}
	ids := make([]parachaintypes.AuthorityDiscoveryID, 0, len(*p.discoveryIds))
	for id := range *p.discoveryIds {
		ids = append(ids, id)
	}
	return ids
}

type v2State struct {
	implicitView     parachainutil.ImplicitView
	candidates       *candidates
	perRelayParent   map[common.Hash]*perRelayParentState
	perSession       map[parachaintypes.SessionIndex]*perSessionState
	unusedTopologies map[parachaintypes.SessionIndex]events.NewGossipTopology
	peers            map[string]peerState
	keystore         keystore.Keystore
	authorities      map[parachaintypes.AuthorityDiscoveryID]string
	requestManager   requestManager // TODO: #4377
	responseManager  any            // TODO: #4378
}

func newV2State(ks keystore.Keystore, iv parachainutil.ImplicitView) *v2State {
	return &v2State{
		implicitView:     iv,
		candidates:       nil,
		perRelayParent:   map[common.Hash]*perRelayParentState{},
		perSession:       map[parachaintypes.SessionIndex]*perSessionState{},
		unusedTopologies: map[parachaintypes.SessionIndex]events.NewGossipTopology{},
		peers:            map[string]peerState{},
		keystore:         ks,
		authorities:      map[parachaintypes.AuthorityDiscoveryID]string{},
		requestManager:   nil,
		responseManager:  nil,
	}
}
