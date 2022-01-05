// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"bytes"

	"github.com/ChainSafe/gossamer/internal/lib/common"
	"github.com/ChainSafe/gossamer/internal/trie/pools"
)

// SetEncodingAndHash sets the encoding and hash slices
// given to the branch. Note it does not copy them, so beware.
func (b *Branch) SetEncodingAndHash(enc, hash []byte) {
	b.encoding = enc
	b.hashDigest = hash
}

// GetHash returns the hash of the branch.
// Note it does not copy it, so modifying
// the returned hash will modify the hash
// of the branch.
func (b *Branch) GetHash() []byte {
	return b.hashDigest
}

// EncodeAndHash returns the encoding of the branch and
// the blake2b hash digest of the encoding of the branch.
// If the encoding is less than 32 bytes, the hash returned
// is the encoding and not the hash of the encoding.
func (b *Branch) EncodeAndHash() (encoding, hash []byte, err error) {
	if !b.dirty && b.encoding != nil && b.hashDigest != nil {
		return b.encoding, b.hashDigest, nil
	}

	buffer := pools.EncodingBuffers.Get().(*bytes.Buffer)
	buffer.Reset()
	defer pools.EncodingBuffers.Put(buffer)

	err = b.Encode(buffer)
	if err != nil {
		return nil, nil, err
	}

	bufferBytes := buffer.Bytes()

	b.encoding = make([]byte, len(bufferBytes))
	copy(b.encoding, bufferBytes)
	encoding = b.encoding // no need to copy

	if buffer.Len() < 32 {
		b.hashDigest = make([]byte, len(bufferBytes))
		copy(b.hashDigest, bufferBytes)
		hash = b.hashDigest // no need to copy
		return encoding, hash, nil
	}

	// Note: using the sync.Pool's buffer is useful here.
	hashArray, err := common.Blake2bHash(buffer.Bytes())
	if err != nil {
		return nil, nil, err
	}
	b.hashDigest = hashArray[:]
	hash = b.hashDigest // no need to copy

	return encoding, hash, nil
}

// SetEncodingAndHash sets the encoding and hash slices
// given to the branch. Note it does not copy them, so beware.
func (l *Leaf) SetEncodingAndHash(enc, hash []byte) {
	l.encodingMu.Lock()
	l.encoding = enc
	l.encodingMu.Unlock()
	l.hashDigest = hash
}

// GetHash returns the hash of the leaf.
// Note it does not copy it, so modifying
// the returned hash will modify the hash
// of the branch.
func (l *Leaf) GetHash() []byte {
	return l.hashDigest
}

// EncodeAndHash returns the encoding of the leaf and
// the blake2b hash digest of the encoding of the leaf.
// If the encoding is less than 32 bytes, the hash returned
// is the encoding and not the hash of the encoding.
func (l *Leaf) EncodeAndHash() (encoding, hash []byte, err error) {
	l.encodingMu.RLock()
	if !l.IsDirty() && l.encoding != nil && l.hashDigest != nil {
		l.encodingMu.RUnlock()
		return l.encoding, l.hashDigest, nil
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
	l.encoding = make([]byte, len(bufferBytes))
	copy(l.encoding, bufferBytes)
	l.encodingMu.Unlock()
	encoding = l.encoding // no need to copy

	if len(bufferBytes) < 32 {
		l.hashDigest = make([]byte, len(bufferBytes))
		copy(l.hashDigest, bufferBytes)
		hash = l.hashDigest // no need to copy
		return encoding, hash, nil
	}

	// Note: using the sync.Pool's buffer is useful here.
	hashArray, err := common.Blake2bHash(buffer.Bytes())
	if err != nil {
		return nil, nil, err
	}

	l.hashDigest = hashArray[:]
	hash = l.hashDigest // no need to copy

	return encoding, hash, nil
}
