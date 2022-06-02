// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"io"
)

const (
	leafHeader            byte = 1 // 01
	branchHeader          byte = 2 // 10
	branchWithValueHeader byte = 3 // 11
)

const (
	keyLenOffset    = 0x3f
	nodeHeaderShift = 6
)

// encodeHeader writes the encoded header for the node.
func encodeHeader(node *Node, writer io.Writer) (err error) {
	var header byte
	if node.Type() == Leaf {
		header = leafHeader
	} else if node.Value == nil {
		header = branchHeader
	} else {
		header = branchWithValueHeader
	}
	header <<= nodeHeaderShift

	if len(node.Key) < keyLenOffset {
		header |= byte(len(node.Key))
		_, err = writer.Write([]byte{header})
		return err
	}

	header = header | keyLenOffset
	_, err = writer.Write([]byte{header})
	if err != nil {
		return err
	}

	err = encodeKeyLength(len(node.Key), writer)
	return err
}
