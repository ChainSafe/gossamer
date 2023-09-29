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

var InvalidErasureRoot = errors.New("Invalid erasure root")

type commandAndRelayParent struct {
	relayParent   common.Hash
	command       ValidatedCandidateCommand
	validationRes BackgroundValidationResult
	candidateHash CandidateHash
}

type ValidatedCandidateCommand byte

/*
	enum ValidatedCandidateCommand {
		Second(BackgroundValidationResult),
		Attest(BackgroundValidationResult),
		AttestNoPoV(CandidateHash),
	}
*/

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

func attestedToBacked(attested AttestedCandidate, tableContext TableContext) *parachaintypes.CandidateBacked {
	// TODO: implement this function
	return &parachaintypes.CandidateBacked{}
}

// TODO: use actual type of overseearSender and overseearReceiver
func runCandidateBacking(overseearSender chan<- interface{}, overseearReceiver <-chan interface{}) {
	jobs := make(map[common.Hash]CandidateBackingJob)

	// TODO: figure out buffer size of these channels.
	validationSender := make(chan<- commandAndRelayParent)
	validationReceiver := make(<-chan commandAndRelayParent)

	for {
		if err := run_iteration(
			jobs,
			validationSender,
			validationReceiver,
			overseearSender,
			overseearReceiver,
		); err == nil {
			return
		}
	}
}

func run_iteration(
	jobs map[common.Hash]CandidateBackingJob,
	validationSender chan<- commandAndRelayParent,
	validationReceiver <-chan commandAndRelayParent,
	overseearSender chan<- interface{},
	overseearReceiver <-chan interface{},
) error {
	for {
		select {
		case cmdAndParent, ok := <-validationReceiver:
			if !ok {
				return nil
			}

			if job, ok := jobs[cmdAndParent.relayParent]; ok {
				if err := job.handleValidatedCandidateCommand(
					cmdAndParent.command,
					cmdAndParent.validationRes,
					cmdAndParent.candidateHash,
				); err != nil {
					return fmt.Errorf("handling validated candidate command: %w", err)
				}
			}

		case data, ok := <-overseearReceiver:
			if !ok {
				return nil
			}

			fmt.Print(data) // remove this line, just to avoid unused error

			// switch data := data.(type) {
			// TODO: Implement this case
			// FromOrchestra::Signal(OverseerSignal::ActiveLeaves(update)) => handle_active_leaves_update(
			// 	&mut *ctx,
			// 	update,
			// 	jobs,
			// 	&keystore,
			// 	&background_validation_tx,
			// 	&metrics,
			// ).await?,

			// TODO: Implement this case
			// FromOrchestra::Communication { msg } => handle_communication(&mut *ctx, jobs, msg).await?,
			// }
		}
	}
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
	backed       map[CandidateHash]bool
	keystore     *keystore.Keystore
	table        Table
	tableContext TableContext
}

