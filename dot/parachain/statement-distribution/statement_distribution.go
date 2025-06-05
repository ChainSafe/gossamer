// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statementdistribution

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ChainSafe/gossamer/dot/parachain/backing"
	networkbridgemessages "github.com/ChainSafe/gossamer/dot/parachain/network-bridge/messages"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	parachainutil "github.com/ChainSafe/gossamer/dot/parachain/util"
	validationprotocol "github.com/ChainSafe/gossamer/dot/parachain/validation-protocol"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/libp2p/go-libp2p/core/peer"
)

var (
	errEncodedStatementsDoNotMatch = errors.New("encoded statements do not match")
	errUnkownLocalValidator        = errors.New("unknown local validator")
	errEmptyGroup                  = errors.New("group of validators empty")
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "parachain-statement-distribution"))

type StatementDistribution struct {
	SubSystemToOverseer chan<- any
}

type MuxedMessage interface {
	isMuxedMessage()
}

type overseerMessage struct {
	inner any
}

func (*overseerMessage) isMuxedMessage() {}

type responderMessage struct {
	inner any // should be replaced with AttestedCandidateRequest type
}

func (*responderMessage) isMuxedMessage() {}

type reputationChangeMessage struct{}

func (*reputationChangeMessage) isMuxedMessage() {}

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
		default:
			logger.Warn("Unhandled message type: " + fmt.Sprintf("%v", innerMessage))
		}
	}
}

func taskResponder(responderCh chan any) {}

// awaitMessageFrom waits for messages from either the overseerToSubSystem, responderCh, or reputationDelay
func (s *StatementDistribution) awaitMessageFrom(
	overseerToSubSystem <-chan any,
	responderCh chan any,
	reputationDelay <-chan time.Time,
) MuxedMessage {
	select {
	case msg := <-overseerToSubSystem:
		return &overseerMessage{inner: msg}
	case msg := <-responderCh:
		return &responderMessage{inner: msg}
	case <-reputationDelay:
		return &reputationChangeMessage{}
	}
}

