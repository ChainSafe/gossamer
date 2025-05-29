// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package gossipsupport

import (
	"context"
	"fmt"
	networkbridge "github.com/ChainSafe/gossamer/dot/parachain/network-bridge"
	networkbridgeevents "github.com/ChainSafe/gossamer/dot/parachain/network-bridge/events"
	networkbridgemessages "github.com/ChainSafe/gossamer/dot/parachain/network-bridge/messages"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pkg/errors"
	"math"
	"reflect"
	"time"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/multiformats/go-multiaddr"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "parachain-gossip-support"))

const (
	// LowConnectivityWarnDelay indicates the duration after which we consider low connectivity a problem
	LowConnectivityWarnDelay = 600 * time.Second
	// LowConnectivityWarnThreshold means if connectivity is lower than this in percent, issue warning in logs
	LowConnectivityWarnThreshold = 90
	// BackoffDuration indicates how much time should we wait to reissue a connection request since the last
	// authority discovery resolution failure
	BackoffDuration = 5
	// TryReResolveAuthorities indicates the authority discovery queries runs every time minutes
	TryReResolveAuthorities = 300
)

type leafSession struct {
	currentIndex parachaintypes.SessionIndex
	Leaf         common.Hash
}

type BlockState interface {
	GetRuntime(blockHash common.Hash) (instance runtime.Instance, err error)
}

// GossipSupport is the parachain subsystem that is responsible for keeping track of session changes and issuing a
// connection request to all validators in the next, current and a few past sessions if we are a validator
// in these sessions.
type GossipSupport struct {
	subSystemToOverseer chan<- any
	blockState          BlockState

	keystore         keystore.Keystore
	lastSessionIndex *parachaintypes.SessionIndex
	// The minimum known session we build the topology for.
	minKnownSession parachaintypes.SessionIndex
	// A timestamp if we failed to resolve
	// at least a third of authorities the last time.
	// nil otherwise
	lastFailure *time.Time
	// Validators can restart during a session, so if they change
	// their PeerID, we will connect to them in the best case after
	// a session, so we need to try more often to resolved peers and
	// reconnect to them. The authority_discovery queries runs every ten
	// minutes, so we can't detect changes in the address more often
	// than that.
	lastConnectionRequest *time.Time
	// First time we did not reach our connectivity threshold.
	// This is the time of the first failed attempt to connect to >2/3 of all validators in a potential sequence of
	// failed attempts. It will be cleared once we reached >2/3 connectivity.
	failureStart *time.Time
	// Successfully resolved connections
	// waiting for actual connection.
	resolvedAuthorities map[parachaintypes.AuthorityDiscoveryID]map[multiaddr.Multiaddr]struct{}
	// Actually connected authorities.
	connectedAuthorities map[parachaintypes.AuthorityDiscoveryID]parachaintypes.PeerID
	// By `PeerId`
	// Needed for efficient handling of disconnect events.
	connectedPeers map[parachaintypes.PeerID]map[parachaintypes.AuthorityDiscoveryID]struct{}
	// Authority discovery service.
	authorityDiscovery networkbridge.AuthorityDiscoveryService
	// The oldest session we need to build a topology for because
	// the finalized blocks are from a session we haven't built a topology for.
	finalizedNeededSession *parachaintypes.SessionIndex
}

func NewGossipSupport(
	ks keystore.Keystore,
	overseerChan chan<- any,
	blockState BlockState,
) *GossipSupport {
	return &GossipSupport{
		subSystemToOverseer: overseerChan,
		blockState:          blockState,
		keystore:            ks,

		lastSessionIndex:      nil,
		minKnownSession:       math.MaxUint32,
		lastFailure:           nil,
		lastConnectionRequest: nil,
		failureStart:          nil,
		resolvedAuthorities:   make(map[parachaintypes.AuthorityDiscoveryID]map[multiaddr.Multiaddr]struct{}),
		connectedAuthorities:  make(map[parachaintypes.AuthorityDiscoveryID]parachaintypes.PeerID),
		connectedPeers:        make(map[parachaintypes.PeerID]map[parachaintypes.AuthorityDiscoveryID]struct{}),
		// TODO: authorityDiscovery needs to be passed in from param, however; AuthorityDiscoveryService is not
		// implemented yet on overseer level.
		// NetworkBridgeSender, NetworkBridgeReceiver, DisputeDistribution subsyestems are also depending on it.
		authorityDiscovery:     nil,
		finalizedNeededSession: nil,
	}
}

func (gs *GossipSupport) Name() parachaintypes.SubSystemName {
	return parachaintypes.GossipSupport
}

func (gs *GossipSupport) ProcessActiveLeavesUpdateSignal(signal parachaintypes.ActiveLeavesUpdateSignal) error {
	logger.Trace("Process ActiveLeavesUpdateSignal")

	leaf := signal.Activated.Hash

	rt, err := gs.blockState.GetRuntime(leaf)
	if err != nil {
		return err
	}

	currentIndex, err := rt.ParachainHostSessionIndexForChild()
	if err != nil {
		return err
	}

	sinceFailure := time.Duration(0)
	if gs.lastFailure != nil {
		sinceFailure = time.Now().Sub(*gs.lastFailure)
	}

	sinceLastReconnect := time.Duration(0)
	if gs.lastConnectionRequest != nil {
		sinceLastReconnect = time.Now().Sub(*gs.lastConnectionRequest)
	}

	forceRequest := sinceFailure >= BackoffDuration
	reResolveAuthorities := sinceLastReconnect >= TryReResolveAuthorities
	ls := &leafSession{
		currentIndex,
		leaf,
	}

	maybeNewSession := ls
	if gs.lastSessionIndex != nil && currentIndex <= *gs.lastSessionIndex {
		maybeNewSession = nil
	}

	maybeIssueConnection := maybeNewSession
	if forceRequest || reResolveAuthorities {
		maybeIssueConnection = ls
	}

	if maybeIssueConnection != nil {
		sessionIndex := maybeIssueConnection.currentIndex
		relayParent := maybeIssueConnection.Leaf

		sessionInfo, err := rt.ParachainHostSessionInfo(sessionIndex)
		if err != nil {
			logger.Warnf("failed to get session info for session %d", sessionIndex)
			return err
		}

		// Note: we only update `last_session_index` once we've successfully gotten the `SessionInfo`.
		isNewSession := maybeNewSession != nil
		if isNewSession {
			logger.Debugf("new session detected for session %d", sessionIndex)
			gs.lastSessionIndex = &sessionIndex
		}

		// Connect to authorities from the past/present/future.
		connections, err := authoritiesPastPresentFuture(rt)
		if err != nil {
			return err
		}

		now := time.Now()
		gs.lastConnectionRequest = &now

		// Remove all of our locally controlled validator indices, so we don't connect to ourselves
		filteredConnections, removedCounter := removeAllControlled(gs.keystore, connections)
		if removedCounter != 0 {
			connections = filteredConnections
		} else {
			// If we control none of them, issue an empty connection request
			// to clean up all connections.
			connections = make([]parachaintypes.AuthorityDiscoveryID, 0)
		}

		if forceRequest || isNewSession {
			gs.issueConnectionRequest(connections)
		} else {
			gs.issueConnectionRequestToChanged(connections)
		}

		if isNewSession {
			err := gs.buildTopologyForLastFinalizedIfNeeded(sessionIndex, rt)
			if err != nil {
				logger.Warnf("failed to build topology for last finalized session %d, %s", sessionIndex, err.Error())
				return err
			}

			// Gossip topology is only relevant for authorities in the current session.
			ourIndex, err := gs.getKeyIndexAndUpdateMetrics(sessionInfo)
			if err != nil {
				logger.Warnf("failed to get our index for session %d, %s", sessionIndex, err.Error())
				return err
			}
			err = gs.updateGossipTopology(ourIndex, relayParent)
			if err != nil {
				return err
			}
		}

		// authority discovery is just a cache so let's try every time we try to re-connect
		// if new authorities are present
		gs.updateAuthorityIDs(sessionInfo.DiscoveryKeys)
	}

	return nil
}

func (gs *GossipSupport) ProcessBlockFinalizedSignal(signal parachaintypes.BlockFinalizedSignal) error {
	rt, err := gs.blockState.GetRuntime(signal.Hash)
	if err != nil {
		return err
	}

	if gs.lastSessionIndex != nil {
		if err := gs.buildTopologyForLastFinalizedIfNeeded(*gs.lastSessionIndex, rt); err != nil {
			logger.Warnf("Failed to build topology for last finalized session: %s", err.Error())
			return err
		}
	}
	return nil
}

func (gs *GossipSupport) Stop() { logger.Tracef("Stopping GossipSupport subsystem") }

// Run starts the GossipSupport subsystem
func (gs *GossipSupport) Run(ctx context.Context, overseerToSubsystem <-chan any) {
	checkConnectivityTicker := time.NewTicker(LowConnectivityWarnDelay)
	for {
		select {
		case <-checkConnectivityTicker.C:
			gs.checkConnectivity()
		case msg := <-overseerToSubsystem:
			err := gs.processMessage(msg)
			if err != nil {
				logger.Errorf("processing message: %s", err.Error())
			}
		case <-ctx.Done():
			if err := ctx.Err(); err != nil && !errors.Is(err, context.Canceled) {
				logger.Errorf("ctx error: %s\n", err)
			}
			return
		}
	}
}

func (gs *GossipSupport) processMessage(msg any) error {
	switch msg := msg.(type) {
	case parachaintypes.ActiveLeavesUpdateSignal:
		err := gs.ProcessActiveLeavesUpdateSignal(msg)
		if err != nil {
			return fmt.Errorf("processing active leaves update signal: %w", err)
		}
	case networkbridgeevents.PeerConnected:
		gs.processPeerConnectedEvent(msg)
	case networkbridgeevents.PeerDisconnected:
		gs.processPeerDisconnectedEvent(msg)
	case parachaintypes.BlockFinalizedSignal:
		return gs.ProcessBlockFinalizedSignal(msg)
	default:
		return fmt.Errorf("%w: %T", parachaintypes.ErrUnknownOverseerMessage, msg)
	}
	return nil
}

func (gs *GossipSupport) processPeerConnectedEvent(_event networkbridgeevents.PeerConnected) {
	//TODO implement in #4509
}

func (gs *GossipSupport) processPeerDisconnectedEvent(_event networkbridgeevents.PeerDisconnected) {
	//TODO implement in #4509
}

// checkConnectivity checks connectivity and report on it in logs.
func (gs *GossipSupport) checkConnectivity() {
	absoluteConnected := len(gs.connectedAuthorities)
	absoluteResolved := len(gs.resolvedAuthorities)

	connectedRatio := 100
	if absoluteResolved != 0 {
		connectedRatio = 100 * absoluteConnected / absoluteResolved
	}

	unconnectedAuthorities := make(map[parachaintypes.AuthorityDiscoveryID]map[multiaddr.Multiaddr]struct{})
	for authID, v := range gs.resolvedAuthorities {
		if _, ok := gs.connectedAuthorities[authID]; !ok {
			unconnectedAuthorities[authID] = v
		}
	}

	if connectedRatio <= LowConnectivityWarnThreshold {
		logger.Debugf("connectivity seems low, we are only connected to %d of available validators"+
			" (see debug logs for details)", connectedRatio)
	}

	logger.Debugf("connectivity Report: \n"+
		"connected ratio: %d, \n"+
		"absolute connected: %d, \n"+
		"absolute resolved: %d, \n"+
		"unconnected authorities: %+v \n", connectedRatio, absoluteConnected, absoluteResolved, unconnectedAuthorities)
}

// buildTopologyForLastFinalizedIfNeeded builds the gossip topology for the session of the last finalized block
// if we haven't built one
func (gs *GossipSupport) buildTopologyForLastFinalizedIfNeeded(
	currentSessionIndex parachaintypes.SessionIndex,
	rt runtime.Instance,
) error {
	if currentSessionIndex < gs.minKnownSession {
		gs.minKnownSession = currentSessionIndex
	}

	if gs.finalizedNeededSession == nil || (gs.finalizedNeededSession != nil && *gs.finalizedNeededSession < gs.minKnownSession) {
		finalizedBlock, err := rt.FinalizeBlock()
		if err != nil {
			return err
		}

		finalizedSessionIndex, err := rt.ParachainHostSessionIndexForChild()
		if err != nil {
			return err
		}

		if finalizedSessionIndex < gs.minKnownSession &&
			gs.finalizedNeededSession != nil && *gs.finalizedNeededSession != finalizedSessionIndex {
			logger.Debugf("Building topology for finalized block session: block number: %d, block hash: %s, "+
				"session index: %d", finalizedBlock.Number, finalizedBlock.Hash(), finalizedSessionIndex)

			finalizedSessionInfo, err := rt.ParachainHostSessionInfo(finalizedSessionIndex)
			if err != nil {
				return err
			}

			ourIndex, err := gs.getKeyIndexAndUpdateMetrics(finalizedSessionInfo)
			if err != nil {
				return err
			}

			err = gs.updateGossipTopology(ourIndex, finalizedBlock.Hash())
			if err != nil {
				return err
			}
		}

		gs.finalizedNeededSession = &finalizedSessionIndex
	}

	return nil
}

// getKeyIndexAndUpdateMetrics checks if the node is an authority and also updates `polkadot_node_is_authority` and
// `polkadot_node_is_parachain_validator` metrics accordingly(We currently don't have metrics implemented,
// corresponding logic will be added later)
// On success, returns the index of our keys in `session_info.discovery_keys`.
func (gs *GossipSupport) getKeyIndexAndUpdateMetrics(SessionInfo *parachaintypes.SessionInfo) (uint, error) {
	authCheckResult, err := ensureIamAnAuthority(gs.keystore, SessionInfo.DiscoveryKeys)
	if err != nil {
		logger.Tracef("we are no longer an authority")
		return authCheckResult, err
	}

	logger.Tracef("we are now an authority")

	// The subset of authorities participating in parachain consensus.
	parachainValidatorsThisSession := len(SessionInfo.Validators)

	if authCheckResult < uint(parachainValidatorsThisSession) {
		logger.Tracef("we are now a parachain validator")
	} else {
		logger.Tracef("we are no longer a parachain validator")
	}

	return authCheckResult, err
}

func (gs *GossipSupport) updateGossipTopology(_ourIndex uint, _relayParent common.Hash) error {
	// TODO: implement in #4510
	return nil
}

func (gs *GossipSupport) updateAuthorityIDs(authorities []parachaintypes.AuthorityDiscoveryID) {
	authorityIDs := make(map[parachaintypes.PeerID]map[parachaintypes.AuthorityDiscoveryID]struct{})

	for _, authority := range authorities {
		peerIDs := make([]parachaintypes.PeerID, 0)
		addrs := gs.authorityDiscovery.GetAddressesByAuthorityID(authority)
		for addr := range addrs {
			_, peerID := peer.SplitAddr(addr)
			peerIDs = append(peerIDs, parachaintypes.PeerID(peerID))
		}

		logger.Tracef("resolved to peer ids")

		for _, peerID := range peerIDs {
			authorityIDs[peerID] = map[parachaintypes.AuthorityDiscoveryID]struct{}{
				authority: {},
			}
		}
	}

	// peer was authority and now isn't
	for peerID, current := range gs.connectedPeers {
		// empty -> nonempty is handled in the next loop
		_, ok := authorityIDs[peerID]
		if len(current) != 0 && !ok {
			gs.subSystemToOverseer <- networkbridgemessages.UpdateAuthorityIDs{
				PeerID:                peer.ID(peerID),
				AuthorityDiscoveryIDs: nil,
			}

			for c := range current {
				delete(gs.connectedAuthorities, c)
			}
		}
	}

	// peer has new authority set.
	for peerID, newOne := range authorityIDs {
		if p, ok := gs.connectedPeers[peerID]; ok {
			if reflect.DeepEqual(newOne, p) {
				delete(gs.connectedPeers, peerID)
			}

			updatedAuthDiscovery := make([]parachaintypes.AuthorityDiscoveryID, 0)
			for c := range newOne {
				updatedAuthDiscovery = append(updatedAuthDiscovery, c)
			}
			gs.subSystemToOverseer <- networkbridgemessages.UpdateAuthorityIDs{
				PeerID:                peer.ID(peerID),
				AuthorityDiscoveryIDs: updatedAuthDiscovery,
			}

			for _, AuthorityDiscoveryIDs := range gs.connectedPeers {
				for p := range AuthorityDiscoveryIDs {
					delete(gs.connectedAuthorities, p)
				}
			}

			for a := range newOne {
				gs.connectedAuthorities[a] = peerID
			}

			gs.connectedPeers[peerID] = newOne
		}
	}
}

// authoritiesPastPresentFuture gets the authorities of the past, present, and future.
func authoritiesPastPresentFuture(rt runtime.Instance) ([]parachaintypes.AuthorityDiscoveryID, error) {
	authorities, err := rt.GrandpaAuthorities()
	if err != nil {
		return nil, err
	}

	logger.Debugf("Determined past/present/future authorities with size: %d", len(authorities))

	authoritiesIDs := make([]parachaintypes.AuthorityDiscoveryID, len(authorities))
	for i, authority := range authorities {
		authoritiesIDs[i] = parachaintypes.AuthorityDiscoveryID(authority.Key.Encode())
	}

	return authoritiesIDs, nil
}

// removeAllControlled filters out all controlled keys in the given set. Returns the number of keys removed along with
// the result of filtered authorities
func removeAllControlled(
	ks keystore.Keystore,
	authorities []parachaintypes.AuthorityDiscoveryID,
) ([]parachaintypes.AuthorityDiscoveryID, uint) {
	var removeCounter uint
	resultAuthorities := make([]parachaintypes.AuthorityDiscoveryID, 0)
	for _, key := range authorities {
		publicKey, err := sr25519.NewPublicKey(key[:])
		if err != nil {
			continue
		}
		authKey := ks.GetKeypair(publicKey)
		if authKey != nil {
			removeCounter++
		} else {
			resultAuthorities = append(resultAuthorities, key)
		}
	}

	return resultAuthorities, removeCounter
}

func (gs *GossipSupport) issueConnectionRequest(authorities []parachaintypes.AuthorityDiscoveryID) {
	num := len(authorities)
	validatorAddrs, resolved, failures := gs.resolveAuthorities(authorities)
	gs.resolvedAuthorities = resolved

	logger.Debugf("Issuing a connection request: %d", num)

	gs.subSystemToOverseer <- networkbridgemessages.ConnectTOResolvedValidators{
		ValidatorAddrs: validatorAddrs,
		PeerSet:        networkbridgemessages.ValidationProtocol,
	}

	// issue another request for the same session
	// if at least a third of the authorities were not resolved.
	if num != 0 && 3*failures >= uint(num) {
		timestamp := time.Now()
		if gs.failureStart == nil {
			gs.failureStart = &timestamp
		} else {
			first := *gs.failureStart
			if first.Sub(time.Now()) >= LowConnectivityWarnDelay {
				logger.Warnf("Low connectivity - authority lookup failed for too many validators.")
			}
			logger.Debugf("Low connectivity (due to authority lookup failures) - expected on startup.")
		}
	} else {
		gs.lastFailure = nil
		gs.failureStart = nil
	}
}

func (gs *GossipSupport) issueConnectionRequestToChanged(authorities []parachaintypes.AuthorityDiscoveryID) {
	_, resolved, _ := gs.resolveAuthorities(authorities)

	changed := make(map[multiaddr.Multiaddr]struct{})

	for authority, newAddresses := range resolved {
		newPeerIDs := make(map[peer.ID]struct{})
		for addr := range newAddresses {
			_, peerID := peer.SplitAddr(addr)
			newPeerIDs[peerID] = struct{}{}
		}

		oldAddresses := gs.resolvedAuthorities[authority]
		if oldAddresses != nil {
			oldPeerIDs := make(map[peer.ID]struct{})
			for addr := range oldAddresses {
				_, peerID := peer.SplitAddr(addr)
				oldPeerIDs[peerID] = struct{}{}
			}
			if !isSuperSet(oldPeerIDs, newPeerIDs) {
				changed = newAddresses
			}
		} else {
			changed = newAddresses
		}
	}

	logger.Debugf("Issuing a connection request to changed validators")

	if len(changed) == 0 {
		gs.resolvedAuthorities = resolved

		gs.subSystemToOverseer <- networkbridgemessages.AddToResolvedValidators{
			ValidatorAddrs: changed,
			PeerSet:        networkbridgemessages.ValidationProtocol,
		}
	}
}

// resolveAuthorities parses the validator address and resolved authorities along with the failure time of parsing
func (gs *GossipSupport) resolveAuthorities(
	authorities []parachaintypes.AuthorityDiscoveryID,
) (map[multiaddr.Multiaddr]struct{}, map[parachaintypes.AuthorityDiscoveryID]map[multiaddr.Multiaddr]struct{}, uint) {
	validatorAddrs := make(map[multiaddr.Multiaddr]struct{})
	resolved := make(map[parachaintypes.AuthorityDiscoveryID]map[multiaddr.Multiaddr]struct{}, len(authorities))
	var failures uint

	for _, authority := range authorities {
		addrs := gs.authorityDiscovery.GetAddressesByAuthorityID(authority)

		if addrs != nil {
			validatorAddrs = addrs
			resolved[authority] = addrs
		} else {
			failures++
			logger.Debugf("Couldn't resolve addresses of authority: %v", authority)
		}
	}

	return validatorAddrs, resolved, failures
}

// isSuperSet returns true if the superset is a superset of subset
func isSuperSet[K, V comparable](superset, subset map[K]V) bool {
	for k, v := range subset {
		if val, ok := superset[k]; !ok || val != v {
			return false
		}
	}
	return true
}

// ensureIamAnAuthority return an error if we're not a validator in the given set (do not have keys). Otherwise,
// returns the index of our keys in authorities.
func ensureIamAnAuthority(ks keystore.Keystore, authorities []parachaintypes.AuthorityDiscoveryID) (uint, error) {
	for i, authority := range authorities {
		publicKey, err := sr25519.NewPublicKey(authority[:])
		if err != nil {
			continue
		}
		authKey := ks.GetKeypair(publicKey)
		if authKey != nil {
			return uint(i), nil
		}
	}

	return 0, fmt.Errorf("node is not a validator")
}