func (job *CandidateBackingJob) handleValidatedCandidateCommand(
	command ValidatedCandidateCommand,
	validationRes BackgroundValidationResult,
	candidateHash CandidateHash,
) error {
	delete(job.awaiting_validation, candidateHash)
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

func (job *CandidateBackingJob) handleSecondCommand(
	validationRes BackgroundValidationResult,
	candidateHash CandidateHash,
) error {
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
	if job.seconded != nil || job.issued_statements[candidateHash] {
		return nil
	}

	job.seconded = &candidateHash
	job.issued_statements[candidateHash] = true

	statement := NewStatementVDT()
	if err := statement.Set(Seconded{
		Descriptor:  validationRes.CandidateReceipt.Descriptor,
		Commitments: validationRes.CandidateCommitments,
	}); err != nil {
		return fmt.Errorf("setting value to statement vdt: %s", err)
	}

	signedFullStatement, err := job.signImportAndDistributeStatement(statement)
	if err != nil {
		return err
	}

	fmt.Printf("\njust to avoid unused error: signedFullStatement: %v\n", signedFullStatement) // remove this line

	if signedFullStatement != nil {
		/*
			TODO: implement this function

			// Break cycle - bounded as there is only one candidate to
			// second per block.
			ctx.send_unbounded_message(CollatorProtocolMessage::Seconded(
				self.parent,
				stmt,
			));
		*/
	}
	return nil
}

func (job *CandidateBackingJob) handleAttestCommand(
	validationRes BackgroundValidationResult,
	candidateHash CandidateHash,
) error {
	// We are done - avoid new validation spawns:
	delete(job.fallbacks, candidateHash)

	// sanity check.
	if _, isIssued := job.issued_statements[candidateHash]; isIssued {
		return nil
	}

	if validationRes.isValid {
		statement := NewStatementVDT()
		if err := statement.Set(Valid{candidateHash.Value}); err != nil {
			return fmt.Errorf("setting value to statement vdt: %s", err)
		}
		if _, err := job.signImportAndDistributeStatement(statement); err != nil {
			return err
		}
	}

	job.issued_statements[candidateHash] = true
	return nil
}

func (job *CandidateBackingJob) handleAttestNoPoVCommand(candidateHash CandidateHash) error {
	attesting, ok := job.fallbacks[candidateHash]
	if !ok {
		logger.Warn("AttestNoPoV was triggered without fallback being available.")
		return nil
	}

	backingLen := len(attesting.backing)
	if backingLen > 1 {
		lastBackingIndex := attesting.backing[backingLen-1]
		attesting.backing = attesting.backing[:backingLen-1]
		attesting.from_validator = lastBackingIndex

		// TODO: Implement self.kick_off_validation_work(ctx, attesting, c_span) method
		// self.kick_off_validation_work(ctx, attesting, c_span).await?
	}

	return nil
}

// Import a statement into the statement table and return the summary of the import.
func (job *CandidateBackingJob) importStatement(signedFullStatement *SignedFullStatement) (*Summary, error) {
	candidateHash, err := signedFullStatement.Payload.CandidateHash()
	if err != nil {
		return nil, fmt.Errorf("getting candidate hash from statement: %w", err)
	}

	_, isBacked := job.backed[*candidateHash]
	if !isBacked {
		// only add if we don't consider this backed.
		job.unbacked_candidates[*candidateHash] = true
	}

	summary, err := job.table.importStatement(&job.tableContext, signedFullStatement)
	if err != nil {
		logger.Errorf("importing statement: %s", err)
	}

	if summary == nil {
		// self.issue_new_misbehaviors(ctx.sender());
		return summary, nil
	}

	attested, err := job.table.attestedCandidate(&summary.Candidate, &job.tableContext)
	if err != nil {
		logger.Errorf("getting attested candidate: %s", err)
	}

	if attested != nil && !isBacked {
		job.backed[*candidateHash] = true
		delete(job.unbacked_candidates, *candidateHash)

		backedCandidate := attestedToBacked(*attested, job.tableContext)
		if backedCandidate != nil {
			// The provisioner waits on candidate-backing, which means
			// that we need to send unbounded messages to avoid cycles.
			//
			// Backed candidates are bounded by the number of validators,
			// parachains, and the block production rate of the relay chain.
			// let message = ProvisionerMessage::ProvisionableData(
			// 	self.parent,
			// 	ProvisionableData::BackedCandidate(backed.receipt()),
			// );
			// ctx.send_unbounded_message(message);
		}
	}

	// TODO: implement this function
	// self.issue_new_misbehaviors(ctx.sender());
	return summary, nil
}

func (job *CandidateBackingJob) signImportAndDistributeStatement(statement StatementVDT) (*SignedFullStatement, error) {
	signedFullStatement, err := job.tableContext.validator.Sign(*job.keystore, statement)
	if err != nil {
		logger.Errorf("signing statement: %w", err)
		return nil, nil
	}

	_, err = job.importStatement(signedFullStatement)
	if err != nil {
		return nil, fmt.Errorf("importing statement: %w", err)
	}

	// TODO: distribute the statement
	// let smsg = StatementDistributionMessage::Share(self.parent, signed_statement.clone());
	// ctx.send_unbounded_message(smsg);
	return signedFullStatement, nil
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
func (v Validator) Sign(keystore keystore.Keystore, Payload StatementVDT) (*SignedFullStatement, error) {
	signedFullStatement := SignedFullStatement{
		Payload:        Payload,
		ValidatorIndex: v.index,
	}

	signature, err := signedFullStatement.Sign(keystore, v.signing_context, v.key)
	if err != nil {
		return nil, err
	}
	signedFullStatement.Signature = *signature

	return &signedFullStatement, nil
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

// Requests a set of backable candidates that could be backed in a child of the given
// relay-parent, referenced by its hash.
type BackingMsgGetBackedCandidates struct {
	RelayParent common.Hash
	// TODO: add other fields
}

// Note that the Candidate Backing subsystem should second the given candidate in the context of the
// given relay parent. This candidate must be validated.
type BackingMsgSecond struct {
	RelayParent      common.Hash
	CandidateReceipt parachaintypes.CandidateReceipt
	PoV              PoV
}

// Note a validator's statement about a particular candidate. Disagreements about validity must be escalated
// to a broader check by the Disputes Subsystem, though that escalation is deferred until the approval voting
// stage to guarantee availability. Agreements are simply tallied until a quorum is reached.
type BackingMsgStatement struct {
	RelayParent         common.Hash
	SignedFullStatement SignedFullStatement
}

func (job *CandidateBackingJob) handleCandidateBakingMessage(candidateBackingMessage any) {
	switch message := candidateBackingMessage.(type) {
	case BackingMsgGetBackedCandidates:
		job.handleBackingMsgGetBackedCandidates(message)
	case BackingMsgSecond:
		job.handleBackingMsgSecond(message)
	case BackingMsgStatement:
		job.handleBackingMsgStatement(message)
	default:
		logger.Error("Unknown candidate backing message")
	}
}

func (job *CandidateBackingJob) handleBackingMsgGetBackedCandidates(message BackingMsgGetBackedCandidates) error {
	// TODO: implement this function
	return nil
}

func (job *CandidateBackingJob) handleBackingMsgSecond(message BackingMsgSecond) error {
	// TODO: implement this function
	return nil
}

func (job *CandidateBackingJob) handleBackingMsgStatement(message BackingMsgStatement) error {
	// TODO: implement this function
	return nil
}
