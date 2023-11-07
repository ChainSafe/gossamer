// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/ChainSafe/gossamer/lib/common"
)

const (
	// NoMaxInlineValueSize is the numeric representation used to indicate that there is no max value size.
	NoMaxInlineValueSize = math.MaxInt
	// V1MaxInlineValueSize is the maximum size of a value to be hashed in state trie version 1.
	V1MaxInlineValueSize = 32
)

// Version is the state trie version which dictates how a
// Merkle root should be constructed. It is defined in
// https://spec.polkadot.network/#defn-state-version
type Version uint8

const (
	// V0 is the state trie version 0 where the values of the keys are
	// inserted into the trie directly.
	// TODO set to iota once CI passes
	V0 Version = iota
	V1
)

// ErrParseVersion is returned when parsing a state trie version fails.
var ErrParseVersion = errors.New("parsing version failed")

// DefaultStateVersion sets the state version we should use as default
// See https://github.com/paritytech/substrate/blob/5e76587825b9a9d52d8cb02ba38828adf606157b/primitives/storage/src/lib.rs#L435-L439
const DefaultStateVersion = V1

// Entry is a key-value pair used to build a trie
type Entry struct{ Key, Value []byte }

// Entries is a list of entry used to build a trie
type Entries []Entry

// String returns a string representation of trie version
func (v Version) String() string {
	switch v {
	case V0:
		return "v0"
	case V1:
		return "v1"
	default:
		panic(fmt.Sprintf("unknown version %d", v))
	}
}

func (v Version) maxInlineValue() int {
	switch v {
	case V0:
		return NoMaxInlineValueSize
	case V1:
		return V1MaxInlineValueSize
	default:
		panic(fmt.Sprintf("unknown version %d", v))
	}
}

// Root returns the root hash of the trie built using the given entries
func (v Version) Root(entries Entries) (common.Hash, error) {
	t := NewEmptyTrie()

	for _, kv := range entries {
		err := t.Put(kv.Key, kv.Value)
		if err != nil {
			return common.EmptyHash, err
		}
	}

	return t.Hash()
}

// ParseVersion parses a state trie version string.
func ParseVersion[T string | uint32](v T) (version Version, err error) {
	var s string
	switch value := any(v).(type) {
	case string:
		s = value
	case uint32:
		s = fmt.Sprintf("V%d", value)
	}

	switch {
	case strings.EqualFold(s, V0.String()):
		return V0, nil
	case strings.EqualFold(s, V1.String()):
		return V1, nil
	default:
		return version, fmt.Errorf("%w: %q must be one of [%s, %s]",
			ErrParseVersion, s, V0, V1)
	}
}
