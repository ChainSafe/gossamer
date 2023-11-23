package generic

// / Something to identify a block.
// #[derive(PartialEq, Eq, Clone, Encode, Decode, RuntimeDebug)]
// pub enum BlockId<Block: BlockT> {
type BlockID any
type BlockIDs[H, N any] interface {
	BlockIDHash[H] | BlockIDNumber[N]
}

// / Identify by block header hash.
//
//	Hash(Block::Hash),
type BlockIDHash[H any] struct {
	Inner H
}

// / Identify by block number.
//
//	Number(NumberFor<Block>),
type BlockIDNumber[N any] struct {
	Inner N
}
