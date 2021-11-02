package network

import "testing"

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

// Before: 				104853	    	 11119 ns/op	   65598 B/op	       1 allocs/op
// Array ptr: 		2742781	       438.3 ns/op	       2 B/op	       0 allocs/op
// Slices: 				2560960	       463.8 ns/op	       2 B/op	       0 allocs/op
// Slice pointer: 2683528	       460.8 ns/op	       2 B/op	       0 allocs/op
