// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"encoding/binary"
	"errors"

	"github.com/ChainSafe/gossamer/lib/common"
	bloomfilter "github.com/holiman/bloomfilter/v2"
)

// ErrKeySize is returned when key size does not fit
var ErrKeySize = errors.New("cannot have nil keystore")

type bloomStateHasher []byte

func (f bloomStateHasher) Write(p []byte) (n int, err error) { panic("not implemented") }
func (f bloomStateHasher) Sum(b []byte) []byte               { panic("not implemented") }
func (f bloomStateHasher) Reset()                            { panic("not implemented") }
func (f bloomStateHasher) BlockSize() int                    { panic("not implemented") }
func (f bloomStateHasher) Size() int                         { return 8 }
func (f bloomStateHasher) Sum64() uint64                     { return binary.BigEndian.Uint64(f) }

// bloomState is a wrapper for bloom filter.
// The keys of all generated entries will be recorded here so that in the pruning
// stage the entries belong to the specific version can be avoided for deletion.
type bloomState struct {
	bloom *bloomfilter.Filter
}

// newBloomState creates a brand new state bloom for state generation
// The bloom filter will be created by the passing bloom filter size. the parameters
// are picked so that the false-positive rate for mainnet is low enough.
func newBloomState(size uint64) (*bloomState, error) {
	bloom, err := bloomfilter.New(size*1024*1024*8, 4)
	if err != nil {
		return nil, err
	}
	logger.Infof("initialised state bloom with size %f", float64(bloom.M()/8))
	return &bloomState{bloom: bloom}, nil
}

// put writes key to bloom filter
func (sb *bloomState) put(key []byte) error {
	if len(key) != common.HashLength {
		return ErrKeySize
	}

	sb.bloom.Add(bloomStateHasher(key))
	return nil
}

// contain is the wrapper of the underlying contains function which
// reports whether the key is contained.
// - If it says yes, the key may be contained
// - If it says no, the key is definitely not contained.
func (sb *bloomState) contain(key []byte) bool {
	return sb.bloom.Contains(bloomStateHasher(key))
}
