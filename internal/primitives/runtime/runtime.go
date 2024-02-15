package runtime

import (
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hashing"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"golang.org/x/exp/constraints"
)

// / An abstraction over justification for a block's validity under a consensus algorithm.
// /
// / Essentially a finality proof. The exact formulation will vary between consensus
// / algorithms. In the case where there are multiple valid proofs, inclusion within
// / the block itself would allow swapping justifications to change the block's hash
// / (and thus fork the chain). Sending a `Justification` alongside a block instead
// / bypasses this problem.
// /
// / Each justification is provided as an encoded blob, and is tagged with an ID
// / to identify the consensus engine that generated the proof (we might have
// / multiple justifications from different engines for the same block).
// pub type Justification = (ConsensusEngineId, EncodedJustification);
type Justification struct {
	ConsensusEngineID
	EncodedJustification
}

// / The encoded justification specific to a consensus engine.
// pub type EncodedJustification = Vec<u8>;
type EncodedJustification []byte

// / Collection of justifications for a given block, multiple justifications may
// / be provided by different consensus engines for the same block.
// pub struct Justifications(Vec<Justification>);
type Justifications []Justification

// IntoJustification returns a copy of the encoded justification for the given consensus
// engine, if it exists
func (j Justifications) IntoJustification(engineID ConsensusEngineID) *EncodedJustification {
	for _, justification := range j {
		if justification.ConsensusEngineID == engineID {
			return &justification.EncodedJustification
		}
	}
	return nil
}

// / Consensus engine unique ID.
// pub type ConsensusEngineId = [u8; 4];
type ConsensusEngineID [4]byte

// / Header number.
type Number interface {
	~uint | ~uint32 | ~uint64
}

type Hash interface {
	constraints.Ordered

	Bytes() []byte
	String() string
	// scale.Marshaler
	// scale.Unmarshaler
}

// / Abstraction around hashing
// Stupid bug in the Rust compiler believes derived
// traits must be fulfilled by all type parameters.
type Hasher[H Hash] interface {
	// /// Produce the hash of some byte-slice.
	// fn hash(s: &[u8]) -> Self::Output {
	// 	<Self as Hasher>::hash(s)
	// }
	Hash(s []byte) H

	// /// Produce the hash of some codec-encodable value.
	// fn hash_of<S: Encode>(s: &S) -> Self::Output {
	// 	Encode::using_encoded(s, <Self as Hasher>::hash)
	// }
	HashOf(s any) H

	// /// The ordered Patricia tree root of the given `input`.
	// fn ordered_trie_root(input: Vec<Vec<u8>>, state_version: StateVersion) -> Self::Output;

	// /// The Patricia tree root of the given mapping.
	// fn trie_root(input: Vec<(Vec<u8>, Vec<u8>)>, state_version: StateVersion) -> Self::Output;
}

// / Blake2-256 Hash implementation.
type BlakeTwo256 struct{}

// / Produce the hash of some byte-slice.
func (bt256 BlakeTwo256) Hash(s []byte) hash.H256 {
	h := hashing.Blake2_256(s)
	return hash.H256(h[:])
}

// / Produce the hash of some codec-encodable value.
func (bt256 BlakeTwo256) HashOf(s any) hash.H256 {
	bytes := scale.MustMarshal(s)
	return bt256.Hash(bytes)
}

var _ Hasher[hash.H256] = BlakeTwo256{}

// / Something which fulfills the abstract idea of a Substrate header. It has types for a `Number`,
// / a `Hash` and a `Hashing`. It provides access to an `extrinsics_root`, `state_root` and
// / `parent_hash`, as well as a `digest` and a block `number`.
// /
// / You can also create a `new` one from those fields.
type Header[N Number, H Hash] interface {
	/// Returns a reference to the header number.
	Number() N
	/// Sets the header number.
	SetNumber(number N)

	/// Returns a reference to the extrinsics root.
	ExtrinsicsRoot() H
	/// Sets the extrinsic root.
	SetExtrinsicsRoot(root H)

	/// Returns a reference to the state root.
	StateRoot() H
	/// Sets the state root.
	SetStateRoot(root H)

	/// Returns a reference to the parent hash.
	ParentHash() H
	/// Sets the parent hash.
	SetParentHash(hash H)

	/// Returns a reference to the digest.
	Digest() Digest
	/// Get a mutable reference to the digest.
	DigestMut() *Digest

	/// Returns the hash of the header.
	Hash() H
}

// / Something which fulfills the abstract idea of a Substrate block. It has types for
// / `Extrinsic` pieces of information as well as a `Header`.
// /
// / You can get an iterator over each of the `extrinsics` and retrieve the `header`.
type Block[N Number, H Hash] interface {
	/// Returns a reference to the header.
	Header() Header[N, H]
	/// Returns a reference to the list of extrinsics.
	Extrinsics() []Extrinsic
	/// Split the block into header and list of extrinsics.
	Deconstruct() (header Header[N, H], extrinsics []Extrinsic)
	/// Returns the hash of the block.
	Hash() H
	/// Creates an encoded block from the given `header` and `extrinsics` without requiring the
	/// creation of an instance.
	// fn encode_from(header: &Self::Header, extrinsics: &[Self::Extrinsic]) -> Vec<u8>;
}

// / Something that acts like an `Extrinsic`.
type Extrinsic interface {
	/// Is this `Extrinsic` signed?
	/// If no information are available about signed/unsigned, `None` should be returned.
	IsSigned() *bool

	/// Create new instance of the extrinsic.
	///
	/// Extrinsics can be split into:
	/// 1. Inherents (no signature; created by validators during block production)
	/// 2. Unsigned Transactions (no signature; represent "system calls" or other special kinds of
	/// calls) 3. Signed Transactions (with signature; a regular transactions with known origin)
	// fn new(_call: Self::Call, _signed_data: Option<Self::SignaturePayload>) -> Option<Self> {
	// 	None
	// }
}
