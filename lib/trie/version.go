// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	"errors"
	"fmt"
	"strings"
)

// Version is the state trie version which dictates how a
// Merkle root should be constructed. It is defined in
// https://spec.polkadot.network/#defn-state-version
type Version uint8

const (
	// V0 is the state trie version 0 where the values of the keys are
	// inserted into the trie directly.
	// TODO set to iota once CI passes
	V0 Version = 1
)

func (v Version) String() string {
	switch v {
	case V0:
		return "v0"
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
	default:
		return version, fmt.Errorf("%w: %q must be %s",
			ErrParseVersion, s, V0)
	}
}
