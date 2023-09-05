package parachain

import (
	"errors"
	"fmt"

	parachainruntime "github.com/ChainSafe/gossamer/dot/parachain/runtime"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/erasure"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

type ValidatedCandidateCommand byte

const (
	// We were instructed to second the candidate that has been already validated.
	Second = ValidatedCandidateCommand(iota)
	// We were instructed to validate the candidate.
	Attest
	// We were not able to `Attest` because backing validator did not send us the PoV.
	AttestNoPoV
)

type BackgroundValidationResult struct {
	CandidateReceipt     parachaintypes.CandidateReceipt
	CandidateCommitments parachaintypes.CandidateCommitments
	PoV                  PoV
	isValid              bool
}

var InvalidErasureRoot = errors.New("Invalid erasure root")

func ValidateAndMakeAvailable(
	nValidators uint,
	runtimeInstance parachainruntime.RuntimeInstance,
	povRequestor PoVRequestor,
	candidateReceipt parachaintypes.CandidateReceipt,
) error {

	// TODO: either use already available data (from candidate selection) if possible,
	// or request it from the validator.
	// https://github.com/paritytech/polkadot/blob/9b1fc27cec47f01a2c229532ee7ab79cc5bb28ef/node/core/backing/src/lib.rs#L697-L708
	pov := povRequestor.RequestPoV(candidateReceipt.Descriptor.PovHash) // temporary

	candidateCommitments, persistedValidationData, isValid, err := ValidateFromChainState(runtimeInstance, povRequestor, candidateReceipt)
	if err != nil {
		return err
	}
	fmt.Printf("\n\ncandidateCommitments: %v\n\n", candidateCommitments) // remove this. just to avoid unused error

	if isValid {
		candidateHash := CandidateHash{common.MustBlake2bHash(scale.MustMarshal(candidateReceipt))}
		if err := MakePoVAvailable(
			nValidators,
			pov,
			candidateHash,
			*persistedValidationData,
			candidateReceipt.Descriptor.ErasureRoot,
		); err != nil {
			return err
		}
	}

	// TODO: If is not valid Report to collator protocol,
	// about the invalidity so that it can punish the collator that sent us this candidate

	return nil
}

func MakePoVAvailable(
	nValidators uint,
	pov PoV,
	candidateHash CandidateHash,
	validationData parachaintypes.PersistedValidationData,
	expectedErasureRoot common.Hash,
) error {
	availableData := AvailableData{pov, validationData}
	availableDataBytes, err := scale.Marshal(availableData)
	if err != nil {
		return err
	}

	chunks, err := erasure.ObtainChunks(nValidators, availableDataBytes)
	if err != nil {
		return err
	}

	chunksTrie, err := erasure.ChunksToTrie(chunks)
	if err != nil {
		return err
	}

	root, err := chunksTrie.Hash()
	if err != nil {
		return err
	}

	if root != expectedErasureRoot {
		return InvalidErasureRoot
	}

	// TODO: send a message to overseear to store the available data
	// https://github.com/paritytech/polkadot/blob/9b1fc27cec47f01a2c229532ee7ab79cc5bb28ef/node/core/backing/src/lib.rs#L566-L573

	return nil
}

func runCandidateBacking() {
	for {
		if err := run_iteration(); err == nil {
			return
		}
	}
}

func run_iteration() error {
	for {
		select {
		// case <- recieve validated candidate command:
		// handleValidatedCandidateCommand()

		// case <-
		}
	}
	return nil
}

// Holds all data needed for candidate backing job operation.
type CandidateBackingJob struct {
	// The hash of the relay parent on top of which this job is doing it's work.
	parent common.Hash
	// The `ParaId` assigned to this validator
	assignment *parachaintypes.ParaID
	// The collator required to author the candidate, if any.
	required_collator *parachaintypes.CollatorID
	// Spans for all candidates that are not yet backable.
	unbacked_candidates map[CandidateHash]bool
	// We issued `Seconded`, `Valid` or `Invalid` statements on about these candidates.
	issued_statements map[CandidateHash]bool
	// These candidates are undergoing validation in the background.
	awaiting_validation map[CandidateHash]bool
	// Data needed for retrying in case of `ValidatedCandidateCommand::AttestNoPoV`.
	fallbacks map[CandidateHash]AttestingData
	// `Some(h)` if this job has already issued `Seconded` statement for some candidate with `h` hash.
	seconded *CandidateHash
	// The candidates that are includable, by hash. Each entry here indicates
	// that we've sent the provisioner the backed candidate.
	backed        map[CandidateHash]bool
	keystore      *keystore.Keystore
	table         Table
	table_context TableContext
}

func (job *CandidateBackingJob) signImportAndDistributeStatement(statement Statement) (*checkedSignedFullStatement, error) {
	signedFullStatement, err := job.table_context.validator.Sign(*job.keystore, statement)
	if err != nil {
		return nil, err
	}
	//-----------------------------------------

	// self.import_statement(ctx, &signed_statement, root_span)?;
	candidateHash, err := signedFullStatement.Payload.CandidateHash()
	if err != nil {
		return nil, fmt.Errorf("getting candidate hash from statement: %w", err)
	}

	_, isBacked := job.backed[*candidateHash]
	if !isBacked {
		// only add if we don't consider this backed.
		_, ok := job.unbacked_candidates[*candidateHash]
		if !ok {
			job.unbacked_candidates[*candidateHash] = true
		}
	}

	// continue from polkadot/node/core/backing/src/lib.rs , line number 1026

	//-----------------------------------------
	// TODO: distribute the statement
	// let smsg = StatementDistributionMessage::Share(self.parent, signed_statement.clone());
	// ctx.send_unbounded_message(smsg);
	return nil, nil
}

func (job *CandidateBackingJob) handleSecondCommand(
	validationRes BackgroundValidationResult,
	candidateHash CandidateHash,
) error {
	candidateHash = CandidateHash{
		Value: common.MustBlake2bHash(scale.MustMarshal(validationRes.CandidateReceipt)),
	}
	delete(job.awaiting_validation, candidateHash)

	if !validationRes.isValid {
		// // Break cycle - bounded as there is only one candidate to
		// // second per block.
		// ctx.send_unbounded_message(CollatorProtocolMessage::Invalid(
		// 	self.parent,
		// 	candidate,
		// ));
		return nil
	}

	// sanity check.
	_, isIssued := job.issued_statements[candidateHash]
	if job.seconded != nil && !isIssued {
		job.seconded = &candidateHash
		job.issued_statements[candidateHash] = true

		statement := NewStatement()
		if err := statement.Set(Seconded{
			Descriptor:  validationRes.CandidateReceipt.Descriptor,
			Commitments: validationRes.CandidateCommitments,
		}); err != nil {
			return fmt.Errorf("setting value to statement vdt: %s", err)
		}

		// TODO: Implement self.sign_import_and_distribute_statement(ctx, statement) method

		// if job.sign_import_and_distribute_statement(ctx, statement) != nil {
		// 	// // Break cycle - bounded as there is only one candidate to
		// 	// // second per block.
		// 	// ctx.send_unbounded_message(CollatorProtocolMessage::Invalid(
		// 	// 	self.parent,
		// 	// 	candidate,
		// 	// ));
		// }
	}
	return nil
}

func (job *CandidateBackingJob) handleAttestCommand(
	validationRes BackgroundValidationResult,
	candidateHash CandidateHash,
) error {
	candidateHash = CandidateHash{
		Value: common.MustBlake2bHash(scale.MustMarshal(validationRes.CandidateReceipt)),
	}
	delete(job.awaiting_validation, candidateHash)

	// We are done - avoid new validation spawns:
	delete(job.fallbacks, candidateHash)

	// sanity check.
	_, isIssued := job.issued_statements[candidateHash]
	if !isIssued {
		if validationRes.isValid {
			statement := NewStatement()
			if err := statement.Set(Valid{candidateHash.Value}); err != nil {
				return fmt.Errorf("setting value to statement vdt: %s", err)
			}
			// self.sign_import_and_distribute_statement(ctx, statement, &root_span)?;
		}
		job.issued_statements[candidateHash] = true
	}
	return nil
}

func (job *CandidateBackingJob) handleAttestNoPoVCommand(candidateHash CandidateHash) error {
	delete(job.awaiting_validation, candidateHash)

	attesting, ok := job.fallbacks[candidateHash]
	if ok {
		backingLen := len(attesting.backing)
		if backingLen > 1 {
			lastBackingIndex := attesting.backing[backingLen-1]
			attesting.backing = attesting.backing[:backingLen-1]
			attesting.from_validator = lastBackingIndex

			// TODO: Implement self.kick_off_validation_work(ctx, attesting, c_span) method
			// self.kick_off_validation_work(ctx, attesting, c_span).await?
		}
	}

	return nil
}

func (job *CandidateBackingJob) handleValidatedCandidateCommand(
	command ValidatedCandidateCommand,
	validationRes BackgroundValidationResult,
	candidateHash CandidateHash,
) error {
	switch command {
	case Second:
		return job.handleSecondCommand(validationRes, candidateHash)
	case Attest:
		return job.handleAttestCommand(validationRes, candidateHash)
	case AttestNoPoV:
		return job.handleAttestNoPoVCommand(candidateHash)
	}
	return nil
}

// Stores votes
type Table struct {
	// TODO: fill the fields
}

type TableContext struct {
	validator  *Validator
	groups     map[parachaintypes.ParaID]parachaintypes.ValidatorIndex
	validators []parachaintypes.ValidatorID
}

// Local validator information
//
// It can be created if the local node is a validator in the context of a particular
// relay chain block.
type Validator struct {
	signing_context SigningContext
	key             parachaintypes.ValidatorID
	index           parachaintypes.ValidatorIndex
}

// Sign a payload with this validator
func (v Validator) Sign(keystore keystore.Keystore, Payload Statement) (*checkedSignedFullStatement, error) {
	checkedSignedFullStatement := checkedSignedFullStatement{
		Payload:        Payload,
		ValidatorIndex: v.index,
	}

	signature, err := checkedSignedFullStatement.Sign(keystore, v.signing_context, v.key)
	if err != nil {
		return nil, err
	}
	checkedSignedFullStatement.Signature = *signature

	return &checkedSignedFullStatement, nil
}

// A type returned by runtime with current session index and a parent hash.
type SigningContext struct {
	/// Current session index.
	SessionIndex parachaintypes.SessionIndex
	/// Hash of the parent.
	ParentHash common.Hash
}

// In case a backing validator does not provide a PoV, we need to retry with other backing
// validators.
//
// This is the data needed to accomplish this. Basically all the data needed for spawning a
// validation job and a list of backing validators, we can try.
type AttestingData struct {
	/// The candidate to attest.
	candidate parachaintypes.CandidateReceipt
	/// Hash of the PoV we need to fetch.
	pov_hash common.Hash
	/// Validator we are currently trying to get the PoV from.
	from_validator parachaintypes.ValidatorIndex
	/// Other backing validators we can try in case `from_validator` failed.
	backing []parachaintypes.ValidatorIndex
}
