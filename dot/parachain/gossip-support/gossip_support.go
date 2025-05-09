// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package gossipsupport

import (
	"context"
	"fmt"
	networkbridgeevents "github.com/ChainSafe/gossamer/dot/parachain/network-bridge/events"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/pkg/errors"
	"math"
	"time"

	networkbridge "github.com/ChainSafe/gossamer/dot/parachain/network-bridge"
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
	subSystemToOverseer    chan<- any
	blockState             BlockState
	keystore               keystore.Keystore
	lastSessionIndex       *parachaintypes.SessionIndex
	minKnownSession        parachaintypes.SessionIndex
	lastFailure            *time.Time
	lastConnectionRequest  *time.Time
	failureStart           *time.Time
	resolvedAuthorities    map[parachaintypes.AuthorityDiscoveryID]map[multiaddr.Multiaddr]struct{}
	connectedAuthorities   map[parachaintypes.AuthorityDiscoveryID]parachaintypes.PeerID
	connectedPeers         map[parachaintypes.PeerID]map[parachaintypes.AuthorityDiscoveryID]struct{}
	authorityDiscovery     networkbridge.AuthorityDiscoveryService
	finalizedNeededSession *uint32

	// TODO: Metrics
}

func NewGossipSupport(
	ks keystore.Keystore,
	overseerChan chan<- any,
) *GossipSupport {
	return &GossipSupport{
		subSystemToOverseer: overseerChan,

		keystore:              ks,
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

		isNewSession := maybeNewSession != nil
		if isNewSession {
			logger.Debugf("new session detected for session %d", sessionIndex)
			gs.lastSessionIndex = &sessionIndex
		}

		// TODO: Connect to authorities from the past/present/future.

		if isNewSession {
			err := gs.buildTopologyForLastFinalizedIfNeeded(sessionIndex)
			if err != nil {
				logger.Warnf("failed to build topology for last finalized session %d, %s", sessionIndex, err.Error())
				return err
			}

			ourIndex, err := gs.getKeyIndexAndUpdateMetrics(sessionInfo)
			if err != nil {
				logger.Warnf("failed to get our index for session %d, %s", sessionIndex, err.Error())
				return err
			}

			gs.updateGossipTopology(ourIndex)
		}

		// authority discovery is just a cache so let's try every time we try to re-connect
		// if new authorities are present
		gs.updateAuthorityIDs(sessionInfo.DiscoveryKeys)
	}

	return nil
}

func (gs *GossipSupport) ProcessBlockFinalizedSignal(signal parachaintypes.BlockFinalizedSignal) error {
	//TODO implement #4507
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

func (gs *GossipSupport) processPeerConnectedEvent(event networkbridgeevents.PeerConnected) {
	//TODO implement in #4509

}

func (gs *GossipSupport) processPeerDisconnectedEvent(event networkbridgeevents.PeerDisconnected) {
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

func (gs *GossipSupport) buildTopologyForLastFinalizedIfNeeded(currentSessionIndex parachaintypes.SessionIndex) error {
	// TODO:
	return nil
}

func (gs *GossipSupport) getKeyIndexAndUpdateMetrics(SessionInfo *parachaintypes.SessionInfo) (uint, error) {
	// TODO:
	return 0, nil
}

func (gs *GossipSupport) updateGossipTopology(_ourIndex uint) {
	// TODO: implement in #4510
}

func (gs *GossipSupport) updateAuthorityIDs([]parachaintypes.AuthorityDiscoveryID) {
	// TODO:
}
