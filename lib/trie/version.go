// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ChainSafe/gossamer/lib/common"
)

const V1MaxValueSize = common.HashLength

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

func (v Version) ShouldHashValue(value []byte) bool {
	switch v {
	case V0:
		return false
	case V1:
		return len(value) >= V1MaxValueSize
	default:
		panic(fmt.Sprintf("unknown version %d", v))
	}
}

var ErrParseVersion = errors.New("parsing version failed")

// ParseVersion parses a state trie version string.
func ParseVersion(s string) (version Version, err error) {
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

func EnforceValidVersion(version uint8) {
	v := Version(version)

	switch v {
	case V0, V1:
		return
	default:
		panic("Invalid state version")
	}
}
