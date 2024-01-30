// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package backing

import (
	"errors"
	"fmt"

	collatorprotocolmessages "github.com/ChainSafe/gossamer/dot/parachain/collator-protocol/messages"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/tidwall/btree"
)

var (
	errNilBackgroundValidationResult          = errors.New("background validation result is nil")
	errRelayParentNoLongerRelevent            = errors.New("relay parent is no longer relevant")
	errNilCandidateInBgValidationResult       = errors.New("candidate receipt is nil in background validation result")
	errNilOutputsInBackgroundValidationResult = errors.New("outputs is nil in background validation result")
)

type backgroundValidationResult struct {
	// outputs contains the validation outputs when there is no error.
	// It should be nil if an error occurs during background validation.
	outputs *backgroundValidationOutputs

	// candidate should have values assigned if there is an error; otherwise, it should be nil.
	candidate *parachaintypes.CandidateReceipt

	// err represents any error that occurred during background validation
	err error
}

// backgroundValidationOutputs contains the outputs of the background validation.
type backgroundValidationOutputs struct {
	candidateReceipt        parachaintypes.CandidateReceipt
	candidateCommitments    parachaintypes.CandidateCommitments
	persistedValidationData parachaintypes.PersistedValidationData
}

// relayParentAndCommand contains the relay parent and the command to be executed on validated candidate,
// along with the result of the background validation.
type relayParentAndCommand struct {
	relayParent   common.Hash
	command       validatedCandidateCommand
	validationRes *backgroundValidationResult
	candidateHash parachaintypes.CandidateHash
}

// validatedCandidateCommand represents commands for handling validated candidates.
// This is not a command to validate a candidate, but to react to a validation result.
type validatedCandidateCommand byte

const (
	// We were instructed to second the candidate that has been already validated.
	second = validatedCandidateCommand(iota)
	// We were instructed to validate the candidate.
	attest
	// We were not able to `Attest` because backing validator did not send us the PoV.
	attestNoPoV
)

// processValidatedCandidateCommand notes the result of a background validation of a candidate and reacts accordingly..
func (cb *CandidateBacking) processValidatedCandidateCommand(rpAndCmd relayParentAndCommand) error {
	rpState, ok := cb.perRelayParent[rpAndCmd.relayParent]
	if !ok {
		return fmt.Errorf("%w: %s", errRelayParentNoLongerRelevent, rpAndCmd.relayParent)
	}
	if rpState == nil {
		return fmt.Errorf("%w; relay parent: %s", errNilRelayParentState, rpAndCmd.relayParent)
	}

	delete(rpState.awaitingValidation, rpAndCmd.candidateHash)

	switch rpAndCmd.command {
	case second:
		err := cb.handleCommandSecond(rpAndCmd.validationRes, rpAndCmd.candidateHash, rpState)
		if err != nil {
			return fmt.Errorf("unable to second the candidate: %w", err)
		}
		return nil
	case attest:
		handleCommandAttest(*rpAndCmd.validationRes)
	case attestNoPoV:
		handleCommandAttestNoPoV(rpAndCmd.candidateHash)
	}

	return nil
}

