package storage

import (
	"strings"

	"github.com/tidwall/btree"
)

// / Storage key.
type StorageKey []byte

// / Storage key with read/write tracking information.
type TrackedStorageKey struct {
	Key         []byte
	Reads       uint32
	Writes      uint32
	Whitelisted bool
}

// / Storage key of a child trie, it contains the prefix to the key.
type PrefixedStorageKey []byte

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

// / Instantiates child information for a default child trie
// / of kind `ChildType::ParentKeyId`, using an unprefixed parent
// / storage key.
func NewDefaultChildInfo(storageKey []byte) ChildInfo {
	return ChildInfoParentKeyID{
		data: storageKey,
	}
}

// / Type of child.
// / It does not strictly define different child type, it can also
// / be related to technical consideration or api variant.
type ChildType uint32

const (
	/// If runtime module ensures that the child key is a unique id that will
	/// only be used once, its parent key is used as a child trie unique id.
	ChildTypeParentKeyID ChildType = iota + 1
)

// impl ChildType {
// 	/// Try to get a child type from its `u32` representation.
// 	pub fn new(repr: u32) -> Option<ChildType> {
// 		Some(match repr {
// 			r if r == ChildType::ParentKeyId as u32 => ChildType::ParentKeyId,
// 			_ => return None,
// 		})
// 	}

// /// Transform a prefixed key into a tuple of the child type
// /// and the unprefixed representation of the key.
//
//	pub fn from_prefixed_key<'a>(storage_key: &'a PrefixedStorageKey) -> Option<(Self, &'a [u8])> {
//		let match_type = |storage_key: &'a [u8], child_type: ChildType| {
//			let prefix = child_type.parent_prefix();
//			if storage_key.starts_with(prefix) {
//				Some((child_type, &storage_key[prefix.len()..]))
//			} else {
//				None
//			}
//		};
//		match_type(storage_key, ChildType::ParentKeyId)
//	}
func NewChildTypeFromPrefixedKey(storageKey PrefixedStorageKey) *struct {
	ChildType
	Key []byte
} {
	childType := ChildTypeParentKeyID
	prefix := childType.ParentPrefix()
	if strings.Index(string(storageKey), string(prefix)) == 0 {
		return &struct {
			ChildType
			Key []byte
		}{childType, storageKey[len(prefix):]}
	} else {
		return nil
	}
}

// 	/// Produce a prefixed key for a given child type.
// 	fn new_prefixed_key(&self, key: &[u8]) -> PrefixedStorageKey {
// 		let parent_prefix = self.parent_prefix();
// 		let mut result = Vec::with_capacity(parent_prefix.len() + key.len());
// 		result.extend_from_slice(parent_prefix);
// 		result.extend_from_slice(key);
// 		PrefixedStorageKey(result)
// 	}

// 	/// Prefixes a vec with the prefix for this child type.
// 	fn do_prefix_key(&self, key: &mut Vec<u8>) {
// 		let parent_prefix = self.parent_prefix();
// 		let key_len = key.len();
// 		if !parent_prefix.is_empty() {
// 			key.resize(key_len + parent_prefix.len(), 0);
// 			key.copy_within(..key_len, parent_prefix.len());
// 			key[..parent_prefix.len()].copy_from_slice(parent_prefix);
// 		}
// 	}

// 	/// Returns the location reserved for this child trie in their parent trie if there
// 	/// is one.
// 	pub fn parent_prefix(&self) -> &'static [u8] {
// 		match self {
// 			&ChildType::ParentKeyId => well_known_keys::DEFAULT_CHILD_STORAGE_KEY_PREFIX,
// 		}
// 	}
// }

// / Prefix of the default child storage keys in the top trie.
// pub const DEFAULT_CHILD_STORAGE_KEY_PREFIX: &[u8] = b":child_storage:default:";
var DefaultChildStorageKeyPrefix = []byte(":child_storage:default:")

func (ct ChildType) ParentPrefix() []byte {
	switch ct {
	case ChildTypeParentKeyID:
		return DefaultChildStorageKeyPrefix
	default:
		panic("wtf?")
	}
}

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
