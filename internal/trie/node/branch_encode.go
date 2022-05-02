// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"bytes"
	"fmt"
	"hash"
	"io"
	"runtime"

	"github.com/ChainSafe/gossamer/internal/trie/pools"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

type encodingAsyncResult struct {
	index  int
	buffer *bytes.Buffer
	err    error
}

func runEncodeChild(child *Node, index int,
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
func encodeChildrenOpportunisticParallel(children []*Node, buffer io.Writer) (err error) {
	// Buffered channels since children might be encoded in this
	// goroutine or another one.
	resultsCh := make(chan encodingAsyncResult, ChildrenCapacity)

	for i, child := range children {
		if child == nil || child.Type() == Leaf {
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
	resultBuffers := make([]*bytes.Buffer, ChildrenCapacity)
	for range children {
		result := <-resultsCh
		if result.err != nil && err == nil { // only set the first error we get
			err = result.err
		}

		resultBuffers[result.index] = result.buffer

		// write as many completed buffers to the result buffer.
		for currentIndex < ChildrenCapacity &&
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

func encodeChildrenSequentially(children []*Node, buffer io.Writer) (err error) {
	for i, child := range children {
		err = encodeChild(child, buffer)
		if err != nil {
			return fmt.Errorf("cannot encode child at index %d: %w", i, err)
		}
	}
	return nil
}

func encodeChild(child *Node, buffer io.Writer) (err error) {
	if child == nil {
		return nil
	}

	scaleEncodedChildHash, err := scaleEncodeHash(child)
	if err != nil {
		return fmt.Errorf("failed to hash and scale encode child: %w", err)
	}

	_, err = buffer.Write(scaleEncodedChildHash)
	if err != nil {
		return fmt.Errorf("failed to write child to buffer: %w", err)
	}

	return nil
}

// scaleEncodeHash hashes the node (blake2b sum on encoded value)
// and then SCALE encodes it. This is used to encode children
// nodes of branches.
func scaleEncodeHash(node *Node) (encoding []byte, err error) {
	buffer := pools.DigestBuffers.Get().(*bytes.Buffer)
	buffer.Reset()
	defer pools.DigestBuffers.Put(buffer)

	err = hashNode(node, buffer)
	if err != nil {
		return nil, fmt.Errorf("cannot hash %s: %w", node.Type(), err)
	}

	encoding, err = scale.Marshal(buffer.Bytes())
	if err != nil {
		return nil, fmt.Errorf("cannot scale encode hashed %s: %w", node.Type(), err)
	}

	return encoding, nil
}

func hashNode(node *Node, digestWriter io.Writer) (err error) {
	encodingBuffer := pools.EncodingBuffers.Get().(*bytes.Buffer)
	encodingBuffer.Reset()
	defer pools.EncodingBuffers.Put(encodingBuffer)

	err = node.Encode(encodingBuffer)
	if err != nil {
		return fmt.Errorf("cannot encode %s: %w", node.Type(), err)
	}

	// if length of encoded leaf is less than 32 bytes, do not hash
	if encodingBuffer.Len() < 32 {
		_, err = digestWriter.Write(encodingBuffer.Bytes())
		if err != nil {
			return fmt.Errorf("cannot write encoded %s to buffer: %w", node.Type(), err)
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
		return fmt.Errorf("cannot hash encoding of %s: %w", node.Type(), err)
	}

	_, err = digestWriter.Write(hasher.Sum(nil))
	if err != nil {
		return fmt.Errorf("cannot write hash sum of %s to buffer: %w", node.Type(), err)
	}
	return nil
}