func (cb *CandidateBacking) handleCommandSecond(
	bgValidationResult *backgroundValidationResult,
	candidateHash parachaintypes.CandidateHash,
	rpState *perRelayParentState,
) error {
	if bgValidationResult == nil {
		return errNilBackgroundValidationResult
	}

	// If there is an error, we notify collator protocol about it.
	if bgValidationResult.err != nil {
		if bgValidationResult.candidate == nil {
			return errNilCandidateInBgValidationResult
		}

		cb.SubSystemToOverseer <- collatorprotocolmessages.Invalid{
			Parent:           rpState.relayParent,
			CandidateReceipt: *bgValidationResult.candidate,
		}
		return nil
	}

	if bgValidationResult.outputs == nil {
		return errNilOutputsInBackgroundValidationResult
	}

	if rpState.issuedStatements[candidateHash] {
		// already issued a statement for this candidate
		return nil
	}

	pvd := bgValidationResult.outputs.persistedValidationData
	commitments := bgValidationResult.outputs.candidateCommitments
	candidate := bgValidationResult.outputs.candidateReceipt

	parentHeadDataHash, err := common.Blake2bHash(pvd.ParentHead.Data)
	if err != nil {
		return fmt.Errorf("hashing parent head data: %w", err)
	}

	commitmentsHeadDataHash, err := common.Blake2bHash(commitments.HeadData.Data)
	if err != nil {
		return fmt.Errorf("hashing commitments head data: %w", err)
	}

	if parentHeadDataHash == commitmentsHeadDataHash {
		return nil
	}

	commitedCandidate := parachaintypes.CommittedCandidateReceipt{
		Descriptor:  candidate.Descriptor,
		Commitments: commitments,
	}

	hypotheticalCandidate := parachaintypes.HypotheticalCandidateComplete{
		CandidateHash:             candidateHash,
		CommittedCandidateReceipt: commitedCandidate,
		PersistedValidationData:   pvd,
	}

	// sanity check that we're allowed to second the candidate and that it doesn't conflict with other candidates we've seconded.
	fragmentTreeMembership, err := cb.secondingSanityCheck(hypotheticalCandidate, false)
	if err != nil {
		return err
	}

	statement := parachaintypes.NewStatementVDT()
	err = statement.Set(parachaintypes.Seconded(commitedCandidate))
	if err != nil {
		return fmt.Errorf("setting statement: %w", err)
	}

	// If we get an errRejectedByProspectiveParachains,
	// then the statement has not been distributed or imported into the table
	signedFullStatementWithPVD, err := signImportAndDistributeStatement(
		cb.SubSystemToOverseer, rpState, cb.perCandidate, statement, &pvd, cb.keystore)

	if err != nil {
		if errors.Is(err, errRejectedByProspectiveParachains) {
			cb.SubSystemToOverseer <- collatorprotocolmessages.Invalid{
				Parent:           candidate.Descriptor.RelayParent,
				CandidateReceipt: candidate,
			}
		}
		return err
	}

	perCandidate, ok := cb.perCandidate[candidateHash]
	if !ok {
		logger.Warnf("missing `per candidate` for seconded candidate: %s", candidateHash.Value)
	} else {
		perCandidate.secondedLocally = true
	}

	// update seconded depths in active leaves.
	for leaf, depths := range fragmentTreeMembership {
		leafState, ok := cb.perLeaf[leaf]
		if !ok {
			logger.Warnf("missing `per leaf` for known active leaf: %s", leaf)
			continue
		}

		secondedAtDepth, ok := leafState.secondedAtDepth[parachaintypes.ParaID(candidate.Descriptor.ParaID)]
		if !ok {
			var btreeMap btree.Map[uint, parachaintypes.CandidateHash]
			leafState.secondedAtDepth[parachaintypes.ParaID(candidate.Descriptor.ParaID)] = &btreeMap
			secondedAtDepth = &btreeMap
		}

		for _, depth := range depths {
			secondedAtDepth.Set(depth, candidateHash)
		}
	}

	rpState.issuedStatements[candidateHash] = true
	cb.SubSystemToOverseer <- collatorprotocolmessages.Seconded{
		Parent: rpState.relayParent,
		Stmt:   signedFullStatementWithPVD.SignedFullStatement,
	}

	return nil
}

func handleCommandAttest(bgValidationResult backgroundValidationResult)   {}
func handleCommandAttestNoPoV(candidateHash parachaintypes.CandidateHash) {}

func signImportAndDistributeStatement(
	subSystemToOverseer chan<- any,
	rpState *perRelayParentState,
	perCandidate map[parachaintypes.CandidateHash]*perCandidateState,
	statementVDT parachaintypes.StatementVDT,
	pvd *parachaintypes.PersistedValidationData,
	keystore keystore.Keystore,
) (parachaintypes.SignedFullStatementWithPVD, error) {
	signedStatement, err := rpState.tableContext.validator.sign(keystore, statementVDT)
	if err != nil {
		return parachaintypes.SignedFullStatementWithPVD{}, fmt.Errorf("signing statement: %w", err)
	}

	signedStatementWithPVD := parachaintypes.SignedFullStatementWithPVD{
		SignedFullStatement:     signedStatement,
		PersistedValidationData: pvd,
	}

	summary, err := rpState.importStatement(subSystemToOverseer, signedStatementWithPVD, perCandidate)
	if err != nil {
		return parachaintypes.SignedFullStatementWithPVD{}, fmt.Errorf("importing statement: %w", err)
	}

	// `Share` must always be sent before `Backed`. We send the latter in `postImportStatement` below.
	subSystemToOverseer <- parachaintypes.StatementDistributionMessageShare{
		RelayParent:                rpState.relayParent,
		SignedFullStatementWithPVD: signedStatementWithPVD,
	}

	rpState.postImportStatement(subSystemToOverseer, summary)
	return signedStatementWithPVD, nil
}
