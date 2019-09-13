// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package common

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"math/big"

	log "github.com/ChainSafe/log15"
)

// Length of hashes in bytes.
const (
	// HashLength is the expected length of the hash
	HashLength = 32
)

// Hash used to store a blake2b hash
type Hash [32]byte

func (h *Hash) String() string {
	return fmt.Sprintf("0x%x", h[:])
}

// BlockHeader is the header of a Polkadot block
type BlockHeader struct {
	ParentHash     Hash     // the block hash of the block's parent
	Number         *big.Int // block number
	StateRoot      Hash     // the root of the state trie
	ExtrinsicsRoot Hash     // the root of the extrinsics trie
	Digest         []byte   // any additional block info eg. logs, seal
}

// NewHash casts a byte array to a Hash
// if the input is longer than 32 bytes, it takes the first 32 bytes
func NewHash(in []byte) (res Hash) {
	res = [32]byte{}
	copy(res[:], in)
	return res
}

// ToBytes turns a hash to a byte array
func (h Hash) ToBytes() []byte {
	b := [32]byte(h)
	return b[:]
}

// BytesToHash sets b to hash.
// If b is larger than len(h), b will be cropped from the left.
func BytesToHash(b []byte) Hash {
	var h Hash
	h.SetBytes(b)
	return h
}

// SetBytes sets the hash to the value of b.
// If b is larger than len(h), b will be cropped from the left.
func (h *Hash) SetBytes(b []byte) {
	if len(b) > len(h) {
		b = b[len(b)-HashLength:]
	}

	copy(h[HashLength-len(b):], b)
}

func ToBytes(key interface{}) []byte {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(key)
	if err != nil {
		log.Crit("error converting to bytes", "error", err)
		return nil
	}
	return buf.Bytes()
}
