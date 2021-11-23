// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	"bytes"
	"errors"
	"fmt"
	"hash"
	"io"
	"sync"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"golang.org/x/crypto/blake2b"
)

var encodingBufferPool = &sync.Pool{
	New: func() interface{} {
		const initialBufferCapacity = 1900000 // 1.9MB, from checking capacities at runtime
		b := make([]byte, 0, initialBufferCapacity)
		return bytes.NewBuffer(b)
	},
}

var digestBufferPool = &sync.Pool{
	New: func() interface{} {
		const bufferCapacity = 32
		b := make([]byte, 0, bufferCapacity)
		return bytes.NewBuffer(b)
	},
}

var hasherPool = &sync.Pool{
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

func hashNode(n Node, digestBuffer io.Writer) (err error) {
	encodingBuffer := encodingBufferPool.Get().(*bytes.Buffer)
	encodingBuffer.Reset()
	defer encodingBufferPool.Put(encodingBuffer)

	const parallel = false

	err = encodeNode(n, encodingBuffer, parallel)
	if err != nil {
		return fmt.Errorf("cannot encode node: %w", err)
	}

	// if length of encoded leaf is less than 32 bytes, do not hash
	if encodingBuffer.Len() < 32 {
		_, err = digestBuffer.Write(encodingBuffer.Bytes())
		if err != nil {
			return fmt.Errorf("cannot write encoded node to buffer: %w", err)
		}
		return nil
	}

	// otherwise, hash encoded node
	hasher := hasherPool.Get().(hash.Hash)
	hasher.Reset()
	defer hasherPool.Put(hasher)

	// Note: using the sync.Pool's buffer is useful here.
	_, err = hasher.Write(encodingBuffer.Bytes())
	if err != nil {
		return fmt.Errorf("cannot hash encoded node: %w", err)
	}

	_, err = digestBuffer.Write(hasher.Sum(nil))
	if err != nil {
		return fmt.Errorf("cannot write hash sum of node to buffer: %w", err)
	}
	return nil
}

var ErrNodeTypeUnsupported = errors.New("node type is not supported")

type bytesBuffer interface {
	// note: cannot compose with io.Writer for mock generation
	Write(p []byte) (n int, err error)
	Len() int
	Bytes() []byte
}

// encodeNode writes the encoding of the node to the buffer given.
// It is the high-level function wrapping the encoding for different
// node types. The encoding has the following format:
// NodeHeader | Extra partial key length | Partial Key | Value
func encodeNode(n Node, buffer bytesBuffer, parallel bool) (err error) {
	switch n := n.(type) {
	case *Branch:
		err := encodeBranch(n, buffer, parallel)
		if err != nil {
			return fmt.Errorf("cannot encode branch: %w", err)
		}
		return nil
	case *Leaf:
		err := encodeLeaf(n, buffer)
		if err != nil {
			return fmt.Errorf("cannot encode leaf: %w", err)
		}

		n.encodingMu.Lock()
		defer n.encodingMu.Unlock()

		// TODO remove this copying since it defeats the purpose of `buffer`
		// and the sync.Pool.
		n.encoding = make([]byte, buffer.Len())
		copy(n.encoding, buffer.Bytes())
		return nil
	case nil:
		_, err := buffer.Write([]byte{0})
		if err != nil {
			return fmt.Errorf("cannot encode nil node: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("%w: %T", ErrNodeTypeUnsupported, n)
	}
}

// encodeBranch encodes a branch with the encoding specified at the top of this package
// to the buffer given.
func encodeBranch(b *Branch, buffer io.Writer, parallel bool) (err error) {
	if !b.dirty && b.encoding != nil {
		_, err = buffer.Write(b.encoding)
		if err != nil {
			return fmt.Errorf("cannot write stored encoding to buffer: %w", err)
		}
		return nil
	}

	encodedHeader, err := b.header()
	if err != nil {
		return fmt.Errorf("cannot encode header: %w", err)
	}

	_, err = buffer.Write(encodedHeader)
	if err != nil {
		return fmt.Errorf("cannot write encoded header to buffer: %w", err)
	}

	keyLE := nibblesToKeyLE(b.key)
	_, err = buffer.Write(keyLE)
	if err != nil {
		return fmt.Errorf("cannot write encoded key to buffer: %w", err)
	}

	childrenBitmap := common.Uint16ToBytes(b.childrenBitmap())
	_, err = buffer.Write(childrenBitmap)
	if err != nil {
		return fmt.Errorf("cannot write children bitmap to buffer: %w", err)
	}

	if b.value != nil {
		bytes, err := scale.Marshal(b.value)
		if err != nil {
			return fmt.Errorf("cannot scale encode value: %w", err)
		}

		_, err = buffer.Write(bytes)
		if err != nil {
			return fmt.Errorf("cannot write encoded value to buffer: %w", err)
		}
	}

	if parallel {
		err = encodeChildrenInParallel(b.children, buffer)
	} else {
		err = encodeChildrenSequentially(b.children, buffer)
	}
	if err != nil {
		return fmt.Errorf("cannot encode children of branch: %w", err)
	}

	return nil
}

func encodeChildrenInParallel(children [16]Node, buffer io.Writer) (err error) {
	type result struct {
		index  int
		buffer *bytes.Buffer
		err    error
	}

	resultsCh := make(chan result)

	for i, child := range children {
		go func(index int, child Node) {
			buffer := encodingBufferPool.Get().(*bytes.Buffer)
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

			encodingBufferPool.Put(resultBuffers[currentIndex])
			resultBuffers[currentIndex] = nil

			currentIndex++
		}
	}

	for _, buffer := range resultBuffers {
		if buffer == nil { // already emptied and put back in pool
			continue
		}
		encodingBufferPool.Put(buffer)
	}

	return err
}

func encodeChildrenSequentially(children [16]Node, buffer io.Writer) (err error) {
	for i, child := range children {
		err = encodeChild(child, buffer)
		if err != nil {
			return fmt.Errorf("cannot encode child at index %d: %w", i, err)
		}
	}
	return nil
}

func encodeChild(child Node, buffer io.Writer) (err error) {
	var isNil bool
	switch impl := child.(type) {
	case *Branch:
		isNil = impl == nil
	case *Leaf:
		isNil = impl == nil
	default:
		isNil = child == nil
	}
	if isNil {
		return nil
	}

	scaleEncodedChild, err := encodeAndHash(child)
	if err != nil {
		return fmt.Errorf("failed to hash and scale encode child: %w", err)
	}

	_, err = buffer.Write(scaleEncodedChild)
	if err != nil {
		return fmt.Errorf("failed to write child to buffer: %w", err)
	}

	return nil
}

func encodeAndHash(n Node) (b []byte, err error) {
	buffer := digestBufferPool.Get().(*bytes.Buffer)
	buffer.Reset()
	defer digestBufferPool.Put(buffer)

	err = hashNode(n, buffer)
	if err != nil {
		return nil, fmt.Errorf("cannot hash node: %w", err)
	}

	scEncChild, err := scale.Marshal(buffer.Bytes())
	if err != nil {
		return nil, fmt.Errorf("cannot scale encode hashed node: %w", err)
	}
	return scEncChild, nil
}

// encodeLeaf encodes a leaf to the buffer given, with the encoding
// specified at the top of this package.
func encodeLeaf(l *Leaf, buffer io.Writer) (err error) {
	l.encodingMu.RLock()
	defer l.encodingMu.RUnlock()
	if !l.dirty && l.encoding != nil {
		_, err = buffer.Write(l.encoding)
		if err != nil {
			return fmt.Errorf("cannot write stored encoding to buffer: %w", err)
		}
		return nil
	}

	encodedHeader, err := l.header()
	if err != nil {
		return fmt.Errorf("cannot encode header: %w", err)
	}

	_, err = buffer.Write(encodedHeader)
	if err != nil {
		return fmt.Errorf("cannot write encoded header to buffer: %w", err)
	}

	keyLE := nibblesToKeyLE(l.key)
	_, err = buffer.Write(keyLE)
	if err != nil {
		return fmt.Errorf("cannot write LE key to buffer: %w", err)
	}

	encodedValue, err := scale.Marshal(l.value) // TODO scale encoder to write to buffer
	if err != nil {
		return fmt.Errorf("cannot scale marshal value: %w", err)
	}

	_, err = buffer.Write(encodedValue)
	if err != nil {
		return fmt.Errorf("cannot write scale encoded value to buffer: %w", err)
	}

	return nil
}
