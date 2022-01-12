// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"bytes"
	"fmt"
	"hash" //nolint
	"io"
	"runtime"

	"github.com/ChainSafe/gossamer/internal/trie/codec" //nolint
	"github.com/ChainSafe/gossamer/internal/trie/pools"
	"github.com/ChainSafe/gossamer/lib/common"
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
		return nil, fmt.Errorf("cannot hash branch: %w", err)
	}

	encoding, err = scale.Marshal(buffer.Bytes())
	if err != nil {
		return nil, fmt.Errorf("cannot scale encode hashed branch: %w", err)
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

	// if length of encoded branch is less than 32 bytes, do not hash
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
func (b *Branch) Encode(buffer Buffer) (err error) {
	if !b.Dirty && b.Encoding != nil {
		_, err = buffer.Write(b.Encoding)
		if err != nil {
			return fmt.Errorf("cannot write stored encoding to buffer: %w", err)
		}
		return nil
	}

	err = b.encodeHeader(buffer)
	if err != nil {
		return fmt.Errorf("cannot encode header: %w", err)
	}

	keyLE := codec.NibblesToKeyLE(b.Key)
	_, err = buffer.Write(keyLE) //nolint
	if err != nil {
		return fmt.Errorf("cannot write encoded key to buffer: %w", err)
	}

	childrenBitmap := common.Uint16ToBytes(b.ChildrenBitmap())
	_, err = buffer.Write(childrenBitmap) //nolint
	if err != nil {
		return fmt.Errorf("cannot write children bitmap to buffer: %w", err)
	}

	if b.Value != nil {
		bytes, err := scale.Marshal(b.Value)
		if err != nil {
			return fmt.Errorf("cannot scale encode value: %w", err)
		}

		_, err = buffer.Write(bytes) //nolint
		if err != nil {
			return fmt.Errorf("cannot write encoded value to buffer: %w", err)
		}
	}

	err = encodeChildrenOpportunisticParallel(b.Children, buffer)
	if err != nil {
		return fmt.Errorf("cannot encode children of branch: %w", err)
	}

	return nil
}

type encodingAsyncResult struct {
	index  int
	buffer *bytes.Buffer
	err    error
}

func runEncodeChild(child Node, index int,
	results chan<- encodingAsyncResult, rateLimit <-chan struct{}) {
	buffer := pools.EncodingBuffers.Get().(*bytes.Buffer)
	buffer.Reset()
	// buffer is put back in the pool after processing its
	// data in the select block below.

	err := encodeChild(child, buffer)

	results <- encodingAsyncResult{
		index:  index,
		buffer: buffer,
		err:    err,
	}
	if rateLimit != nil {
		// Only run if runEncodeChild is launched
		// in its own goroutine.
		<-rateLimit
	}
}

var parallelLimit = runtime.NumCPU()

var parallelEncodingRateLimit = make(chan struct{}, parallelLimit)

// encodeChildrenOpportunisticParallel encodes children in parallel eventually.
// Leaves are encoded in a blocking way, and branches are encoded in separate
// goroutines IF they are less than the parallelLimit number of goroutines already
// running. This is designed to limit the total number of goroutines in order to
// avoid using too much memory on the stack.
func encodeChildrenOpportunisticParallel(children [16]Node, buffer io.Writer) (err error) {
	// Buffered channels since children might be encoded in this
	// goroutine or another one.
	resultsCh := make(chan encodingAsyncResult, len(children))

	for i, child := range children {
		if isNodeNil(child) || child.Type() == LeafType {
			runEncodeChild(child, i, resultsCh, nil)
			continue
		}

		// Branch child
		select {
		case parallelEncodingRateLimit <- struct{}{}:
			// We have a goroutine available to encode
			// the branch in parallel.
			go runEncodeChild(child, i, resultsCh, parallelEncodingRateLimit)
		default:
			// we reached the maximum parallel goroutines
			// so encode this branch in this goroutine
			runEncodeChild(child, i, resultsCh, nil)
		}
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
			if err == nil && len(bufferSlice) > 0 {
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

func encodeChildrenSequentially(children [16]Node, buffer io.Writer) (err error) {
	for i, child := range children {
		err = encodeChild(child, buffer)
		if err != nil {
			return fmt.Errorf("cannot encode child at index %d: %w", i, err)
		}
	}
	return nil
}

func isNodeNil(n Node) (isNil bool) {
	switch impl := n.(type) {
	case *Branch:
		isNil = impl == nil
	case *Leaf:
		isNil = impl == nil
	default:
		isNil = n == nil
	}
	return isNil
}

func encodeChild(child Node, buffer io.Writer) (err error) {
	if isNodeNil(child) {
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
