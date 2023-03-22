package parachaininteraction

// pub enum ValidationProtocol {
// 	/// Bitfield distribution messages
// 	#[codec(index = 1)]
// 	#[from]
// 	BitfieldDistribution(BitfieldDistributionMessage),
// 	/// Statement distribution messages
// 	#[codec(index = 3)]
// 	#[from]
// 	StatementDistribution(StatementDistributionMessage),
// 	/// Approval distribution messages
// 	#[codec(index = 4)]
// 	#[from]
// 	ApprovalDistribution(ApprovalDistributionMessage),
// }

// This subsystem groups the requests it handles in two categories: candidate validation and PVF pre-checking.

// The first category can be further subdivided in two request types: one which draws out validation data from the state, and another which accepts all validation data exhaustively. Validation returns three possible outcomes on the response channel: the candidate is valid, the candidate is invalid, or an internal error occurred.

// Parachain candidates are validated against their validation function: A piece of Wasm code that describes the state-transition of the parachain. Validation function execution is not metered. This means that an execution which is an infinite loop or simply takes too long must be forcibly exited by some other means. For this reason, we recommend dispatching candidate validation to be done on subprocesses which can be killed if they time-out.

// Received candidates submitted by collators and must have its validity verified by the assigned Polkadot validators. For each candidate to be valid, the validator must successfully verify the following conditions in the following order:

//     The candidate does not exceed any parameters in the persisted validation data (Definition 227).

//     The signature of the collator is valid.

//     Validate the candidate by executing the parachain Runtime (Section 8.3.1).

// If all steps are valid, the Polkadot validator must create the necessary candidate commitments (Definition 107) and submit the appropriate statement for each candidate (Section 8.2.1).

func Validate(c Collation) {
	//  CandidateValidationMessage::ValidateFromChainState(

	// 	CandidateValidationMessage::ValidateFromExhaustive(

	// 	CandidateValidationMessage::PreCheck(

}

// look at node/core/candidate-validation/src/lib.rs
