package generic

import (
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
)

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

// / Abstraction over a substrate block.
// #[derive(PartialEq, Eq, Clone, Encode, Decode, RuntimeDebug, scale_info::TypeInfo)]
// #[cfg_attr(feature = "std", derive(Serialize, Deserialize))]
// #[cfg_attr(feature = "std", serde(rename_all = "camelCase"))]
// #[cfg_attr(feature = "std", serde(deny_unknown_fields))]
//
//	pub struct Block<Header, Extrinsic: MaybeSerialize> {
//		/// The block header.
//		pub header: Header,
//		/// The accompanying extrinsics.
//		pub extrinsics: Vec<Extrinsic>,
//	}
type Block[N runtime.Number, H runtime.Hash, Hasher runtime.Hasher[H]] struct {
	/// The block header.
	header runtime.Header[N, H]
	/// The accompanying extrinsics.
	extrinsics []runtime.Extrinsic
}

func (b Block[N, H, Hasher]) Header() runtime.Header[N, H] {
	return b.header
}

func (b Block[N, H, Hasher]) Extrinsics() []runtime.Extrinsic {
	return b.extrinsics
}

func (b Block[N, H, Hasher]) Deconstruct() (header runtime.Header[N, H], extrinsics []runtime.Extrinsic) {
	panic("unimplemented")
	return nil, nil
}

func (b Block[N, H, Hasher]) Hash() H {
	hasher := *new(Hasher)
	return hasher.HashOf(b.header)
}

var _ runtime.Block[uint, hash.H256] = Block[uint, hash.H256, runtime.BlakeTwo256]{}

func NewBlock[N runtime.Number, H runtime.Hash, Hasher runtime.Hasher[H]](
	header runtime.Header[N, H], extrinsics []runtime.Extrinsic) Block[N, H, Hasher] {
	return Block[N, H, Hasher]{
		header:     header,
		extrinsics: extrinsics,
	}
}
