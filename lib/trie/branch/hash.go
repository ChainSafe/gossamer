// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package branch

import (
	"bytes"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/trie/pools"
)

// SetEncodingAndHash sets the encoding and hash slices
// given to the branch. Note it does not copy them, so beware.
func (b *Branch) SetEncodingAndHash(enc, hash []byte) {
	b.Encoding = enc
	b.Hash = hash
}

// GetHash returns the hash of the branch.
// Note it does not copy it, so modifying
// the returned hash will modify the hash
// of the branch.
func (b *Branch) GetHash() []byte {
	return b.Hash
}

// EncodeAndHash returns the encoding of the branch and
// the blake2b hash digest of the encoding of the branch.
// If the encoding is less than 32 bytes, the hash returned
// is the encoding and not the hash of the encoding.
func (b *Branch) EncodeAndHash() (encoding, hash []byte, err error) {
	if !b.Dirty && b.Encoding != nil && b.Hash != nil {
		return b.Encoding, b.Hash, nil
	}

	buffer := pools.EncodingBuffers.Get().(*bytes.Buffer)
	buffer.Reset()
	defer pools.EncodingBuffers.Put(buffer)

	err = b.Encode(buffer)
	if err != nil {
		return nil, nil, err
	}

	bufferBytes := buffer.Bytes()

	b.Encoding = make([]byte, len(bufferBytes))
	copy(b.Encoding, bufferBytes)
	encoding = b.Encoding // no need to copy

	if buffer.Len() < 32 {
		b.Hash = make([]byte, len(bufferBytes))
		copy(b.Hash, bufferBytes)
		hash = b.Hash // no need to copy
		return encoding, hash, nil
	}

	// Note: using the sync.Pool's buffer is useful here.
	hashArray, err := common.Blake2bHash(buffer.Bytes())
	if err != nil {
		return nil, nil, err
	}
	b.Hash = hashArray[:]
	hash = b.Hash // no need to copy

	return encoding, hash, nil
}
