// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"io"
)

const (
	leafHeaderByte            byte = 0x1
	branchHeaderByte          byte = 2
	branchWithValueHeaderByte byte = 3
	keyLenOffset                   = 0x3f
	nodeHeaderShift                = 6
)

func encodeHeader(node *Node, writer io.Writer) (err error) {
	switch node.Type {
	case Leaf:
		return encodeLeafHeader(node, writer)
	case Branch:
		return encodeBranchHeader(node, writer)
	default:
		panic("header encoding not implemented")
	}
}

// encodeBranchHeader writes the encoded header for the branch.
func encodeBranchHeader(branch *Node, writer io.Writer) (err error) {
	var header byte
	if branch.Value == nil {
		header = branchHeaderByte << nodeHeaderShift
	} else {
		header = branchWithValueHeaderByte << nodeHeaderShift
	}

	if len(branch.Key) >= keyLenOffset {
		header = header | keyLenOffset
		_, err = writer.Write([]byte{header})
		if err != nil {
			return err
		}

		err = encodeKeyLength(len(branch.Key), writer)
		if err != nil {
			return err
		}
	} else {
		header = header | byte(len(branch.Key))
		_, err = writer.Write([]byte{header})
		if err != nil {
			return err
		}
	}

	return nil
}

// encodeLeafHeader writes the encoded header for the leaf.
func encodeLeafHeader(leaf *Node, writer io.Writer) (err error) {
	header := leafHeaderByte << nodeHeaderShift

	if len(leaf.Key) < 63 {
		header |= byte(len(leaf.Key))
		_, err = writer.Write([]byte{header})
		return err
	}

	header |= keyLenOffset
	_, err = writer.Write([]byte{header})
	if err != nil {
		return err
	}

	err = encodeKeyLength(len(leaf.Key), writer)
	if err != nil {
		return err
	}

	return nil
}
