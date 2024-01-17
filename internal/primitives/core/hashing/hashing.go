package hashing

import (
	"golang.org/x/crypto/blake2b"
)

// / Do a Blake2 256-bit hash and return result.
//
//	pub fn blake2_256(data: &[u8]) -> [u8; 32] {
//		blake2(data)
//	}
func Blake2_256(data []byte) [32]byte {
	h, err := blake2b.New256(nil)
	if err != nil {
		panic(err)
	}
	_, err = h.Write(data)
	if err != nil {
		panic(err)
	}
	encoded := h.Sum(nil)
	var arr [32]byte
	copy(arr[:], encoded)
	return arr
}
