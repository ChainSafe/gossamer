// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trieroot

import (
	hashdb "github.com/ChainSafe/gossamer/internal/hash-db"
	"github.com/ChainSafe/gossamer/internal/primitives/kv"
	"github.com/tidwall/btree"
)

type KeyValue = kv.KeyValue

// Different possible value to use for node encoding.
type Value interface {
	isValue()
}
type (
	// Contains a full value.
	InlineValue []byte
	// Contains hash of a value.
	NodeValue []byte
)

func (InlineValue) isValue() {}
func (NodeValue) isValue()   {}

func NewValue[Hasher hashdb.Hasher[H], H hashdb.Hash](value []byte, threshold *uint32) Value {
	if threshold != nil && len(value) >= int(*threshold) {
		h := (*new(Hasher)).Hash(value)
		return NodeValue(h.Bytes())
	}
	return InlineValue(value)
}

// Byte-stream oriented interface for constructing closed-form tries.
type TrieStream interface {
	// Construct a new TrieStream
	New() TrieStream
	// Append an Empty node
	AppendEmptyData()
	// Start a new Branch node, possibly with a value; takes a list indicating
	// which slots in the Branch node has further child nodes.
	BeginBranch(
		maybeKey []byte,
		maybeValue Value,
		hasChildren []bool,
	)
	// Append an empty child node. Optional.
	AppendEmptyChild()
	// Wrap up a Branch node portion of a TrieStream and append the value
	// stored on the Branch (if any).
	EndBranch(value Value)
	// Append a Leaf node
	AppendLeaf(key []byte, value Value)
	// Append an Extension node
	AppendExtension(key []byte)
	// Append a Branch of Extension substream
	AppendSubstream(other TrieStream)
	// Return the finished TrieStream as a vector of bytes.
	Out() []byte
}

func sharedPrefixLength(first []byte, second []byte) uint {
	length := min(first, second)
	var position int = -1
	for i := 0; i < length; i++ {
		var (
			a byte
			b byte
		)
		if i < len(first) {
			a = first[i]
		}
		if i < len(second) {
			b = second[i]
		}
		if a != b {
			position = i
			break
		}
	}
	if position == -1 {
		position = len(first)
		if len(second) < position {
			position = len(second)
		}
	}
	return uint(position)
}

func TrieRoot[Hasher hashdb.Hasher[H], H hashdb.Hash](
	input []KeyValue,
	threshold *uint32,
	stream TrieStream,
) H {

	// first put elements into btree to sort them and to remove duplicates
	inputMap := btree.Map[string, []byte]{}
	for _, kv := range input {
		inputMap.Set(string(kv.Key), kv.Value)
	}

	// convert to nibbles
	nibblesLen := 0
	for _, kv := range input {
		nibblesLen += len(kv.Key) * 2
	}
	nibbles := make([]byte, 0, nibblesLen)
	lens := make([]uint, 0, len(input)+1)
	lens = append(lens, 0)
	for _, kv := range input {
		for _, b := range kv.Key {
			nibbles = append(nibbles, b>>4, b&0x0F)
		}
		lens = append(lens, uint(len(nibbles)))
	}

	// then move them to a vector
	trieInput := make([]KeyValue, len(input))
	for i, kv := range input {
		in := KeyValue{
			Key:   nibbles[lens[i]:lens[i+1]],
			Value: kv.Value,
		}
		trieInput[i] = in
	}

	buildTrie[Hasher, H](trieInput, 0, stream, threshold)
	return (*new(Hasher)).Hash(stream.Out())
}

func buildTrie[Hasher hashdb.Hasher[H], H hashdb.Hash](
	input []KeyValue,
	cursor uint,
	stream TrieStream,
	threshold *uint32,
) {
	switch len(input) {
	case 0:
		// No input, just append empty data.
		stream.AppendEmptyData()

	case 1:
		// Leaf node; append the remainder of the key and the value. Done.
		value := NewValue[Hasher](input[0].Value, threshold)
		stream.AppendLeaf(input[0].Key[cursor:], value)

		// 	_ => {
	default:
		// We have multiple items in the input. Figure out if we should add an extension node or a branch node.
		key := input[0].Key
		value := input[0].Value
		// Count the number of nibbles in the other elements that are shared with the first key.
		sharedNibbleCount := uint(len(key))
		for _, kv := range input[1:] {
			sharedPrefixLength := sharedPrefixLength(key, kv.Key)
			if sharedPrefixLength < sharedNibbleCount {
				sharedNibbleCount = sharedPrefixLength
			}
		}
		// Add an extension node if the number of shared nibbles is greater than what we saw on the last
		// call (cursor): append the new part of the path then recursively append the remainder of all items
		// who had this partial key.
		var oBranchSlice []byte

		if sharedNibbleCount > cursor {
			oBranchSlice = key[cursor:sharedNibbleCount]
			cursor = sharedNibbleCount
		} else {
			oBranchSlice = key[0:0]
		}

		// We'll be adding a branch node because the path is as long as it gets.
		// First we need to figure out what entries this branch node will have...

		// We have a value for exactly this key. Branch node will have a value attached to it.
		var val []byte
		if cursor == uint(len(key)) {
			val = value
		} else {
			val = nil
		}

		// We need to know how many key nibbles each of the children account for.
		sharedNibblesCounts := make([]uint, 16)
		{
			// If the Branch node has a value then the first of the input keys is exactly the key for that value and
			// we don't care about it when finding shared nibbles for our child nodes. (We know it's the first of the
			// input keys, because the input is sorted)
			var begin uint
			if val == nil {
				begin = 0
			} else {
				begin = 1
			}
			for i := uint(0); i < 16; i++ {
				var sharedNibbleCount uint
				for _, kv := range input[begin:] {
					if kv.Key[cursor] == byte(i) {
						sharedNibbleCount++
					} else {
						break
					}
				}
				sharedNibblesCounts[i] = sharedNibbleCount
				begin += sharedNibbleCount
			}
		}

		// Put out the node header:
		var v Value
		if val != nil {
			v = NewValue[Hasher](val, threshold)
		}
		hasChildren := make([]bool, 16)
		for i, count := range sharedNibblesCounts {
			if count > 0 {
				hasChildren[i] = true
			}
		}
		stream.BeginBranch(oBranchSlice, v, hasChildren)

		// Fill in each slot in the branch node. We don't need to bother with empty slots
		// since they were registered in the header.
		var begin uint = 0
		if val != nil {
			begin = 1
		}
		for _, count := range sharedNibblesCounts {
			if count > 0 {
				buildTrieTrampoline[Hasher, H](
					input[begin:begin+count],
					cursor+1,
					stream,
					threshold,
				)
				begin += count
			} else {
				stream.AppendEmptyChild()
			}
		}

		stream.EndBranch(v)
	}
}

func buildTrieTrampoline[Hasher hashdb.Hasher[H], H hashdb.Hash](
	input []KeyValue,
	cursor uint,
	stream TrieStream,
	threshold *uint32,
) {
	substream := stream.New()
	buildTrie[Hasher, H](input, cursor, substream, threshold)
	stream.AppendSubstream(substream)
}
