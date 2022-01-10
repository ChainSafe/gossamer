// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"io"
)

const (
	keyLenOffset    = 0x3f
	nodeHeaderShift = 6
)

// encodeHeader creates the encoded header for the branch.
func (b *Branch) encodeHeader(writer io.Writer) (err error) {
	var header byte
	if b.Value == nil {
		header = byte(BranchType) << nodeHeaderShift
	} else {
		header = byte(BranchWithValueType) << nodeHeaderShift
	}

	if len(b.Key) >= keyLenOffset {
		header = header | keyLenOffset
		_, err = writer.Write([]byte{header})
		if err != nil {
			return err
		}

		err = encodeKeyLength(len(b.Key), writer)
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

// encodeHeader creates the encoded header for the leaf.
func (l *Leaf) encodeHeader(writer io.Writer) (err error) {
	header := byte(LeafType) << nodeHeaderShift

	if len(l.Key) < 63 {
		header |= byte(len(l.Key))
		_, err = writer.Write([]byte{header})
		return err
	}

	header |= keyLenOffset
	_, err = writer.Write([]byte{header})
	if err != nil {
		return err
	}

	err = encodeKeyLength(len(l.Key), writer)
	if err != nil {
		return err
	}

	return nil
}