// sendPendingGridMessages sends pending grid messages / acknowledgements / follow up statements
// upon learning about a new relay parent.
func (s *StatementDistribution) sendPendingGridMessages(
	rp common.Hash,
	peerID peer.ID, //nolint:unparam
	validationVersion validationprotocol.ValidationVersion, //nolint:unparam
	peerValidatorID parachaintypes.ValidatorIndex,
	groups *groups,
	rpState *perRelayParentState,
	candidates candidatesStore,
) error {
	if rpState.localValidator == nil {
		return errUnkownLocalValidator
	}

	pendingManifests := rpState.localValidator.gridTracker.pendingManifestsFor(peerValidatorID)

	var messages []*networkbridgemessages.SendValidationMessage

	for candidateHash, kind := range pendingManifests {
		confirmed, ok := candidates.getConfirmed(candidateHash)
		if !ok {
			continue // sanity
		}

		groupIndex := confirmed.assignedGroup

		// polkadot-sdk they return an option, and if the group
		// does not exists we get None, here the `get` method
		// returns nil, and `len(nil)` is 0 and we return.
		groupSize := len(groups.get(groupIndex))
		if groupSize == 0 {
			return errEmptyGroup
		}

		localKnowledge, err := localKnowledgeFilter(
			groupSize,
			groupIndex,
			candidateHash,
			rpState.statementStore,
		)
		if err != nil {
			return fmt.Errorf("local knowledge filter: %w", err)
		}

		switch kind {
		case full:
			manifest := validationprotocol.BackedCandidateManifest{
				RelayParent:        rp,
				CandidateHash:      candidateHash,
				GroupIndex:         groupIndex,
				ParaID:             confirmed.receipt.Descriptor.ParaID,
				ParentHeadDataHash: confirmed.parentHash,
				StatementKnowledge: *localKnowledge,
			}

			grid := rpState.localValidator.gridTracker
			grid.manifestSentTo(*groups, peerValidatorID, candidateHash, *localKnowledge)

			if validationVersion == validationprotocol.ValidationVersionV3 {
				sdm := validationprotocol.NewStatementDistributionMessage()
				err := sdm.SetValue(manifest)
				if err != nil {
					panic(fmt.Sprintf("unexpected error when setting VDT value: %s", err.Error()))
				}

				message := validationprotocol.NewValidationProtocolVDT()
				err = message.SetValue(validationprotocol.StatementDistribution{StatementDistributionMessage: sdm})
				if err != nil {
					panic(fmt.Sprintf("unexpected error when setting VDT value: %s", err.Error()))
				}

				messages = append(messages, &networkbridgemessages.SendValidationMessage{
					To:                        []peer.ID{peerID},
					ValidationProtocolMessage: message,
				})
			}
		case acknowledgement:
			innerMessages, _ := acknowledgementAndStatementMessages(
				peerID, validationVersion,
				peerValidatorID,
				groups,
				rpState,
				rp,
				groupIndex,
				candidateHash,
				*localKnowledge,
			)
			messages = append(messages, innerMessages...)
		}
	}

	// Send all remaining pending grid statements for a validator, not just
	// those for the acknowledgements we've sent.
	//
	// otherwise, we might receive statements while the grid peer is "out of view" and then
	// not send them when they get back "in view". problem!
	gt := rpState.localValidator.gridTracker
	pendingStmts := gt.allPendingStatementsFor(peerValidatorID)

	var extraStmtMessages []*networkbridgemessages.SendValidationMessage
	for _, ps := range pendingStmts {
		res := pendingStatementNetworkMessage(
			rpState.statementStore,
			rp,
			peerID, validationVersion,
			ps,
		)

		if res != nil {
			gt.sentOrReceivedDirectStatement(
				*groups,
				ps.validatorIndex,
				peerValidatorID,
				ps.statement,
				false,
			)

			extraStmtMessages = append(extraStmtMessages, res)
		}
	}

	messages = append(messages, extraStmtMessages...)

	if len(messages) > 0 {
		s.SubSystemToOverseer <- networkbridgemessages.SendValidationMessages{Messages: messages}
	}

	return nil
}

// Send backing fresh statements. This should only be performed on importable & confirmed candidates
func (s *StatementDistribution) sendBackingFreshStatements(
	candidateHash parachaintypes.CandidateHash,
	groupIndex parachaintypes.GroupIndex,
	relayParent common.Hash,
	rpState *perRelayParentState,
	confirmed *confirmedCandidate,
	perSessionState *perSessionState,
) error {
	type validatorIndexAndCompact struct {
		validatorIndex parachaintypes.ValidatorIndex
		compact        parachaintypes.CompactStatement
	}

	groupValidators := perSessionState.groups.get(groupIndex)
	var imported []validatorIndexAndCompact

	freshStatements := rpState.statementStore.freshStatementsForBacking(groupValidators, candidateHash)
	for _, freshStmt := range freshStatements {
		v := freshStmt.ValidatorIndex
		compact, err := freshStmt.Payload.ToCompact()
		if err != nil {
			return err
		}

		var (
			innerStmtKind any
			withPVD       *parachaintypes.PersistedValidationData
		)

		switch inner := compact.(type) {
		case *parachaintypes.CompactValid:
			innerStmtKind = parachaintypes.Valid(inner.CandidateHash())

		case *parachaintypes.CompactSeconded:
			innerStmtKind = parachaintypes.Seconded(confirmed.receipt)
			withPVD = confirmed.pvd
		}

		convertedStmt := parachaintypes.NewStatementVDT()
		err = convertedStmt.SetValue(innerStmtKind)
		if err != nil {
			panic(fmt.Sprintf("unexpected error setting Statement VDT: %s", err.Error()))
		}

		signed, err := compareAndConvert(freshStmt, convertedStmt, withPVD)
		if err != nil {
			return fmt.Errorf("comparing and converting stmt: %w", err)
		}

		s.SubSystemToOverseer <- backing.StatementMessage{
			RelayParent:         relayParent,
			SignedFullStatement: *signed,
		}

		imported = append(imported, validatorIndexAndCompact{
			validatorIndex: v,
			compact:        compact,
		})
	}

	for _, i := range imported {
		rpState.statementStore.noteKnownByBacking(i.validatorIndex, i.compact)
	}

	return nil
}

