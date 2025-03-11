// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statemachine

import (
	"bytes"

	"github.com/ChainSafe/gossamer/internal/primitives/storage/keys"
)

type PrefixedStorageKey []byte

type ChildType uint8

const (
	ChildTypeParentKeyId = iota
)

func (c ChildType) newPrefixedKey(key []byte) PrefixedStorageKey {
	parentPrefix := c.parentPrefix()
	result := make([]byte, 0)
	result = append(result, parentPrefix...)
	result = append(result, key...)
	return PrefixedStorageKey(result)
}

func (c ChildType) parentPrefix() []byte {
	switch c {
	case ChildTypeParentKeyId:
		return keys.DefaultChildStorageKeyPrefix
	}
	return nil
}

type ChildInfo interface {
	TryUpdate(other ChildInfo) bool
	Keyspace() []byte
	StorageKey() []byte
	PrefixedStorageKey() PrefixedStorageKey
	ChildType() ChildType
}

type ChildInfoParentKeyId []byte

func (c ChildInfoParentKeyId) TryUpdate(other ChildInfo) bool {
	otherType, ok := other.(*ChildInfoParentKeyId)
	if !ok {
		return false
	}

	return bytes.Equal(c, otherType.Keyspace())
}

func (c ChildInfoParentKeyId) Keyspace() []byte {
	return c.StorageKey()
}

func (c ChildInfoParentKeyId) StorageKey() []byte {
	return c
}

func (c ChildInfoParentKeyId) PrefixedStorageKey() PrefixedStorageKey {
	return ChildType.newPrefixedKey(ChildTypeParentKeyId, c)
}

func (c ChildInfoParentKeyId) ChildType() ChildType {
	return ChildTypeParentKeyId
}

func NewDefaultChildInfo(storageKey []byte) ChildInfo {
	return ChildInfoParentKeyId(storageKey)
}
