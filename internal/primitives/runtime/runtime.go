package runtime

import "golang.org/x/exp/constraints"

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

// / A pre-runtime digest.
// /
// / These are messages from the consensus engine to the runtime, although
// / the consensus engine can (and should) read them itself to avoid
// / code and state duplication. It is erroneous for a runtime to produce
// / these, but this is not (yet) checked.
// /
// / NOTE: the runtime is not allowed to panic or fail in an `on_initialize`
// / call if an expected `PreRuntime` digest is not present. It is the
// / responsibility of a external block verifier to check this. Runtime API calls
// / will initialize the block without pre-runtime digests, so initialization
// / cannot fail when they are missing.
type PreRuntime struct {
	ConsensusEngineID
	Bytes []byte
}

// / A message from the runtime to the consensus engine. This should *never*
// / be generated by the native code of any consensus engine, but this is not
// / checked (yet).
type Consensus struct {
	ConsensusEngineID
	Bytes []byte
}

// / Put a Seal on it. This is only used by native code, and is never seen
// / by runtimes.
type Seal struct {
	ConsensusEngineID
	Bytes []byte
}

// / Some other thing. Unsupported and experimental.
type Other []byte

// / An indication for the light clients that the runtime execution
// / environment is updated.
// /
type RuntimeEnvironmentUpdated struct{}

// / Digest item that is able to encode/decode 'system' digest items and
// / provide opaque access to other items.
type DigestItems interface {
	PreRuntime | Consensus | Seal | Other | RuntimeEnvironmentUpdated
}

// / Digest item that is able to encode/decode 'system' digest items and
// / provide opaque access to other items.
type DigestItem any

// / Generic header digest.
// #[derive(PartialEq, Eq, Clone, Encode, Decode, RuntimeDebug, TypeInfo, Default)]
// #[cfg_attr(feature = "std", derive(Serialize, Deserialize))]
//
//	pub struct Digest {
//		/// A list of logs in the digest.
//		pub logs: Vec<DigestItem>,
//	}
type Digest struct {
	/// A list of logs in the digest.
	Logs []DigestItem
}

// / Header number.
type Number interface {
	~uint32 | ~uint64
}

// / Something which fulfills the abstract idea of a Substrate header. It has types for a `Number`,
// / a `Hash` and a `Hashing`. It provides access to an `extrinsics_root`, `state_root` and
// / `parent_hash`, as well as a `digest` and a block `number`.
// /
// / You can also create a `new` one from those fields.
type Header[N Number, H Hash] interface {
	Number() N
	SetNumber(number N)

	ExtrinsicsRoot() H
	SetExtrinsicsRoot(root H)

	StateRoot() H
	SetStateRoot(root H)

	ParentHash() H
	SetParentHash(hash H)

	Digest() Digest

	Hash() H
}

// / Block hash type.
type Hash interface {
	constraints.Ordered
}

// / Something which fulfills the abstract idea of a Substrate block. It has types for
// / `Extrinsic` pieces of information as well as a `Header`.
// /
// / You can get an iterator over each of the `extrinsics` and retrieve the `header`.
type Block[N Number, H Hash] interface {
	Header() Header[N, H]
	Extrinsics() []Extrinsic
	Deconstruct() (header Header[N, H], extrinsics []Extrinsic)
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