// compareAndConvert ensure the original compact statement matches
// the same encoding as the converted statement and transforms the
// converted statement into a SignedFullStatementWithPVD with the
// signature of the original compact statement
func compareAndConvert(
	original parachaintypes.SignedStatement,
	converted parachaintypes.StatementVDT,
	pvd *parachaintypes.PersistedValidationData,
) (*parachaintypes.SignedFullStatementWithPVD, error) {
	expectedCompactEncoded, err := original.Payload.MarshalSCALE()
	if err != nil {
		return nil, fmt.Errorf("marshalling exepected compact payload: %w", err)
	}

	compactSeconded, err := converted.CompactStatement()
	if err != nil {
		return nil, fmt.Errorf("getting compact statement: %w", err)
	}

	encodedCompact, err := compactSeconded.ToEncodable().MarshalSCALE()
	if err != nil {
		return nil, fmt.Errorf("marshalling compact seconded: %w", err)
	}

	if !bytes.Equal(expectedCompactEncoded, encodedCompact) {
		return nil, fmt.Errorf("%w: %v !=  %v", errEncodedStatementsDoNotMatch, expectedCompactEncoded, encodedCompact)
	}

	uncheckedFreshStmt := parachaintypes.UncheckedSignedCompactStatement(original)
	return &parachaintypes.SignedFullStatementWithPVD{
		SignedFullStatement: parachaintypes.SignedFullStatement{
			Payload:        converted,
			ValidatorIndex: uncheckedFreshStmt.ValidatorIndex,
			Signature:      uncheckedFreshStmt.Signature,
		},
		PersistedValidationData: pvd,
	}, nil
}

func localKnowledgeFilter(
	groupSize int,
	groupIndex parachaintypes.GroupIndex,
	candidateHash parachaintypes.CandidateHash,
	statementStore statementStore,
) (*parachaintypes.StatementFilter, error) {
	f, err := parachaintypes.NewStatementFilter(uint(groupSize), false)
	if err != nil {
		return nil, fmt.Errorf("new statement filter: %w", err)
	}

	statementStore.fillStatementFilter(groupIndex, candidateHash, f)
	return f, nil
}

// acknowledgementAndStatementMessages produces acknowledgement and statement messages to be sent over the network,
// noting that they have been sent within the grid topology tracker as well.
func acknowledgementAndStatementMessages(
	peerID peer.ID, validationVersion validationprotocol.ValidationVersion,
	validatorIndex parachaintypes.ValidatorIndex,
	groups *groups,
	rpState *perRelayParentState,
	rp common.Hash,
	groupIndex parachaintypes.GroupIndex,
	candidateHash parachaintypes.CandidateHash,
	localKnowledge parachaintypes.StatementFilter,
) ([]*networkbridgemessages.SendValidationMessage, int) {
	if rpState.localValidator == nil {
		return []*networkbridgemessages.SendValidationMessage{}, 0
	}

	localValidator := rpState.localValidator
	var messages []*networkbridgemessages.SendValidationMessage

	switch validationVersion {
	case validationprotocol.ValidationVersionV3:
		ack := validationprotocol.NewStatementDistributionMessage()
		err := ack.SetValue(validationprotocol.BackedCandidateKnown{
			CandidateHash:      candidateHash,
			StatementKnowledge: localKnowledge,
		})
		if err != nil {
			panic(fmt.Sprintf("failed while defining enum variant: %s", err.Error()))
		}

		ver := validationprotocol.NewValidationProtocolVDT()
		err = ver.SetValue(validationprotocol.StatementDistribution{StatementDistributionMessage: ack})
		if err != nil {
			panic(fmt.Sprintf("failed while defining enum variant: %s", err.Error()))
		}

		messages = append(messages, &networkbridgemessages.SendValidationMessage{
			To:                        []peer.ID{peerID},
			ValidationProtocolMessage: ver,
		})
	}

	localValidator.gridTracker.manifestSentTo(
		*groups,
		validatorIndex,
		candidateHash,
		localKnowledge,
	)

	stmtMessages := postAcknowledgementStatementMessages(
		validatorIndex,
		rp,
		localValidator.gridTracker,
		rpState.statementStore,
		groups,
		groupIndex,
		candidateHash,
		peerID, validationVersion,
	)

	messages = append(messages, stmtMessages...)
	return messages, len(stmtMessages)
}

