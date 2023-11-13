package blockchain

// / Hash and number of a block.
type HashNumber[H any, N any] struct {
	/// The hash of the block.
	Hash H
	/// The number of the block.
	Number N
}

// / A tree-route from one block to another in the chain.
// /
// / All blocks prior to the pivot in the deque is the reverse-order unique ancestry
// / of the first block, the block at the pivot index is the common ancestor,
// / and all blocks after the pivot is the ancestry of the second block, in
// / order.
// /
// / The ancestry sets will include the given blocks, and thus the tree-route is
// / never empty.
// /
// / ```text
// / Tree route from R1 to E2. Retracted is [R1, R2, R3], Common is C, enacted [E1, E2]
// /   <- R3 <- R2 <- R1
// /  /
// / C
// /  \-> E1 -> E2
// / ```
// /
// / ```text
// / Tree route from C to E2. Retracted empty. Common is C, enacted [E1, E2]
// / C -> E1 -> E2
// / ```
// #[derive(Debug, Clone)]
//
//	pub struct TreeRoute<Block: BlockT> {
//		route: Vec<HashAndNumber<Block>>,
//		pivot: usize,
//	}
type TreeRoute[H, N any] struct {
	route []HashNumber[H, N]
	pivot uint
}
