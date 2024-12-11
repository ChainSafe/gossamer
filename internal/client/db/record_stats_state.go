package db

// / State abstraction for recording stats about state access.
// pub struct RecordStatsState<S, B: BlockT> {
type recordStatsState[H, S any] struct {
	/// Usage statistics
	// usage: StateUsageStats,
	/// State machine registered stats
	// overlay_stats: sp_state_machine::StateMachineStats,
	/// Backing state.
	state S
	/// The hash of the block is state belongs to.
	// block_hash: Option<B::Hash>,
	blockHash *H
	/// The usage statistics of the backend. These will be updated on drop.
	// state_usage: Arc<StateUsageStats>,
}

func newRecordStatsState[H, S any](state S, blockHash *H, stateUsage *stateUsageStats) recordStatsState[H, S] {
	return recordStatsState[H, S]{
		state:     state,
		blockHash: blockHash,
	}
}
