// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"bytes"
	"fmt"
	"hash"
	"io"

	"github.com/ChainSafe/gossamer/internal/trie/codec"
	"github.com/ChainSafe/gossamer/internal/trie/pools"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// Encode encodes a leaf to the buffer given.
// The encoding has the following format:
// NodeHeader | Extra partial key length | Partial Key | Value
func (l *Leaf) Encode(buffer Buffer) (err error) {
	l.encodingMu.RLock()
	if !l.Dirty && l.encoding != nil {
		_, err = buffer.Write(l.encoding)
		l.encodingMu.RUnlock()
		if err != nil {
			return fmt.Errorf("cannot write stored encoding to buffer: %w", err)
		}
		return nil
	}
	l.encodingMu.RUnlock()

	err = l.encodeHeader(buffer)
	if err != nil {
		return fmt.Errorf("cannot encode header: %w", err)
	}

	keyLE := codec.NibblesToKeyLE(l.Key)
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
	l.encoding = make([]byte, buffer.Len())
	copy(l.encoding, buffer.Bytes())
	return nil
}

// ScaleEncodeHash hashes the node (blake2b sum on encoded value)
// and then SCALE encodes it. This is used to encode children
// nodes of branches.
func (l *Leaf) ScaleEncodeHash() (encoding []byte, err error) {
	buffer := pools.DigestBuffers.Get().(*bytes.Buffer)
	buffer.Reset()
	defer pools.DigestBuffers.Put(buffer)

	err = l.hash(buffer)
	if err != nil {
		return nil, fmt.Errorf("cannot hash leaf: %w", err)
	}

	scEncChild, err := scale.Marshal(buffer.Bytes())
	if err != nil {
		return nil, fmt.Errorf("cannot scale encode hashed leaf: %w", err)
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
