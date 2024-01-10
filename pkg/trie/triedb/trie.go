// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import "github.com/ChainSafe/gossamer/pkg/trie/triedb/node"

type TrieValue interface {
	Type() string
}

type (
	InlineTrieValue struct {
		Bytes []byte
	}
	NodeTrieValue[H node.HashOut] struct {
		Hash H
	}
	NewNodeTrie[H node.HashOut] struct {
		Hash  *H
		Bytes []byte
	}
)

func (v InlineTrieValue) Type() string  { return "Inline" }
func (v NodeTrieValue[H]) Type() string { return "Node" }
func (v NewNodeTrie[H]) Type() string   { return "NewNode" }

func NewTrieValueFromBytes[H HashOut](value []byte, threshold *uint) TrieValue {
	if threshold != nil && uint(len(value)) >= *threshold {
		return NewNodeTrie[H]{nil, value}
	} else {
		return InlineTrieValue{Bytes: value}
	}
}

type Trie[Hash node.HashOut] interface {
	Root() Hash
	IsEmpty() bool
	Contains(key []byte) (bool, error)
	GetHash(key []byte) (*Hash, error)
	Get(key []byte) (*DBValue, error)
	//TODO:
	//get_with
	//lookup_first_descendant
}

type MutableTrie[Hash node.HashOut] interface {
	insert(key []byte, value []byte) (*TrieValue, error)
	remove(key []byte) (*TrieValue, error)
}
