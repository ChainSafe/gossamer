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
			panic("cannot create Blake2b-256 hasher: " + err.Error())
		}
		return hasher
	},
}

func hashNode(n node, digestBuffer io.Writer) (err error) {
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
		return err
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
	return err
}

var ErrNodeTypeUnsupported = errors.New("node type is not supported")

// encodeNode writes the encoding of the node to the buffer given.
// It is the high-level function wrapping the encoding for different
// node types. The encoding has the following format:
// NodeHeader | Extra partial key length | Partial Key | Value
func encodeNode(n node, buffer *bytes.Buffer, parallel bool) (err error) {
	switch n := n.(type) {
	case *branch:
		err := encodeBranch(n, buffer, parallel)
		if err != nil {
			return fmt.Errorf("cannot encode branch: %w", err)
		}
		return nil
	case *leaf:
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
		buffer.Write([]byte{0})
		return nil
	default:
		return fmt.Errorf("%w: %T", ErrNodeTypeUnsupported, n)
	}
}

func encodeAndHash(n node) ([]byte, error) {
	buffer := digestBufferPool.Get().(*bytes.Buffer)
	buffer.Reset()
	defer digestBufferPool.Put(buffer)

	err := hashNode(n, buffer)
	if err != nil {
		return nil, err
	}

	scEncChild, err := scale.Marshal(buffer.Bytes())
	if err != nil {
		return nil, err
	}
	return scEncChild, nil
}

// encodeBranch encodes a branch with the encoding specified at the top of this package
// to the buffer given.
func encodeBranch(b *branch, buffer io.Writer, parallel bool) (err error) {
	if !b.dirty && b.encoding != nil {
		_, err = buffer.Write(b.encoding)
		if err != nil {
			return fmt.Errorf("cannot write stored encoded branch to buffer: %w", err)
		}
		return nil
	}

	encoding, err := b.header()
	if err != nil {
		return fmt.Errorf("cannot encode branch header: %w", err)
	}

	_, err = buffer.Write(encoding)
	if err != nil {
		return fmt.Errorf("cannot write encoded branch header to buffer: %w", err)
	}

	_, err = buffer.Write(nibblesToKeyLE(b.key))
	if err != nil {
		return fmt.Errorf("cannot write encoded branch key to buffer: %w", err)
	}

	_, err = buffer.Write(common.Uint16ToBytes(b.childrenBitmap()))
	if err != nil {
		return fmt.Errorf("cannot write branch children bitmap to buffer: %w", err)
	}

	if b.value != nil {
		bytes, err := scale.Marshal(b.value)
		if err != nil {
			return fmt.Errorf("cannot scale encode branch value: %w", err)
		}

		_, err = buffer.Write(bytes)
		if err != nil {
			return fmt.Errorf("cannot write encoded branch value to buffer: %w", err)
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

func encodeChildrenInParallel(children [16]node, buffer io.Writer) (err error) {
	type result struct {
		index  int
		buffer *bytes.Buffer
		err    error
	}

	resultsCh := make(chan result)

	for i, child := range children {
		go func(index int, child node) {
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
			// note buffer.Write copies the byte slice given as argument
			_, writeErr := buffer.Write(resultBuffers[currentIndex].Bytes())
			if writeErr != nil && err == nil {
				err = writeErr
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

func encodeChildrenSequentially(children [16]node, buffer io.Writer) (err error) {
	for _, child := range children {
		err = encodeChild(child, buffer)
		if err != nil {
			return err
		}
	}
	return nil
}

func encodeChild(child node, buffer io.Writer) (err error) {
	if child == nil {
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

// encodeLeaf encodes a leaf to the buffer given, with the encoding
// specified at the top of this package.
func encodeLeaf(l *leaf, buffer io.Writer) (err error) {
	l.encodingMu.RLock()
	defer l.encodingMu.RUnlock()
	if !l.dirty && l.encoding != nil {
		_, err = buffer.Write(l.encoding)
		if err != nil {
			return fmt.Errorf("cannot write stored encoding to buffer: %w", err)
		}
		return nil
	}

	encoding, err := l.header()
	if err != nil {
		return fmt.Errorf("cannot encode header: %w", err)
	}

	_, err = buffer.Write(encoding)
	if err != nil {
		return fmt.Errorf("cannot write encoded header to buffer: %w", err)
	}

	_, err = buffer.Write(nibblesToKeyLE(l.key))
	if err != nil {
		return fmt.Errorf("cannot write LE key to buffer: %w", err)
	}

	bytes, err := scale.Marshal(l.value) // TODO scale encoder to write to buffer
	if err != nil {
		return err
	}

	_, err = buffer.Write(bytes)
	if err != nil {
		return fmt.Errorf("cannot write scale encoded value to buffer: %w", err)
	}

	return nil
}
