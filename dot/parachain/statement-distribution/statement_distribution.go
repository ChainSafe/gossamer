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
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	parachainutil "github.com/ChainSafe/gossamer/dot/parachain/util"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
)

var errEncodedStatementsMismatch = errors.New("encoded statements does not match")

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
		compact := freshStmt.Payload

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
		err := convertedStmt.SetValue(innerStmtKind)
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

	encodedCompact, err := compactSeconded.MarshalSCALE()
	if err != nil {
		return nil, fmt.Errorf("marshalling compact seconded: %w", err)
	}

	if !bytes.Equal(expectedCompactEncoded, encodedCompact) {
		return nil, fmt.Errorf("%w: %v !=  %v", errEncodedStatementsMismatch, expectedCompactEncoded, encodedCompact)
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
