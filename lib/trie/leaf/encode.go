// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package leaf

import (
	"bytes"
	"fmt"
	"hash"
	"io"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/trie/encode"
	"github.com/ChainSafe/gossamer/lib/trie/pools"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// SetEncodingAndHash sets the encoding and hash slices
// given to the branch. Note it does not copy them, so beware.
func (l *Leaf) SetEncodingAndHash(enc, hash []byte) {
	l.encodingMu.Lock()
	l.Encoding = enc
	l.encodingMu.Unlock()
	l.Hash = hash
}

// GetHash returns the hash of the leaf.
// Note it does not copy it, so modifying
// the returned hash will modify the hash
// of the branch.
func (l *Leaf) GetHash() []byte {
	return l.Hash
}

// EncodeAndHash returns the encoding of the leaf and
// the blake2b hash digest of the encoding of the leaf.
// If the encoding is less than 32 bytes, the hash returned
// is the encoding and not the hash of the encoding.
func (l *Leaf) EncodeAndHash() (encoding, hash []byte, err error) {
	l.encodingMu.RLock()
	if !l.IsDirty() && l.Encoding != nil && l.Hash != nil {
		l.encodingMu.RUnlock()
		return l.Encoding, l.Hash, nil
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
		l.Hash = make([]byte, len(bufferBytes))
		copy(l.Hash, bufferBytes)
		hash = l.Hash // no need to copy
		return encoding, hash, nil
	}

	// Note: using the sync.Pool's buffer is useful here.
	hashArray, err := common.Blake2bHash(buffer.Bytes())
	if err != nil {
		return nil, nil, err
	}

	l.Hash = hashArray[:]
	hash = l.Hash // no need to copy

	return encoding, hash, nil
}

// Encode encodes a leaf to the buffer given.
// The encoding has the following format:
// NodeHeader | Extra partial key length | Partial Key | Value
func (l *Leaf) Encode(buffer encode.Buffer) (err error) {
	l.encodingMu.RLock()
	if !l.Dirty && l.Encoding != nil {
		_, err = buffer.Write(l.Encoding)
		l.encodingMu.RUnlock()
		if err != nil {
			return fmt.Errorf("cannot write stored encoding to buffer: %w", err)
		}
		return nil
	}
	l.encodingMu.RUnlock()

	encodedHeader, err := l.Header()
	if err != nil {
		return fmt.Errorf("cannot encode header: %w", err)
	}

	_, err = buffer.Write(encodedHeader)
	if err != nil {
		return fmt.Errorf("cannot write encoded header to buffer: %w", err)
	}

	keyLE := encode.NibblesToKeyLE(l.Key)
	_, err = buffer.Write(keyLE)
	if err != nil {
		return fmt.Errorf("cannot write LE key to buffer: %w", err)
	}

	encodedValue, err := scale.Marshal(l.Value) // TODO scale encoder to write to buffer
	if err != nil {
		return fmt.Errorf("cannot scale marshal value: %w", err)
	}

	_, err = buffer.Write(encodedValue)
	if err != nil {
		return fmt.Errorf("cannot write scale encoded value to buffer: %w", err)
	}

	// TODO remove this copying since it defeats the purpose of `buffer`
	// and the sync.Pool.
	l.encodingMu.Lock()
	defer l.encodingMu.Unlock()
	l.Encoding = make([]byte, buffer.Len())
	copy(l.Encoding, buffer.Bytes())
	return nil
}

// ScaleEncodeHash hashes the node (blake2b sum on encoded value)
// and then SCALE encodes it. This is used to encode children
// nodes of branches.
func (l *Leaf) ScaleEncodeHash() (b []byte, err error) {
	buffer := pools.DigestBuffers.Get().(*bytes.Buffer)
	buffer.Reset()
	defer pools.DigestBuffers.Put(buffer)

	err = l.hash(buffer)
	if err != nil {
		return nil, fmt.Errorf("cannot hash node: %w", err)
	}

	scEncChild, err := scale.Marshal(buffer.Bytes())
	if err != nil {
		return nil, fmt.Errorf("cannot scale encode hashed node: %w", err)
	}
	return scEncChild, nil
}

func (l *Leaf) hash(writer io.Writer) (err error) {
	encodingBuffer := pools.EncodingBuffers.Get().(*bytes.Buffer)
	encodingBuffer.Reset()
	defer pools.EncodingBuffers.Put(encodingBuffer)

	err = l.Encode(encodingBuffer)
	if err != nil {
		return fmt.Errorf("cannot encode leaf: %w", err)
	}

	// if length of encoded leaf is less than 32 bytes, do not hash
	if encodingBuffer.Len() < 32 {
		_, err = writer.Write(encodingBuffer.Bytes())
		if err != nil {
			return fmt.Errorf("cannot write encoded leaf to buffer: %w", err)
		}
		return nil
	}

	// otherwise, hash encoded node
	hasher := pools.Hashers.Get().(hash.Hash)
	hasher.Reset()
	defer pools.Hashers.Put(hasher)

	// Note: using the sync.Pool's buffer is useful here.
	_, err = hasher.Write(encodingBuffer.Bytes())
	if err != nil {
		return fmt.Errorf("cannot hash encoded node: %w", err)
	}

	_, err = writer.Write(hasher.Sum(nil))
	if err != nil {
		return fmt.Errorf("cannot write hash sum of leaf to buffer: %w", err)
	}
	return nil
}
