package network

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Benchmark_sizedBufferPool(b *testing.B) {
	sbp := newSizedBufferPool(100, 200)

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
	const maxQueueSize = 2
	const maxIndex = maxMessageSize - 1

	pool := newSizedBufferPool(preAlloc, maxQueueSize)

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
