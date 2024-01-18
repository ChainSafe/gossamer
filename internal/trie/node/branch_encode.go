// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"bytes"
	"fmt"
	"io"
	"runtime"

	"github.com/ChainSafe/gossamer/pkg/scale"
)

type encodingAsyncResult struct {
	index  int
	buffer *bytes.Buffer
	err    error
}

func runEncodeChild(child *Node, index, maxInlineValue int,
	results chan<- encodingAsyncResult, rateLimit <-chan struct{}) {
	buffer := bytes.NewBuffer(nil)
	err := encodeChild(child, maxInlineValue, buffer)

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
// goroutines IF they are less than the parallelLimit number of goroutines al.y
// running. This is designed to limit the total number of goroutines in order to
// avoid using too much memory on the stack.
func encodeChildrenOpportunisticParallel(children []*Node, maxInlineValue int, buffer io.Writer) (err error) {
	// Buffered channels since children might be encoded in this
	// goroutine or another one.
	resultsCh := make(chan encodingAsyncResult, ChildrenCapacity)

	for i, child := range children {
		if child == nil {
			resultsCh <- encodingAsyncResult{index: i}
			continue
		}

		if child.Kind() == Leaf {
			runEncodeChild(child, i, maxInlineValue, resultsCh, nil)
			continue
		}

		// Branch child
		select {
		case parallelEncodingRateLimit <- struct{}{}:
			// We have a goroutine available to encode
			// the branch in parallel.
			go runEncodeChild(child, i, maxInlineValue, resultsCh, parallelEncodingRateLimit)
		default:
			// we reached the maximum parallel goroutines
			// so encode this branch in this goroutine
			runEncodeChild(child, i, maxInlineValue, resultsCh, nil)
		}
	}

	currentIndex := 0
	indexToBuffer := make(map[int]*bytes.Buffer, ChildrenCapacity)
	for range children {
		result := <-resultsCh
		if result.err != nil && err == nil { // only set the first error we get
			err = result.err
		}

		indexToBuffer[result.index] = result.buffer

		// write as many completed buffers to the result buffer.
		for currentIndex < ChildrenCapacity {
			resultBuffer, done := indexToBuffer[currentIndex]
			if !done {
				break
			}

			delete(indexToBuffer, currentIndex)

			nilChildNode := resultBuffer == nil
			if nilChildNode {
				currentIndex++
				continue
			}

			bufferSlice := resultBuffer.Bytes()
			if err == nil && len(bufferSlice) > 0 {
				// note buffer.Write copies the byte slice given as argument
				_, writeErr := buffer.Write(bufferSlice)
				if writeErr != nil && err == nil {
					err = fmt.Errorf(
						"cannot write encoding of child at index %d: %w",
						currentIndex, writeErr)
				}
			}

			currentIndex++
		}
	}

	return err
}

// encodeChild computes the Merkle value of the node
// and then SCALE encodes it to the given buffer.
func encodeChild(child *Node, maxInlineValue int, buffer io.Writer) (err error) {
	merkleValue, err := child.CalculateMerkleValue(maxInlineValue)
	if err != nil {
		return fmt.Errorf("computing %s Merkle value: %w", child.Kind(), err)
	}

	encoder := scale.NewEncoder(buffer)
	err = encoder.Encode(merkleValue)
	if err != nil {
		return fmt.Errorf("scale encoding Merkle value: %w", err)
	}

	return nil
}
