// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package encode

import "io"

// Buffer is an interface with some methods of *bytes.Buffer.
type Buffer interface {
	Writer
	Len() int
	Bytes() []byte
}

// Writer is the io.Writer interface
type Writer io.Writer
