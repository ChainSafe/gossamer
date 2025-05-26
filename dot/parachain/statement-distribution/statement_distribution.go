// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statementdistribution

import (
	"context"
	"fmt"
	"slices"
	"time"

	parachainnetwork "github.com/ChainSafe/gossamer/dot/parachain/network"
	networkbridgemessages "github.com/ChainSafe/gossamer/dot/parachain/network-bridge/messages"
	prospectiveparachainsmessages "github.com/ChainSafe/gossamer/dot/parachain/prospective-parachains/messages"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	parachainutil "github.com/ChainSafe/gossamer/dot/parachain/util"
	validationprotocol "github.com/ChainSafe/gossamer/dot/parachain/validation-protocol"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/libp2p/go-libp2p/core/peer"
)

const (
	HypotheticalMembershipTimeout = 2 * time.Second
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "parachain-statement-distribution"))

type BlockState interface {
	GetHeader(hash common.Hash) (header *types.Header, err error)
	GetRuntime(blockHash common.Hash) (instance runtime.Instance, err error)
}

type StatementDistribution struct {
	blockState          BlockState
	SubSystemToOverseer chan<- any
	state               *v2State
}

func New(overseerChan chan<- any, ks keystore.Keystore, blockState parachainutil.BlockState) *StatementDistribution {
	return &StatementDistribution{
		SubSystemToOverseer: overseerChan,
		blockState:          blockState,
		state:               newV2State(ks, parachainutil.NewBackingImplicitView(blockState, nil)),
	}
}

// Run just receives the ctx and a channel from the overseer to subsystem
func (s *StatementDistribution) Run(ctx context.Context, overseerToSubSystem <-chan any) {
	// Inside the method Run, we spawn a goroutine to handle network incoming requests
	// TODO: https://github.com/ChainSafe/gossamer/issues/4285
	responderCh := make(chan any, 1)
	go taskResponder(responderCh)

	// Timer for reputation aggregator trigger
	reputationDelay := time.NewTicker(parachainutil.ReputationChangeInterval) // Adjust the duration as needed
	defer reputationDelay.Stop()

	for {
		message := s.awaitMessageFrom(overseerToSubSystem, responderCh, reputationDelay.C)

		switch innerMessage := message.(type) {
		case *reputationChangeMessage:
			logger.Info("Reputation change triggered.")
		case *overseerMessage:
			shouldStop, err := s.handleSubsystemMessage(innerMessage.inner)
			if err != nil {
				logger.Errorf("handling subsystem message: %s", err.Error())
			}

			if shouldStop {
				logger.Warn("handling subsystem message: should stop statement distribution")
				break
			}
		default:
			logger.Warn("Unhandled message type: " + fmt.Sprintf("%v", innerMessage))
		}
	}
}

func (s *StatementDistribution) handleSubsystemMessage(overseerMessage any) (bool, error) {
	switch message := overseerMessage.(type) {
	case parachaintypes.ActiveLeavesUpdateSignal:
		if message.Activated != nil {
			if err := s.handleActiveLeavesUpdate(message.Activated); err != nil {
				return false, fmt.Errorf("handling active leaves update: %w", err)
			}
		}
		s.handleDeactivatedLeaves(message.Deactivated)

	case parachaintypes.Conclude:
		return true, nil
	}

	return false, nil
}

// Send a peer, apparently just becoming aware of a relay-parent, all messages
// concerning that relay-parent.
//
// In particular, we send all statements pertaining to our common cluster,
// as well as all manifests, acknowledgements, or other grid statements.
//
// Note that due to the way we handle views, our knowledge of peers' relay parents
// may "oscillate" with relay parents repeatedly leaving and entering the
// view of a peer based on the implicit view of active leaves.
//
// This function is designed to be cheap and not to send duplicate messages in repeated
// cases.
func (s *StatementDistribution) sendPeerMessagesForRelayParent(pid string, rp common.Hash) {
	peerData, ok := s.state.peers[pid]
	if !ok {
		return
	}

	rpState, ok := s.state.perRelayParent[rp]
	if !ok {
		return
	}

	perSessionState, ok := s.state.perSession[rpState.session]
	if !ok {
		return
	}

	for _, vID := range peerData.iterKnownDiscoveryIDs() {
		vIndex, ok := perSessionState.authLookup[vID]
		if !ok {
			continue
		}

		active := rpState.activeValidatorState()
		if active != nil {
			s.sendPendingClusterStatements(rp,
				peer.ID(pid), peerData.protocolVersion,
				vIndex,
				active.clusterTracker,
				s.state.candidates,
				rpState.statementStore,
			)
		}

		s.sendPendingGridStatements(rp,
			peer.ID(pid), peerData.protocolVersion,
			vIndex,
			perSessionState.groups,
			rpState,
			s.state.candidates,
		)
	}
}