func postAcknowledgementStatementMessages(
	recipient parachaintypes.ValidatorIndex,
	rp common.Hash,
	gridTracker *gridTracker,
	stmtStore statementStore,
	groups *groups,
	groupIndex parachaintypes.GroupIndex,
	candidateHash parachaintypes.CandidateHash,
	peerID peer.ID, validationVersion validationprotocol.ValidationVersion,
) []*networkbridgemessages.SendValidationMessage {
	sendingFilter := gridTracker.pendingStatementsFor(recipient, candidateHash)
	if sendingFilter == nil {
		return []*networkbridgemessages.SendValidationMessage{}
	}

	var messages []*networkbridgemessages.SendValidationMessage
	for _, stmt := range stmtStore.groupStatements(groups, groupIndex, candidateHash, sendingFilter) {
		compactStmt, err := stmt.Payload.ToCompact()
		if err != nil {
			logger.Criticalf("getting value from encodable compact statement: %w", err)
			return nil
		}

		gridTracker.sentOrReceivedDirectStatement(*groups, stmt.ValidatorIndex, recipient, compactStmt, false)

		switch validationVersion {
		case validationprotocol.ValidationVersionV3:
			stmtMessage := validationprotocol.NewStatementDistributionMessage()
			err := stmtMessage.SetValue(validationprotocol.Statement{
				RelayParent: rp,
				Compact:     parachaintypes.UncheckedSignedCompactStatement(stmt),
			})
			if err != nil {
				panic(fmt.Sprintf("failed while defining enum variant: %s", err.Error()))
			}

			vp := validationprotocol.NewValidationProtocolVDT()
			err = vp.SetValue(validationprotocol.StatementDistribution{StatementDistributionMessage: stmtMessage})
			if err != nil {
				panic(fmt.Sprintf("failed while defining enum variant: %s", err.Error()))
			}

			messages = append(messages, &networkbridgemessages.SendValidationMessage{
				To:                        []peer.ID{peerID},
				ValidationProtocolMessage: vp,
			})
		}
	}

	return messages
}

func pendingStatementNetworkMessage(
	stmtStore statementStore,
	rp common.Hash,
	peerID peer.ID, validationVersion validationprotocol.ValidationVersion,
	pending originatorStatementPair,
) *networkbridgemessages.SendValidationMessage {
	switch validationVersion {
	case validationprotocol.ValidationVersionV3:
		signed := stmtStore.validatorStatement(pending)
		if signed == nil {
			return nil
		}

		sdmV3 := validationprotocol.NewStatementDistributionMessage()
		err := sdmV3.SetValue(validationprotocol.Statement{
			RelayParent: rp,
			Compact:     parachaintypes.UncheckedSignedCompactStatement(*signed),
		})
		if err != nil {
			panic(fmt.Sprintf("unexpected error setting value in StatementDistributionMessageV3: %s", err))
		}

		// TODO: this will panic as validation protocol does not support V3 yet
		vp := validationprotocol.NewValidationProtocolVDT()
		err = vp.SetValue(validationprotocol.StatementDistribution{StatementDistributionMessage: sdmV3})
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
