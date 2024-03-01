// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import "io"

// Buffer is an interface with some methods of *bytes.Buffer.
type Buffer interface {
	io.Writer
	Len() int
	Bytes() []byte
}
