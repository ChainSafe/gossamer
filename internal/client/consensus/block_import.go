package consensus

import (
	hashdb "github.com/ChainSafe/gossamer/internal/hash-db"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	statemachine "github.com/ChainSafe/gossamer/internal/primitives/state-machine"
)

// / Block import result.
type ImportResult any

// / Block import results
type ImportResults interface {
	ImportResultImported | ImportResultAlreadyInChain | ImportResultKnownBad | ImportResultUnknownParent | ImportResultMissingState
}

// / Block imported.
type ImportResultImported ImportedAux

// Already in the blockchain.
type ImportResultAlreadyInChain struct{}

// /// Block or parent is known to be bad.
type ImportResultKnownBad struct{}

// /// Block parent is not in the chain.
type ImportResultUnknownParent struct{}

// /// Parent state is missing.
type ImportResultMissingState struct{}

// / Auxiliary data associated with an imported block result.
// #[derive(Debug, Default, PartialEq, Eq, Serialize, Deserialize)]
// pub struct ImportedAux {
type ImportedAux struct {
	// /// Only the header has been imported. Block body verification was skipped.
	// pub header_only: bool,
	HeaderOnly bool
	// /// Clear all pending justification requests.
	// pub clear_justification_requests: bool,
	ClearJustificationRequests bool
	// /// Request a justification for the given block.
	// pub needs_justification: bool,
	NeedsJustification bool
	// /// Received a bad justification.
	// pub bad_justification: bool,
	BadJustifications bool
	// /// Whether the block that was imported is the new best block.
	// pub is_new_best: bool,
	IsNewBest bool
}

// / Fork choice strategy.
// #[derive(Debug, PartialEq, Eq, Clone, Copy)]
// pub enum ForkChoiceStrategy {
type ForkChoiceStrategy any
type ForkChoiceStrategies interface {
	ForkChainStrategyLongestChain | ForkChainStrategyCustom
}

// /// Longest chain fork choice.
// LongestChain,
type ForkChainStrategyLongestChain struct{}

// /// Custom fork choice rule, where true indicates the new block should be the best block.
// Custom(bool),
type ForkChainStrategyCustom bool

// / Data required to check validity of a Block.
// #[derive(Debug, PartialEq, Eq, Clone)]
// pub struct BlockCheckParams<Block: BlockT> {
type BlockCheckParams[H, N any] struct {
	/// Hash of the block that we verify.
	// pub hash: Block::Hash,
	Hash H
	/// Block number of the block that we verify.
	// pub number: NumberFor<Block>,
	Number N
	/// Parent hash of the block that we verify.
	// pub parent_hash: Block::Hash,
	ParentHash H
	/// Allow importing the block skipping state verification if parent state is missing.
	// pub allow_missing_state: bool,
	AllowMissingState bool
	/// Allow importing the block if parent block is missing.
	// pub allow_missing_parent: bool,
	AllowMissingParent bool
	/// Re-validate existing block.
	// pub import_existing: bool,
	ImportExisting bool
}

// / Precomputed storage.
// pub enum StorageChanges<Block: BlockT, Transaction> {
type StorageChanges any
type StorageChangesValues[H runtime.Hash, Hasher hashdb.Hasher[H]] interface {
	StorageChangesChanges[H, Hasher] | StorageChangesImport[H]
}

// /// Changes coming from block execution.
type StorageChangesChanges[H runtime.Hash, Hasher hashdb.Hasher[H]] statemachine.StorageChanges[H, Hasher]

// /// Whole new state.
// Import(ImportedState<Block>),
type StorageChangesImport[H any] ImportedState[H]

// / Imported state data. A vector of key-value pairs that should form a trie.
// #[derive(PartialEq, Eq, Clone)]
// pub struct ImportedState<B: BlockT> {
type ImportedState[H any] struct {
	/// Target block hash.
	// pub block: B::Hash,
	Block H
	// /// State keys and values.
	// pub state: sp_state_machine::KeyValueStates,
	State statemachine.KeyValueStates
}

// / Defines how a new state is computed for a given imported block.
// pub enum StateAction<Block: BlockT, Transaction> {
type StateAction any
type StateActions interface {
	StateActionApplyChanges | StateActionExecute | StateActionExecuteIfPossible | StateActionSkip
}

// /// Apply precomputed changes coming from block execution or state sync.
// ApplyChanges(StorageChanges<Block, Transaction>),
type StateActionApplyChanges StorageChanges

