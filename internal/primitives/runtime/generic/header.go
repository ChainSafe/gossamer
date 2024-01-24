package generic

import (
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// / Abstraction over a block header for a substrate chain.
type Header[N runtime.Number, H runtime.Hash] struct {
	/// The parent hash.
	// pub parent_hash: Hash::Output,
	parentHash H
	/// The block number.
	// #[cfg_attr(
	// 	feature = "std",
	// 	serde(serialize_with = "serialize_number", deserialize_with = "deserialize_number")
	// )]
	// #[codec(compact)]
	// pub number: Number,
	number N
	/// The state trie merkle root
	// pub state_root: Hash::Output,
	stateRoot H
	/// The merkle root of the extrinsics.
	// pub extrinsics_root: Hash::Output,
	extrinsicsRoot H
	/// A chain-specific digest of data useful for light clients or referencing auxiliary data.
	digest runtime.Digest

	// store associated Hashing
	hasher runtime.Hasher[H]
}

func (h Header[N, H]) Number() N {
	return h.number
}

// / Sets the header number.
func (h *Header[N, H]) SetNumber(number N) {
	h.number = number
}

// / Returns a reference to the extrinsics root.
func (h Header[N, H]) ExtrinsicsRoot() H {
	return h.extrinsicsRoot
}

// / Sets the extrinsic root.
func (h *Header[N, H]) SetExtrinsicsRoot(root H) {
	h.extrinsicsRoot = root
}

// / Returns a reference to the state root.
func (h Header[N, H]) StateRoot() H {
	return h.stateRoot
}

// / Sets the state root.
func (h *Header[N, H]) SetStateRoot(root H) {
	h.stateRoot = root
}

// / Returns a reference to the parent hash.
func (h Header[N, H]) ParentHash() H {
	return h.parentHash
}

// / Sets the parent hash.
func (h *Header[N, H]) SetParentHash(hash H) {
	h.parentHash = hash
}

// / Returns a reference to the digest.
func (h Header[N, H]) Digest() runtime.Digest {
	return h.digest
}

// / Get a mutable reference to the digest.
func (h Header[N, H]) DigestMut() *runtime.Digest {
	return &h.digest
}

func (h Header[N, H]) MarshalSCALE() ([]byte, error) {
	type helper struct {
		ParentHash H
		// uses compact encoding so we need to cast to uint
		// https://github.com/paritytech/substrate/blob/e374a33fe1d99d59eb24a08981090bdb4503e81b/primitives/runtime/src/generic/header.rs#L47
		Number         uint
		StateRoot      H
		ExtrinsicsRoot H
		Digest         runtime.Digest
	}
	help := helper{h.parentHash, uint(h.number), h.stateRoot, h.extrinsicsRoot, h.digest}
	return scale.Marshal(help)
}

// / Returns the hash of the header.
func (h Header[N, H]) Hash() H {
	return h.hasher.HashOf(h)
}

func (h Header[N, H]) Hasher() runtime.Hasher[H] {
	return h.hasher
}

func NewHeader[N runtime.Number, H runtime.Hash](
	number N,
	extrinsicsRoot H,
	stateRoot H,
	parentHash H,
	digest runtime.Digest,
	hasher runtime.Hasher[H],
) Header[N, H] {
	return Header[N, H]{
		number:         number,
		extrinsicsRoot: extrinsicsRoot,
		stateRoot:      stateRoot,
		parentHash:     parentHash,
		digest:         digest,
		hasher:         hasher,
	}
}
