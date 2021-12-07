// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package leaf

import (
	"io"

	"github.com/ChainSafe/gossamer/lib/trie/encode"
)

// encodeHeader creates the encoded header for the leaf.
func (l *Leaf) encodeHeader(writer io.Writer) (err error) {
	var header byte = 1 << 6

	if len(l.Key) < 63 {
		header = header | byte(len(l.Key))
		_, err = writer.Write([]byte{header})
		return err
	}

	header = header | 0x3f
	_, err = writer.Write([]byte{header})
	if err != nil {
		return err
	}

	err = encode.KeyLength(len(l.Key), writer)
	if err != nil {
		return err
	}

	return nil
}
