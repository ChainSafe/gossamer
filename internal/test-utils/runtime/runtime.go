package runtime

import (
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/core/sr25519"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime/generic"
)

// / The address format for describing accounts.
// pub type Address = sp_core::sr25519::Public;
type Address = sr25519.Public

// pub type Signature = sr25519::Signature;
type Signature = sr25519.Signature

// / The extension to the basic transaction logic.
// pub type TxExtension = (
// 	(CheckNonce<Runtime>, CheckWeight<Runtime>),
// 	CheckSubstrateCall,
// 	frame_metadata_hash_extension::CheckMetadataHash<Runtime>,
// );

type TxExtension struct{}

// / Unchecked extrinsic type as expected by this runtime.
// pub type Extrinsic =
//
//	sp_runtime::generic::UncheckedExtrinsic<Address, RuntimeCall, Signature, TxExtension>;
type Extrinsic = generic.UncheckedExtrinsic[Address, RuntimeCall, Signature, TxExtension]

// / A simple hash type for all our hashing.
// pub type Hash = H256;
type Hash = hash.H256

// / The hashing algorithm used.
// pub type Hashing = BlakeTwo256;
type Hasher = runtime.BlakeTwo256

// / The block number type used in this runtime.
// pub type BlockNumber = u64;
type BlockNumber uint64

// / A test block.
// pub type Block = sp_runtime::generic::Block<Header, Extrinsic>;
type Block = generic.Block[BlockNumber, Hash, Hasher, Extrinsic]

// / A test block's header.
// pub type Header = sp_runtime::generic::Header<BlockNumber, Hashing>;
type Header = generic.Header[BlockNumber, Hash, Hasher]

type RuntimeCall interface {
	isRuntimeCall()
}
