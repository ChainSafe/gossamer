// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import "io"

// Option is an option to use with the `New` constructor.
type Option func(node *Node)

// SetIndex sets the index for the node.
func SetIndex(index int) Option {
	return func(node *Node) {
		node.index = intPtr(index)
	}
}

// SetWriter sets the writer for the node.
func SetWriter(writer io.Writer) Option {
	return func(node *Node) {
		node.writer = writer
	}
}
