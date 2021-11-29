// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package branch

import (
	"bytes"
	"fmt"
	"hash"
	"io"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/trie/encode"
	"github.com/ChainSafe/gossamer/lib/trie/leaf"
	"github.com/ChainSafe/gossamer/lib/trie/node"
	"github.com/ChainSafe/gossamer/lib/trie/pools"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// ScaleEncodeHash hashes the node (blake2b sum on encoded value)
// and then SCALE encodes it. This is used to encode children
// nodes of branches.
func (b *Branch) ScaleEncodeHash() (encoding []byte, err error) {
	buffer := pools.DigestBuffers.Get().(*bytes.Buffer)
	buffer.Reset()
	defer pools.DigestBuffers.Put(buffer)

	err = b.hash(buffer)
	if err != nil {
		return nil, fmt.Errorf("cannot hash node: %w", err)
	}

	encoding, err = scale.Marshal(buffer.Bytes())
	if err != nil {
		return nil, fmt.Errorf("cannot scale encode hashed node: %w", err)
	}

	return encoding, nil
}

func (b *Branch) hash(digestBuffer io.Writer) (err error) {
	encodingBuffer := pools.EncodingBuffers.Get().(*bytes.Buffer)
	encodingBuffer.Reset()
	defer pools.EncodingBuffers.Put(encodingBuffer)

	err = b.Encode(encodingBuffer)
	if err != nil {
		return fmt.Errorf("cannot encode leaf: %w", err)
	}

	// if length of encoded leaf is less than 32 bytes, do not hash
	if encodingBuffer.Len() < 32 {
		_, err = digestBuffer.Write(encodingBuffer.Bytes())
		if err != nil {
			return fmt.Errorf("cannot write encoded branch to buffer: %w", err)
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

	_, err = digestBuffer.Write(hasher.Sum(nil))
	if err != nil {
		return fmt.Errorf("cannot write hash sum of branch to buffer: %w", err)
	}
	return nil
}

// Encode encodes a branch with the encoding specified at the top of this package
// to the buffer given.
func (b *Branch) Encode(buffer encode.Buffer) (err error) {
	if !b.Dirty && b.Encoding != nil {
		_, err = buffer.Write(b.Encoding)
		if err != nil {
			return fmt.Errorf("cannot write stored encoding to buffer: %w", err)
		}
		return nil
	}

	encodedHeader, err := b.Header()
	if err != nil {
		return fmt.Errorf("cannot encode header: %w", err)
	}

	_, err = buffer.Write(encodedHeader)
	if err != nil {
		return fmt.Errorf("cannot write encoded header to buffer: %w", err)
	}

	keyLE := encode.NibblesToKeyLE(b.Key)
	_, err = buffer.Write(keyLE)
	if err != nil {
		return fmt.Errorf("cannot write encoded key to buffer: %w", err)
	}

	childrenBitmap := common.Uint16ToBytes(b.ChildrenBitmap())
	_, err = buffer.Write(childrenBitmap)
	if err != nil {
		return fmt.Errorf("cannot write children bitmap to buffer: %w", err)
	}

	if b.Value != nil {
		bytes, err := scale.Marshal(b.Value)
		if err != nil {
			return fmt.Errorf("cannot scale encode value: %w", err)
		}

		_, err = buffer.Write(bytes)
		if err != nil {
			return fmt.Errorf("cannot write encoded value to buffer: %w", err)
		}
	}

	const parallel = false // TODO
	if parallel {
		err = encodeChildrenInParallel(b.Children, buffer)
	} else {
		err = encodeChildrenSequentially(b.Children, buffer)
	}
	if err != nil {
		return fmt.Errorf("cannot encode children of branch: %w", err)
	}

	return nil
}

func encodeChildrenInParallel(children [16]node.Node, buffer io.Writer) (err error) {
	type result struct {
		index  int
		buffer *bytes.Buffer
		err    error
	}

	resultsCh := make(chan result)

	for i, child := range children {
		go func(index int, child node.Node) {
			buffer := pools.EncodingBuffers.Get().(*bytes.Buffer)
			buffer.Reset()
			// buffer is put back in the pool after processing its
			// data in the select block below.

			err := encodeChild(child, buffer)

			resultsCh <- result{
				index:  index,
				buffer: buffer,
				err:    err,
			}
		}(i, child)
	}

	currentIndex := 0
	resultBuffers := make([]*bytes.Buffer, len(children))
	for range children {
		result := <-resultsCh
		if result.err != nil && err == nil { // only set the first error we get
			err = result.err
		}

		resultBuffers[result.index] = result.buffer

		// write as many completed buffers to the result buffer.
		for currentIndex < len(children) &&
			resultBuffers[currentIndex] != nil {
			bufferSlice := resultBuffers[currentIndex].Bytes()
			if len(bufferSlice) > 0 {
				// note buffer.Write copies the byte slice given as argument
				_, writeErr := buffer.Write(bufferSlice)
				if writeErr != nil && err == nil {
					err = fmt.Errorf(
						"cannot write encoding of child at index %d: %w",
						currentIndex, writeErr)
				}
			}

			pools.EncodingBuffers.Put(resultBuffers[currentIndex])
			resultBuffers[currentIndex] = nil

			currentIndex++
		}
	}

	for _, buffer := range resultBuffers {
		if buffer == nil { // already emptied and put back in pool
			continue
		}
		pools.EncodingBuffers.Put(buffer)
	}

	return err
}

func encodeChildrenSequentially(children [16]node.Node, buffer io.Writer) (err error) {
	for i, child := range children {
		err = encodeChild(child, buffer)
		if err != nil {
			return fmt.Errorf("cannot encode child at index %d: %w", i, err)
		}
	}
	return nil
}

func encodeChild(child node.Node, buffer io.Writer) (err error) {
	var isNil bool
	switch impl := child.(type) {
	case *Branch:
		isNil = impl == nil
	case *leaf.Leaf:
		isNil = impl == nil
	default:
		isNil = child == nil
	}
	if isNil {
		return nil
	}

	scaleEncodedChild, err := child.ScaleEncodeHash()
	if err != nil {
		return fmt.Errorf("failed to hash and scale encode child: %w", err)
	}

	_, err = buffer.Write(scaleEncodedChild)
	if err != nil {
		return fmt.Errorf("failed to write child to buffer: %w", err)
	}

	return nil
}
