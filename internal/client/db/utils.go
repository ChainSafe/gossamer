package db

// / Database metadata.
type meta[N, H any] struct {
	/// Hash of the best known block.
	BestHash H
	/// Number of the best known block.
	BestNumber N
	/// Hash of the best finalized block.
	FinalizedHash H
	/// Number of the best finalized block.
	FinalizedNumber N
	/// Hash of the genesis block.
	GenesisHash H
	/// Finalized state, if any
	FinalizedState *struct {
		Hash   H
		Number N
	}
	/// Block gap, start and end inclusive, if any.
	BlockGap *struct {
		Hash   H
		Number N
	}
}
