package trie

import (
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hasher"
	"github.com/ChainSafe/gossamer/internal/primitives/kv"
	"github.com/ChainSafe/gossamer/internal/primitives/storage"
	"github.com/ChainSafe/gossamer/internal/primitives/trie"
)

type KeyValue = kv.KeyValue

/// Interface that provides trie related functionality.
// #[runtime_interface]
// pub trait Trie {
// 	/// A trie root formed from the iterated items.
// 	fn blake2_256_root(input: Vec<(Vec<u8>, Vec<u8>)>) -> H256 {
// 		LayoutV0::<sp_core::Blake2Hasher>::trie_root(input)
// 	}

// /// A trie root formed from the iterated items.
// #[version(2)]
//
//	fn blake2_256_root(input: Vec<(Vec<u8>, Vec<u8>)>, version: StateVersion) -> H256 {
//		match version {
//			StateVersion::V0 => LayoutV0::<sp_core::Blake2Hasher>::trie_root(input),
//			StateVersion::V1 => LayoutV1::<sp_core::Blake2Hasher>::trie_root(input),
//		}
//	}
func BlakeTwo256Root(input []KeyValue, version storage.StateVersion) hash.H256 {
	switch version {
	case storage.StateVersionV0:
		return trie.LayoutV0[hasher.Blake2Hasher, hash.H256]{}.TrieRoot(input)
	case storage.StateVersionV1:
		return trie.LayoutV1[hasher.Blake2Hasher, hash.H256]{}.TrieRoot(input)
	default:
		panic("unreachable")
	}
	panic("unreachable")
}

// 	/// A trie root formed from the enumerated items.
// 	fn blake2_256_ordered_root(input: Vec<Vec<u8>>) -> H256 {
// 		LayoutV0::<sp_core::Blake2Hasher>::ordered_trie_root(input)
// 	}

// 	/// A trie root formed from the enumerated items.
// 	#[version(2)]
// 	fn blake2_256_ordered_root(input: Vec<Vec<u8>>, version: StateVersion) -> H256 {
// 		match version {
// 			StateVersion::V0 => LayoutV0::<sp_core::Blake2Hasher>::ordered_trie_root(input),
// 			StateVersion::V1 => LayoutV1::<sp_core::Blake2Hasher>::ordered_trie_root(input),
// 		}
// 	}

// 	/// A trie root formed from the iterated items.
// 	fn keccak_256_root(input: Vec<(Vec<u8>, Vec<u8>)>) -> H256 {
// 		LayoutV0::<sp_core::KeccakHasher>::trie_root(input)
// 	}

// 	/// A trie root formed from the iterated items.
// 	#[version(2)]
// 	fn keccak_256_root(input: Vec<(Vec<u8>, Vec<u8>)>, version: StateVersion) -> H256 {
// 		match version {
// 			StateVersion::V0 => LayoutV0::<sp_core::KeccakHasher>::trie_root(input),
// 			StateVersion::V1 => LayoutV1::<sp_core::KeccakHasher>::trie_root(input),
// 		}
// 	}

// 	/// A trie root formed from the enumerated items.
// 	fn keccak_256_ordered_root(input: Vec<Vec<u8>>) -> H256 {
// 		LayoutV0::<sp_core::KeccakHasher>::ordered_trie_root(input)
// 	}

// 	/// A trie root formed from the enumerated items.
// 	#[version(2)]
// 	fn keccak_256_ordered_root(input: Vec<Vec<u8>>, version: StateVersion) -> H256 {
// 		match version {
// 			StateVersion::V0 => LayoutV0::<sp_core::KeccakHasher>::ordered_trie_root(input),
// 			StateVersion::V1 => LayoutV1::<sp_core::KeccakHasher>::ordered_trie_root(input),
// 		}
// 	}

// 	/// Verify trie proof
// 	fn blake2_256_verify_proof(root: H256, proof: &[Vec<u8>], key: &[u8], value: &[u8]) -> bool {
// 		sp_trie::verify_trie_proof::<LayoutV0<sp_core::Blake2Hasher>, _, _, _>(
// 			&root,
// 			proof,
// 			&[(key, Some(value))],
// 		)
// 		.is_ok()
// 	}

// 	/// Verify trie proof
// 	#[version(2)]
// 	fn blake2_256_verify_proof(
// 		root: H256,
// 		proof: &[Vec<u8>],
// 		key: &[u8],
// 		value: &[u8],
// 		version: StateVersion,
// 	) -> bool {
// 		match version {
// 			StateVersion::V0 => sp_trie::verify_trie_proof::<
// 				LayoutV0<sp_core::Blake2Hasher>,
// 				_,
// 				_,
// 				_,
// 			>(&root, proof, &[(key, Some(value))])
// 			.is_ok(),
// 			StateVersion::V1 => sp_trie::verify_trie_proof::<
// 				LayoutV1<sp_core::Blake2Hasher>,
// 				_,
// 				_,
// 				_,
// 			>(&root, proof, &[(key, Some(value))])
// 			.is_ok(),
// 		}
// 	}

// 	/// Verify trie proof
// 	fn keccak_256_verify_proof(root: H256, proof: &[Vec<u8>], key: &[u8], value: &[u8]) -> bool {
// 		sp_trie::verify_trie_proof::<LayoutV0<sp_core::KeccakHasher>, _, _, _>(
// 			&root,
// 			proof,
// 			&[(key, Some(value))],
// 		)
// 		.is_ok()
// 	}

// 	/// Verify trie proof
// 	#[version(2)]
// 	fn keccak_256_verify_proof(
// 		root: H256,
// 		proof: &[Vec<u8>],
// 		key: &[u8],
// 		value: &[u8],
// 		version: StateVersion,
// 	) -> bool {
// 		match version {
// 			StateVersion::V0 => sp_trie::verify_trie_proof::<
// 				LayoutV0<sp_core::KeccakHasher>,
// 				_,
// 				_,
// 				_,
// 			>(&root, proof, &[(key, Some(value))])
// 			.is_ok(),
// 			StateVersion::V1 => sp_trie::verify_trie_proof::<
// 				LayoutV1<sp_core::KeccakHasher>,
// 				_,
// 				_,
// 				_,
// 			>(&root, proof, &[(key, Some(value))])
// 			.is_ok(),
// 		}
// 	}
// }
