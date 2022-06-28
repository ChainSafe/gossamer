// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"io"
)

type prefixedWriter struct {
	prefix []byte
	writer io.Writer
}

func (w *prefixedWriter) Write(p []byte) (n int, err error) {
	toWrite := make([]byte, 0, len(w.prefix)+len(p))
	toWrite = append(toWrite, w.prefix...)
	toWrite = append(toWrite, p...)
	n, err = w.writer.Write(toWrite)

	// n has to match the length of p
	n -= len(w.prefix)
	if n < 0 {
		n = 0
	}

	return n, err
}
