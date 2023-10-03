// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ChainSafe/gossamer/lib/common"
)

const (
	NoMaxValueSize = -1
	V1MaxValueSize = 32
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

// DefaultStateVersion sets the state version we should use as default
// See https://github.com/paritytech/substrate/blob/5e76587825b9a9d52d8cb02ba38828adf606157b/primitives/storage/src/lib.rs#L435-L439
const DefaultStateVersion = V1

type Entry struct{ Key, Value []byte }
type Entries []Entry

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

func (v Version) MaxInlineValueSize() int {
	switch v {
	case V0:
		return NoMaxValueSize
	case V1:
		return V1MaxValueSize
	default:
		panic(fmt.Sprintf("unknown version %d", v))
	}
}

func (v Version) ShouldHashValue(value []byte) bool {
	return v.MaxInlineValueSize() != NoMaxValueSize && len(value) > v.MaxInlineValueSize()
}

func (v Version) Root(entries Entries) (common.Hash, error) {
	t := NewEmptyTrie()

	for _, kv := range entries {
		err := t.Put(kv.Key, kv.Value, v)
		if err != nil {
			return common.EmptyHash, err
		}
	}

	return t.Hash()
}

var ErrParseVersion = errors.New("parsing version failed")

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
