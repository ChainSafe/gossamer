// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package pools

import (
	"bytes"
	"sync"

	"golang.org/x/crypto/blake2b"
)

// SingleByteBuffers is a sync pool of buffers of capacity 1.
var SingleByteBuffers = &sync.Pool{
	New: func() interface{} {
		const bufferLength = 1
		b := make([]byte, bufferLength)
		return bytes.NewBuffer(b)
	},
}

// DigestBuffers is a sync pool of buffers of capacity 32.
var DigestBuffers = &sync.Pool{
	New: func() interface{} {
		const bufferCapacity = 32
		b := make([]byte, 0, bufferCapacity)
		return bytes.NewBuffer(b)
	},
}

// EncodingBuffers is a sync pool of buffers of capacity 1.9MB.
var EncodingBuffers = &sync.Pool{
	New: func() interface{} {
		const initialBufferCapacity = 1900000 // 1.9MB, from checking capacities at runtime
		b := make([]byte, 0, initialBufferCapacity)
		return bytes.NewBuffer(b)
	},
}

// Hashers is a sync pool of blake2b 256 hashers.
var Hashers = &sync.Pool{
	New: func() interface{} {
		hasher, err := blake2b.New256(nil)
		if err != nil {
			// Conversation on why we panic here:
			// https://github.com/ChainSafe/gossamer/pull/2009#discussion_r753430764
			panic("cannot create Blake2b-256 hasher: " + err.Error())
		}
		return hasher
	},
}
