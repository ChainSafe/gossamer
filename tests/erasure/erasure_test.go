package erasure

import (
	"fmt"
	"github.com/klauspost/reedsolomon"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestErasureEncoding(t *testing.T) {
	enc, err := reedsolomon.New(10, 3)
	require.NoError(t, err)
	data := make([][]byte, 13)

	// Create all shards, size them at 50000 each
	//for i := 1; i < 10; i++ {
	//	data[i] = make([]byte, 50000)
	//}
	data = enc.(reedsolomon.Extensions).AllocAligned(50000)
	// fill data
	for i, in := range data[:10] {
		for j := range in {
			in[j] = byte((i + j) & 0xff)
		}
	}

	err = enc.Encode(data)

	ok, err := enc.Verify(data)
	fmt.Printf("OK %v, err %v\n", ok, err)

	fmt.Printf("data3 %v\n", data[3][:10])
	data[3] = nil
	fmt.Printf("data3  after nil %v\n", data[3])
	err = enc.Reconstruct(data)
	fmt.Printf("error %v\n", err)
	fmt.Printf("data 3 reconstruct %v\n", data[3][:10])

}
