// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package storage

import (
	"strings"

	"github.com/ChainSafe/gossamer/pkg/trie"
)

// Storage key.
type StorageKey []byte

// Storage key of a child trie, it contains the prefix to the key.
type PrefixedStorageKey []byte

// Information related to a child state.
type ChildInfo interface {
	// Returns byte sequence (keyspace) that can be use by underlying db to isolate keys.
	// This is a unique id of the child trie. The collision resistance of this value
	// depends on the type of child info use. For `ChildInfo::Default` it is and need to be.
	Keyspace() []byte
	// Returns a reference to the location in the direct parent of
	// this trie but without the common prefix for this kind of
	// child trie.
	StorageKey() StorageKey
	// Return a the full location in the direct parent of
	// this trie.
	PrefixedStorageKey() PrefixedStorageKey
	// Returns the type for this child info.
	ChildType() ChildType
}

// This is the one used by default.
type ChildInfoParentKeyID ChildTrieParentKeyID

// Returns byte sequence (keyspace) that can be use by underlying db to isolate keys.
// This is a unique id of the child trie. The collision resistance of this value
// depends on the type of child info use.
func (cipkid ChildInfoParentKeyID) Keyspace() []byte {
	return cipkid.StorageKey()
}

// Returns a reference to the location in the direct parent of
// this trie but without the common prefix for this kind of
// child trie.
func (cipkid ChildInfoParentKeyID) StorageKey() StorageKey {
	return ChildTrieParentKeyID(cipkid).data
}

// Return a the full location in the direct parent of
// this trie.
func (cipkid ChildInfoParentKeyID) PrefixedStorageKey() PrefixedStorageKey {
	return ChildTypeParentKeyID.NewPrefixedKey(cipkid.data)
}

// Returns the type for this child info.
func (cipkid ChildInfoParentKeyID) ChildType() ChildType {
	return ChildTypeParentKeyID
}

// Instantiates child information for a default child trie
// of kind `ChildType::ParentKeyId`, using an unprefixed parent
// storage key.
func NewDefaultChildInfo(storageKey []byte) ChildInfo {
	return ChildInfoParentKeyID{
		data: storageKey,
	}
}

// Type of child.
// It does not strictly define different child type, it can also
// be related to technical consideration or api variant.
type ChildType uint32

const (
	// If runtime module ensures that the child key is a unique id that will
	// only be used once, its parent key is used as a child trie unique id.
	ChildTypeParentKeyID ChildType = iota + 1
)

// Transform a prefixed key into a tuple of the child type
// and the unprefixed representation of the key.
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

// Produce a prefixed key for a given child type.
func (ct ChildType) NewPrefixedKey(key []byte) PrefixedStorageKey {
	parentPrefix := ct.ParentPrefix()
	result := append(parentPrefix, key...)
	return PrefixedStorageKey(result)
}

// Prefix of the default child storage keys in the top trie.
var DefaultChildStorageKeyPrefix = []byte(":child_storage:default:")

// Returns the location reserved for this child trie in their parent trie if there
// is one.
func (ct ChildType) ParentPrefix() []byte {
	switch ct {
	case ChildTypeParentKeyID:
		return DefaultChildStorageKeyPrefix
	default:
		panic("unreachable")
	}
}

// A child trie of default type.
//
// It uses the same default implementation as the top trie, top trie being a child trie with no
// keyspace and no storage key. Its keyspace is the variable (unprefixed) part of its storage key.
// It shares its trie nodes backend storage with every other child trie, so its storage key needs
// to be a unique id that will be use only once. Those unique id also required to be long enough to
// avoid any unique id to be prefixed by an other unique id.
type ChildTrieParentKeyID struct {
	// Data is the storage key without prefix.
	data []byte
}

// Different possible state version.
//
// V0 and V1 uses a same trie implementation, but V1 will write external value node in the trie for
// value with size greater than 32 bytes.
type StateVersion uint

const (
	StateVersionV0 StateVersion = iota
	StateVersionV1
)

func (svv StateVersion) TrieLayout() trie.TrieLayout {
	switch svv {
	case StateVersionV0:
		return trie.V0
	case StateVersionV1:
		return trie.V1
	default:
		panic("unreachable")
	}
}
