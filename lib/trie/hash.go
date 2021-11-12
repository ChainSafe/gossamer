// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	"bytes"
	"errors"
	"fmt"
	"hash"
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

func hashNode(n node, digestBuffer *bytes.Buffer) (err error) {
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

// encodeBranch encodes a branch with the encoding specified in hashedOrEncodedNode
// to the buffer given.
func encodeBranch(b *branch, buffer *bytes.Buffer, parallel bool) (err error) {
	if !b.dirty && b.encoding != nil {
		_, err = buffer.Write(b.encoding)
		return err
	}

	encoding, err := b.header()
	if err != nil {
		return err
	}

	buffer.Write(encoding)
	buffer.Write(nibblesToKeyLE(b.key))
	buffer.Write(common.Uint16ToBytes(b.childrenBitmap()))

	if b.value != nil {
		bytes, err := scale.Marshal(b.value)
		if err != nil {
			return err
		}
		buffer.Write(bytes)
	}

	if parallel {
		return encodeChildsInParallel(b.children, buffer)
	}
	return encodeChildsSequentially(b.children, buffer)
}

func encodeChildsInParallel(children [16]node, buffer *bytes.Buffer) (err error) {
	type result struct {
		index  int
		buffer *bytes.Buffer
	}

	resultsCh := make(chan result)
	errorCh := make(chan error)

	for i, child := range children {
		go func(index int, child node) {
			buffer := encodingBufferPool.Get().(*bytes.Buffer)
			buffer.Reset()
			// buffer is put back in the pool after processing its
			// data in the select block below.

			err := encodeChild(child, buffer)
			if err != nil {
				errorCh <- err
				return
			}

			resultsCh <- result{
				index:  index,
				buffer: buffer,
			}
		}(i, child)
	}

	currentIndex := 0
	resultBuffers := make([]*bytes.Buffer, len(children))
	for range children {
		select {
		case result := <-resultsCh:
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
		case newErr := <-errorCh:
			if err == nil { // only set the first error we get
				err = newErr
			}
		}
	}

	return err
}

func encodeChildsSequentially(children [16]node, buffer *bytes.Buffer) (err error) {
	for _, child := range children {
		err = encodeChild(child, buffer)
		if err != nil {
			return err
		}
	}
	return nil
}

func encodeChild(child node, buffer *bytes.Buffer) (err error) {
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
func encodeLeaf(l *leaf, buffer *bytes.Buffer) (err error) {
	if !l.dirty && l.encoding != nil {
		_, err = buffer.Write(l.encoding)
		return err
	}

	encoding, err := l.header()
	if err != nil {
		return err
	}
	_, _ = buffer.Write(encoding)

	_, _ = buffer.Write(nibblesToKeyLE(l.key))

	bytes, err := scale.Marshal(l.value) // TODO scale encoder to write to buffer
	if err != nil {
		return err
	}

	_, _ = buffer.Write(bytes)
	return nil
}