// /// Execute block body (required) and compute state.
// Execute,
type StateActionExecute struct{}

// /// Execute block body if parent state is available and compute state.
// ExecuteIfPossible,
type StateActionExecuteIfPossible struct{}

// /// Don't execute or import state.
// Skip,
type StateActionSkip struct{}

// / Data required to import a Block.
// pub struct BlockImportParams<Block: BlockT, Transaction> {
type BlockImportParams[H runtime.Hash, N runtime.Number] struct {
	/// Origin of the Block
	// pub origin: BlockOrigin,
	Origin BlockOrigin
	/// The header, without consensus post-digests applied. This should be in the same
	/// state as it comes out of the runtime.
	///
	/// Consensus engines which alter the header (by adding post-runtime digests)
	/// should strip those off in the initial verification process and pass them
	/// via the `post_digests` field. During block authorship, they should
	/// not be pushed to the header directly.
	///
	/// The reason for this distinction is so the header can be directly
	/// re-executed in a runtime that checks digest equivalence -- the
	/// post-runtime digests are pushed back on after.
	// pub header: Block::Header,
	Header runtime.Header[N, H]
	/// Justification(s) provided for this block from the outside.
	// pub justifications: Option<Justifications>,
	Justifications *runtime.Justifications
	/// Digest items that have been added after the runtime for external
	/// work, like a consensus signature.
	// pub post_digests: Vec<DigestItem>,
	PostDigests []runtime.DigestItem
	/// The body of the block.
	// pub body: Option<Vec<Block::Extrinsic>>,
	Body *[]runtime.Extrinsic
	/// Indexed transaction body of the block.
	// pub indexed_body: Option<Vec<Vec<u8>>>,
	IndexedBody *[][]byte
	/// Specify how the new state is computed.
	// pub state_action: StateAction<Block, Transaction>,
	StateAction StateAction
	/// Is this block finalized already?
	/// `true` implies instant finality.
	// pub finalized: bool,
	Finalized bool
	/// Intermediate values that are interpreted by block importers. Each block importer,
	/// upon handling a value, removes it from the intermediate list. The final block importer
	/// rejects block import if there are still intermediate values that remain unhandled.
	// pub intermediates: HashMap<Cow<'static, [u8]>, Box<dyn Any + Send>>,
	Intermediates map[string]any
	/// Auxiliary consensus data produced by the block.
	/// Contains a list of key-value pairs. If values are `None`, the keys will be deleted. These
	/// changes will be applied to `AuxStore` database all as one batch, which is more efficient
	/// than updating `AuxStore` directly.
	// pub auxiliary: Vec<(Vec<u8>, Option<Vec<u8>>)>,
	Auxiliary []struct {
		Key  []byte
		Data *[]byte
	}
	/// Fork choice strategy of this import. This should only be set by a
	/// synchronous import, otherwise it may race against other imports.
	/// `None` indicates that the current verifier or importer cannot yet
	/// determine the fork choice value, and it expects subsequent importer
	/// to modify it. If `None` is passed all the way down to bottom block
	/// importer, the import fails with an `IncompletePipeline` error.
	// pub fork_choice: Option<ForkChoiceStrategy>,
	ForkChoice *ForkChoiceStrategy
	/// Re-validate existing block.
	// pub import_existing: bool,
	ImportExisting bool
	/// Cached full header hash (with post-digests applied).
	// pub post_hash: Option<Block::Hash>,
	PostHash *H
}

// /// Block import trait.
// #[async_trait::async_trait]
// pub trait BlockImport<B: BlockT> {
type BlockImport[H runtime.Hash, N runtime.Number] interface {
	// 	/// The error type.
	// 	type Error: std::error::Error + Send + 'static;
	// 	/// The transaction type used by the backend.
	// 	type Transaction: Send + 'static;

	/// Check block preconditions.
	// 	async fn check_block(
	// 		&mut self,
	// 		block: BlockCheckParams<B>,
	// 	) -> Result<ImportResult, Self::Error>;
	CheckBlock(block BlockCheckParams[H, N]) chan<- struct {
		ImportResult
		Error error
	}

	/// Import a block.
	// async fn import_block(
	//
	//	&mut self,
	//	block: BlockImportParams<B, Self::Transaction>,
	//
	// ) -> Result<ImportResult, Self::Error>;
	ImportBlock(block BlockImportParams[H, N]) chan<- struct {
		ImportResult
		Error error
	}
}
