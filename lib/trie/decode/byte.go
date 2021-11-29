// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package decode

import "io"

// ReadNextByte reads the next byte from the reader.
func ReadNextByte(reader io.Reader) (b byte, err error) {
	buffer := make([]byte, 1)
	_, err = reader.Read(buffer)
	if err != nil {
		return 0, err
	}
	return buffer[0], nil
}
