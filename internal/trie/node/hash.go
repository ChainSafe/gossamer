// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"bytes"

	"github.com/ChainSafe/gossamer/internal/trie/pools"
	"github.com/ChainSafe/gossamer/lib/common"
)

// SetEncodingAndHash sets the encoding and hash slices
// given to the branch. Note it does not copy them, so beware.
func (b *Branch) SetEncodingAndHash(enc, hash []byte) {
	b.Encoding = enc
	b.HashDigest = hash
}

// GetHash returns the hash of the branch.
// Note it does not copy it, so modifying
// the returned hash will modify the hash
// of the branch.
func (b *Branch) GetHash() []byte {
	return b.HashDigest
}

// EncodeAndHash returns the encoding of the branch and
// the blake2b hash digest of the encoding of the branch.
// If the encoding is less than 32 bytes, the hash returned
// is the encoding and not the hash of the encoding.
func (b *Branch) EncodeAndHash() (encoding, hash []byte, err error) {
	if !b.Dirty && b.Encoding != nil && b.HashDigest != nil {
		return b.Encoding, b.HashDigest, nil
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
		b.HashDigest = make([]byte, len(bufferBytes))
		copy(b.HashDigest, bufferBytes)
		hash = b.HashDigest // no need to copy
		return encoding, hash, nil
	}

	// Note: using the sync.Pool's buffer is useful here.
	hashArray, err := common.Blake2bHash(buffer.Bytes())
	if err != nil {
		return nil, nil, err
	}
	b.HashDigest = hashArray[:]
	hash = b.HashDigest // no need to copy

	return encoding, hash, nil
}

// SetEncodingAndHash sets the encoding and hash slices
// given to the branch. Note it does not copy them, so beware.
func (l *Leaf) SetEncodingAndHash(enc, hash []byte) {
	l.encodingMu.Lock()
	l.Encoding = enc
	l.encodingMu.Unlock()
	l.HashDigest = hash
}

// GetHash returns the hash of the leaf.
// Note it does not copy it, so modifying
// the returned hash will modify the hash
// of the branch.
func (l *Leaf) GetHash() []byte {
	return l.HashDigest
}

// EncodeAndHash returns the encoding of the leaf and
// the blake2b hash digest of the encoding of the leaf.
// If the encoding is less than 32 bytes, the hash returned
// is the encoding and not the hash of the encoding.
func (l *Leaf) EncodeAndHash() (encoding, hash []byte, err error) {
	l.encodingMu.RLock()
	if !l.IsDirty() && l.Encoding != nil && l.HashDigest != nil {
		l.encodingMu.RUnlock()
		return l.Encoding, l.HashDigest, nil
	}
	l.encodingMu.RUnlock()

	buffer := pools.EncodingBuffers.Get().(*bytes.Buffer)
	buffer.Reset()
	defer pools.EncodingBuffers.Put(buffer)

	err = l.Encode(buffer)
	if err != nil {
		return nil, nil, err
	}

	bufferBytes := buffer.Bytes()

	l.encodingMu.Lock()
	// TODO remove this copying since it defeats the purpose of `buffer`
	// and the sync.Pool.
	l.Encoding = make([]byte, len(bufferBytes))
	copy(l.Encoding, bufferBytes)
	l.encodingMu.Unlock()
	encoding = l.Encoding // no need to copy

	if len(bufferBytes) < 32 {
		l.HashDigest = make([]byte, len(bufferBytes))
		copy(l.HashDigest, bufferBytes)
		hash = l.HashDigest // no need to copy
		return encoding, hash, nil
	}

	// Note: using the sync.Pool's buffer is useful here.
	hashArray, err := common.Blake2bHash(buffer.Bytes())
	if err != nil {
		return nil, nil, err
	}

	l.HashDigest = hashArray[:]
	hash = l.HashDigest // no need to copy

	return encoding, hash, nil
}
