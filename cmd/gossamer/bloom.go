package main

import (
	"encoding/binary"
	"errors"

	"github.com/ChainSafe/gossamer/lib/common"
	log "github.com/ChainSafe/log15"
	bloomfilter "github.com/holiman/bloomfilter/v2"
)

// ErrKeySize is returned when key size does not fit
var ErrKeySize = errors.New("cannot have nil keystore")

type stateBloomHasher []byte

func (f stateBloomHasher) Write(p []byte) (n int, err error) { panic("not implemented") }
func (f stateBloomHasher) Sum(b []byte) []byte               { panic("not implemented") }
func (f stateBloomHasher) Reset()                            { panic("not implemented") }
func (f stateBloomHasher) BlockSize() int                    { panic("not implemented") }
func (f stateBloomHasher) Size() int                         { return 8 }
func (f stateBloomHasher) Sum64() uint64                     { return binary.BigEndian.Uint64(f) }

// stateBloom is a wrapper for bloom filter.
// The keys of all generated entries will be recorded here so that in the pruning
// stage the entries belong to the specific version can be avoided for deletion.
type stateBloom struct {
	bloom *bloomfilter.Filter
}

// newStateBloomWithSize creates a brand new state bloom for state generation
// The bloom filter will be created by the passing bloom filter size. the parameters
// are picked so that the false-positive rate for mainnet is low enough.
func newStateBloomWithSize(size uint64) (*stateBloom, error) {
	bloom, err := bloomfilter.New(size*1024*1024*8, 4)
	if err != nil {
		return nil, err
	}
	log.Info("initialised state bloom", "size", float64(bloom.M()/8))
	return &stateBloom{bloom: bloom}, nil
}

// put writes key to bloom filter
func (sb *stateBloom) put(key []byte) error {
	if len(key) != common.HashLength {
		return ErrKeySize
	}

	sb.bloom.Add(stateBloomHasher(key))
	return nil
}

// contain is the wrapper of the underlying contains function which
// reports whether the key is contained.
// - If it says yes, the key may be contained
// - If it says no, the key is definitely not contained.
func (sb *stateBloom) contain(key []byte) bool {
	return sb.bloom.Contains(stateBloomHasher(key))
}
