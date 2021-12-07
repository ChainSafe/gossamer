// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package branch

import (
	"io"

	"github.com/ChainSafe/gossamer/lib/trie/encode"
)

// encodeHeader creates the encoded header for the branch.
func (b *Branch) encodeHeader(writer io.Writer) (err error) {
	var header byte
	if b.Value == nil {
		header = 2 << 6
	} else {
		header = 3 << 6
	}

	if len(b.Key) >= 63 {
		header = header | 0x3f
		_, err = writer.Write([]byte{header})
		if err != nil {
			return err
		}

		err = encode.KeyLength(len(b.Key), writer)
		if err != nil {
			return err
		}
	} else {
		header = header | byte(len(b.Key))
		_, err = writer.Write([]byte{header})
		if err != nil {
			return err
		}
	}

	return nil
}
