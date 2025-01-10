// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package storage

import (
	"strings"

	"github.com/ChainSafe/gossamer/internal/primitives/storage/keys"
	"github.com/ChainSafe/gossamer/pkg/trie"
	"github.com/tidwall/btree"
)

// StorageKey is a storage key.
type StorageKey []byte

// PrefixedStorageKey is a storage key of a child trie, it contains the prefix to the key.
type PrefixedStorageKey []byte

// StorageChild is child trie storage data.
type StorageChild struct {
	Data      btree.Map[string, []byte] // Child data for storage.
	ChildInfo ChildInfo                 // Associated child info for a child trie.
}

// Storage contains data needed for a storage.
type Storage struct {
	// Top trie storage data.
	Top btree.Map[string, []byte]
	// Children trie storage data. Key does not include prefix, only for the default trie kind,
	// of [ChildTypeParentKeyID] type.
	ChildrenDefault map[string]StorageChild
}

// ChildInfo is information related to a child state.
type ChildInfo interface {
	// Keyspace returns byte sequence (keyspace) that can be use by underlying db to isolate keys.
	// This is a unique id of the child trie. The collision resistance of this value
	// depends on the type of child info use. For [ChildTypeParentKeyID] it is and need to be.
	Keyspace() []byte
	// StorageKey returns a reference to the location in the direct parent of
	// this trie but without the common prefix for this kind of child trie.
	StorageKey() StorageKey
	// PrefixedStorageKey returns the full location in the direct parent of this trie.
	PrefixedStorageKey() PrefixedStorageKey
	// ChildType returns the type for this child info.
	ChildType() ChildType
}

// ChildInfoParentKeyID is the default ChildTrieParentKeyID.
type ChildInfoParentKeyID ChildTrieParentKeyID

// Keyspace returns byte sequence (keyspace) that can be use by underlying db to isolate keys.
// This is a unique id of the child trie. The collision resistance of this value
// depends on the type of child info use.
func (cipkid ChildInfoParentKeyID) Keyspace() []byte {
	return cipkid.StorageKey()
}

// StorageKey returns a reference to the location in the direct parent of
// this trie but without the common prefix for this kind of
// child trie.
func (cipkid ChildInfoParentKeyID) StorageKey() StorageKey {
	return ChildTrieParentKeyID(cipkid).data
}

// PrefixedStorageKey returns a the full location in the direct parent of
// this trie.
func (cipkid ChildInfoParentKeyID) PrefixedStorageKey() PrefixedStorageKey {
	return ChildTypeParentKeyID.NewPrefixedKey(cipkid.data)
}

// ChildType returns the type for this child info.
func (cipkid ChildInfoParentKeyID) ChildType() ChildType {
	return ChildTypeParentKeyID
}

// NewDefaultChildInfo instantiates child information for a default child trie
// of kind ChildInfoParentKeyID, using an unprefixed parent
// storage key.
func NewDefaultChildInfo(storageKey []byte) ChildInfo {
	return ChildInfoParentKeyID{
		data: storageKey,
	}
}

// ChildType is the type of child.
// It does not strictly define different child type, it can also
// be related to technical consideration or api variant.
type ChildType uint32

const (
	// If runtime module ensures that the child key is a unique id that will
	// only be used once, its parent key is used as a child trie unique id.
	ChildTypeParentKeyID ChildType = iota + 1
)

// NewChildTypeFromPrefixedKey transforms a prefixed key into a tuple of the child type
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

// NewPrefixedKey produces a prefixed key for a given child type.
func (ct ChildType) NewPrefixedKey(key []byte) PrefixedStorageKey {
	parentPrefix := ct.ParentPrefix()
	result := append(parentPrefix, key...)
	return PrefixedStorageKey(result)
}

// ParentPrefix returns the location reserved for this child trie in their parent trie if there
// is one.
func (ct ChildType) ParentPrefix() []byte {
	switch ct {
	case ChildTypeParentKeyID:
		return keys.DefaultChildStorageKeyPrefix
	default:
		panic("unreachable")
	}
}

// ChildTrieParentKeyID is a child trie of default type.
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

// StateVersion represents different possible state version.
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
