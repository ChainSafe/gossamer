package rawdb

import "github.com/ChainSafe/gossamer/common"

var (
	// Data prefixes
	headerPrefix = []byte("hdr") // headerPrefix + hash -> header
)

// headerKey = headerPrefix + num (uint64 big endian) + hash
func headerKey(hash common.Hash) []byte {
	return append(headerPrefix, hash.ToBytes()...)
}