// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Benchmark_sizedBufferPool(b *testing.B) {
	const preAllocate = 100
	const poolSize = 200
	sbp := newSizedBufferPool(preAllocate, poolSize)

	b.RunParallel(func(p *testing.PB) {
		for p.Next() {
			buffer := sbp.get()
			buffer[0] = 1
			buffer[len(buffer)-1] = 1
			sbp.put(buffer)
		}
	})
}

// Before:        104853	    	 11119 ns/op	   65598 B/op	       1 allocs/op
// Array ptr:     2742781	       438.3 ns/op	       2 B/op	       0 allocs/op
// Slices:        2560960	       463.8 ns/op	       2 B/op	       0 allocs/op
// Slice pointer: 2683528	       460.8 ns/op	       2 B/op	       0 allocs/op

func Test_sizedBufferPool(t *testing.T) {
	t.Parallel()

	const preAlloc = 1
	const poolSize = 2
	const maxIndex = maxMessageSize - 1

	pool := newSizedBufferPool(preAlloc, poolSize)

	first := pool.get() // pre-allocated one
	first[maxIndex] = 1

	second := pool.get() // new one
	second[maxIndex] = 2

	third := pool.get() // new one
	third[maxIndex] = 3

	fourth := pool.get() // new one
	fourth[maxIndex] = 4

	pool.put(fourth)
	pool.put(third)
	pool.put(second) // discarded
	pool.put(first)  // discarded

	b := pool.get() // fourth
	assert.Equal(t, byte(4), b[maxIndex])

	b = pool.get() // third
	assert.Equal(t, byte(3), b[maxIndex])
}

func Test_sizedBufferPool_race(t *testing.T) {
	t.Parallel()

	const preAlloc = 1
	const poolSize = 2

	pool := newSizedBufferPool(preAlloc, poolSize)

	const parallelism = 4

	readyWait := new(sync.WaitGroup)
	readyWait.Add(parallelism)

	doneWait := new(sync.WaitGroup)
	doneWait.Add(parallelism)

	// run for 50ms
	ctxTimerStarted := make(chan struct{})
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		const timeout = 50 * time.Millisecond
		readyWait.Wait()
		ctx, cancel = context.WithTimeout(ctx, timeout)
		close(ctxTimerStarted)
	}()
	defer cancel()

	for i := 0; i < parallelism; i++ {
		go func() {
			defer doneWait.Done()
			readyWait.Done()
			readyWait.Wait()
			<-ctxTimerStarted

			for ctx.Err() != nil {
				// test relies on the -race detector
				// to detect concurrent writes to the buffer.
				b := pool.get()
				b[0] = 1
				pool.put(b)
			}
		}()
	}

	doneWait.Wait()
}
