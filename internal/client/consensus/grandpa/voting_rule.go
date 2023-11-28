package grandpa

import (
	"github.com/ChainSafe/gossamer/internal/primitives/blockchain"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
)

// / A future returned by a `VotingRule` to restrict a given vote, if any restriction is necessary.
// pub type VotingRuleResult<Block> =
// Pin<Box<dyn Future<Output = Option<(<Block as BlockT>::Hash, NumberFor<Block>)>> + Send>>;
type VotingRuleResult[H, N any] chan<- *struct {
	Hash   H
	Number N
}

// / A trait for custom voting rules in GRANDPA.
// pub trait VotingRule<Block, B>: DynClone + Send + Sync
// where
//
//	Block: BlockT,
//	B: HeaderBackend<Block>,
//
//	{
type VotingRule[H runtime.Hash, N runtime.Number] interface {
	/// Restrict the given `current_target` vote, returning the block hash and
	/// number of the block to vote on, and `None` in case the vote should not
	/// be restricted. `base` is the block that we're basing our votes on in
	/// order to pick our target (e.g. last round estimate), and `best_target`
	/// is the initial best vote target before any vote rules were applied. When
	/// applying multiple `VotingRule`s both `base` and `best_target` should
	/// remain unchanged.
	///
	/// The contract of this interface requires that when restricting a vote, the
	/// returned value **must** be an ancestor of the given `current_target`,
	/// this also means that a variant must be maintained throughout the
	/// execution of voting rules wherein `current_target <= best_target`.
	// fn restrict_vote(
	// 	&self,
	// 	backend: Arc<B>,
	// 	base: &Block::Header,
	// 	best_target: &Block::Header,
	// 	current_target: &Block::Header,
	// ) -> VotingRuleResult<Block>;
	RestrictVote(
		backend blockchain.HeaderBackend[H, N],
		base runtime.Header[N, H],
		bestTarget runtime.Header[N, H],
		currentTarget runtime.Header[N, H],
	) VotingRuleResult[H, N]
}