func (s *StatementDistribution) fragmentChainUpdateInner(rp *common.Hash,
	requiredParentHash *common.Hash, requiredParentParaID *parachaintypes.ParaID,
	knowHypotheticals *[]parachaintypes.HypotheticalCandidate) {

	// 1. get hypothetical candidates
	var hypotheticals []parachaintypes.HypotheticalCandidate

	if knowHypotheticals != nil {
		hypotheticals = *knowHypotheticals
	} else {
		s.state.candidates.frontierHypotheticals(requiredParentHash, requiredParentParaID)
	}

	// 2. find out which are in the frontier
	response := make(chan []*prospectiveparachainsmessages.HypotheticalMembershipResponseItem)
	hypotheticalMembershipRequest := prospectiveparachainsmessages.GetHypotheticalMembership{
		Candidates:               hypotheticals,
		FragmentChainRelayParent: rp,
		Response:                 response,
	}

	s.SubSystemToOverseer <- hypotheticalMembershipRequest

	var candidateMemberships []*prospectiveparachainsmessages.HypotheticalMembershipResponseItem
	select {
	case <-time.After(HypotheticalMembershipTimeout):
		logger.Warnf("timed out waiting for hypothetical membership response for relay-parent %s", rp.String())
		return
	case resp := <-response:
		candidateMemberships = resp

	}

	// 3. note that they are importable under a given leaf hash.
	for _, item := range candidateMemberships {
		// skip parablocks which aren't potential candidates
		if len(item.HypotheticalMembership) == 0 {
			continue
		}

		for _, leafHash := range item.HypotheticalMembership {
			s.state.candidates.noteImportableUnder(item.HypotheticalCandidate, leafHash)
		}

		// 4. for confirmed candidates, send all statements which are new to backing.
		if complete, ok := item.HypotheticalCandidate.(*parachaintypes.HypotheticalCandidateComplete); ok {
			confirmedCandidate := s.state.candidates.getConfirmed(complete.ClaimedCandidateHash)
			perRelayParentState, ok := s.state.perRelayParent[complete.CommittedCandidateReceipt.Descriptor.RelayParent]
			if confirmedCandidate == nil || !ok {
				continue
			}

			groupIndex := confirmedCandidate.assignedGroup
			perSessionState, ok := s.state.perSession[perRelayParentState.session]
			if !ok {
				continue
			}

			// Sanity check if group_index is valid for this para at relay parent.
			expectedGroups, ok := perRelayParentState.groupsPerPara[complete.CommittedCandidateReceipt.Descriptor.ParaID]
			if !ok {
				continue
			}

			if !slices.Contains(expectedGroups, groupIndex) {
				logger.Warnf("group index %d not found for para %d at relay parent %s",
					groupIndex, complete.CommittedCandidateReceipt.Descriptor.ParaID, rp.String())
				continue
			}

			s.sendBackingFreshStatements(
				complete.ClaimedCandidateHash,
				confirmedCandidate.assignedGroup,
				complete.CommittedCandidateReceipt.Descriptor.RelayParent,
				perRelayParentState,
				confirmedCandidate,
				perSessionState,
			)
		}
	}

	panic("unimplemented")
}

// Send a peer all pending cluster statements for a relay parent.
func (s *StatementDistribution) sendPendingClusterStatements(rp common.Hash,
	peerID peer.ID, validationVersion parachainnetwork.ValidationVersion,
	peerValidatorID parachaintypes.ValidatorIndex,
	clusterTracker groupTracker,
	candidates candidatesTracker,
	statementStore statementStore,
) {
	pendingStmts := clusterTracker.pendingStatementsFor(peerValidatorID)
	for _, stmt := range pendingStmts {
		if !candidates.isConfirmed(stmt.compact.CandidateHash()) {
			continue
		}

		msg := pendingStatementNetworkMessage(statementStore, rp, peerID, validationVersion, stmt)
		if msg != nil {
			clusterTracker.noteSend(peerValidatorID, stmt.validadorIdx, stmt.compact)
			// TODO: create a SendValidationMessages to send a batch of messages
			s.SubSystemToOverseer <- msg
		}
	}
}

func pendingStatementNetworkMessage(
	stmtStore statementStore,
	rp common.Hash,
	peerID peer.ID, validationVersion parachainnetwork.ValidationVersion,
	pending pendingStmt,
) *networkbridgemessages.SendValidationMessage {
	if validationVersion == parachainnetwork.ValidationVersionV3 {
		signed := stmtStore.validatorStatement(pending)
		if signed == nil {
			return nil
		}

		sdmV3 := validationprotocol.NewStatementDistributionMessageV3()
		err := sdmV3.SetValue(validationprotocol.StatementV3{
			Hash:                     rp,
			UncheckedSignedStatement: parachaintypes.UncheckedSignedCompactStatement(*signed),
		})
		if err != nil {
			panic(fmt.Sprintf("unexpected error setting value in StatementDistributionMessageV3: %s", err))
		}

		// TODO: this will panic as validation protocol does not support V3 yet
		vp := validationprotocol.NewValidationProtocolVDT()
		err = vp.SetValue(sdmV3)
		if err != nil {
			panic(fmt.Sprintf("unexpected error setting value in NewValidationProtocolVDT: %s", err))
		}

		return &networkbridgemessages.SendValidationMessage{
			To:                        []peer.ID{peerID},
			ValidationProtocolMessage: vp,
		}
	}

	return nil
}

// Send a peer all pending grid messages / acknowledgements / follow up statements
// upon learning about a new relay parent.
func (s *StatementDistribution) sendPendingGridStatements(relayParent common.Hash,
	peerID peer.ID, validationVersion parachainnetwork.ValidationVersion,
	peerValidatorID parachaintypes.ValidatorIndex,
	groups *groups,
	rpState *perRelayParentState,
	candidates candidatesTracker) {
	panic("unimplemented issue #4730")
}

// Send backing fresh statements. This should only be performed on importable & confirmed candidates
func (s *StatementDistribution) sendBackingFreshStatements(candidateHash parachaintypes.CandidateHash,
	groupIndex parachaintypes.GroupIndex,
	relayParent common.Hash,
	rpState *perRelayParentState,
	confirmed *confirmedCandidate,
	perSessionState *perSessionState) {
	panic("unimplemented issue #4419")
}

// TODO: https://github.com/ChainSafe/gossamer/issues/4285
func taskResponder(responderCh chan any) {}
