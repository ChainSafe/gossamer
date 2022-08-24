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

var ErrVersionNotValid = errors.New("version not valid")

// ParseVersion parses a state trie version string or uint32.
func ParseVersion[T string | uint32](x T) (version Version, err error) {
	var s string
	switch value := any(x).(type) {
	case string:
		s = value
	case uint32:
		s = fmt.Sprintf("V%d", value)
	default:
		panic(fmt.Sprintf("unsupported type %T", x))
	}

	switch {
	case strings.EqualFold(s, V0.String()):
		return V0, nil
	default:
		return version, fmt.Errorf("%w: %s", ErrVersionNotValid, s)
	}
}
