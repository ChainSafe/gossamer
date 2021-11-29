// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package leaf

import "github.com/ChainSafe/gossamer/lib/trie/encode"

// Header creates the encoded header for the leaf.
func (l *Leaf) Header() (encoding []byte, err error) {
	var header byte = 1 << 6
	var encodedPublicKeyLength []byte

	if len(l.Key) >= 63 {
		header = header | 0x3f
		encodedPublicKeyLength, err = encode.ExtraPartialKeyLength(len(l.Key))
		if err != nil {
			return nil, err
		}
	} else {
		header = header | byte(len(l.Key))
	}

	encoding = append([]byte{header}, encodedPublicKeyLength...)

	return encoding, nil
}
