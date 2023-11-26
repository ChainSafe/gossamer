package storage

import "github.com/tidwall/btree"

// / Storage key.
type StorageKey []byte

// / Storage key with read/write tracking information.
type TrackedStorageKey struct {
	Key         []byte
	Reads       uint32
	Writes      uint32
	Whitelisted bool
}

// / Storage data associated to a [`StorageKey`].
type StorageData []byte

// / Child trie storage data.
type StorageChild struct {
	/// Child data for storage.
	Data btree.Map[string, []byte]
	/// Associated child info for a child
	/// trie.
	ChildInfo ChildInfo
}

// / Struct containing data needed for a storage.
// #[cfg(feature = "std")]
// #[derive(Default, Debug, Clone)]
type Storage struct {
	/// Top trie storage data.
	Top btree.Map[string, []byte]
	/// Children trie storage data. Key does not include prefix, only for the `default` trie kind,
	/// of `ChildType::ParentKeyId` type.
	ChildrenDefault map[string]StorageChild
}

// / Information related to a child state.
type ChildInfo any
type ChildInfos interface {
	ChildInfoParentKeyID
}

// / This is the one used by default.
type ChildInfoParentKeyID ChildTrieParentKeyID

// / A child trie of default type.
// /
// / It uses the same default implementation as the top trie, top trie being a child trie with no
// / keyspace and no storage key. Its keyspace is the variable (unprefixed) part of its storage key.
// / It shares its trie nodes backend storage with every other child trie, so its storage key needs
// / to be a unique id that will be use only once. Those unique id also required to be long enough to
// / avoid any unique id to be prefixed by an other unique id.
type ChildTrieParentKeyID struct {
	/// Data is the storage key without prefix.
	data []byte
}

// / Different possible state version.
// /
// / V0 and V1 uses a same trie implementation, but V1 will write external value node in the trie for
// / value with size at least `TRIE_VALUE_NODE_THRESHOLD`.
// #[derive(Debug, Clone, Copy, Eq, PartialEq)]
// #[cfg_attr(feature = "std", derive(Encode, Decode))]
//
//	pub enum StateVersion {
//		/// Old state version, no value nodes.
//		V0 = 0,
//		/// New state version can use value nodes.
//		V1 = 1,
//	}
type StateVersion uint

const (
	StateVersionV0 StateVersion = iota
	StateVersionV1
)
