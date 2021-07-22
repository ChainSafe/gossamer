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
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
)

const (
	// HashLength is the expected length of the common.Hash type
	HashLength = 32
)

// Hash used to store a blake2b hash
type Hash [32]byte

// EmptyHash is an empty [32]byte{}
var EmptyHash = Hash{}

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

// HashValidator validates hash fields
func HashValidator(field reflect.Value) interface{} {
	// Try to convert to hash type.
	if valuer, ok := field.Interface().(Hash); ok {
		// Check if the hash is empty.
		if valuer.Equal(Hash{}) {
			return ""
		}
		return valuer.ToBytes()
	}
	return ""
}

// Equal compares two hashes
func (h Hash) Equal(g Hash) bool {
	return bytes.Equal(h[:], g[:])
}

// String returns the hex string for the hash
func (h Hash) String() string {
	return fmt.Sprintf("0x%x", h[:])
}

// SetBytes sets the hash to the value of b.
// If b is larger than len(h), b will be cropped from the left.
func (h *Hash) SetBytes(b []byte) {
	if len(b) > len(h) {
		b = b[len(b)-HashLength:]
	}

	copy(h[HashLength-len(b):], b)
}

// ReadHash reads a 32-byte hash from the reader and returns it
func ReadHash(r io.Reader) (Hash, error) {
	buf := make([]byte, 32)
	_, err := r.Read(buf)
	if err != nil {
		return Hash{}, err
	}
	h := [32]byte{}
	copy(h[:], buf)
	return Hash(h), nil
}

// BytesToHash sets b to hash.
// If b is larger than len(h), b will be cropped from the left.
func BytesToHash(b []byte) Hash {
	var h Hash
	h.SetBytes(b)
	return h
}

// UnmarshalJSON converts hex data to hash
func (h *Hash) UnmarshalJSON(data []byte) error {
	trimmedData := strings.Trim(string(data), "\"")
	if len(trimmedData) < 2 {
		return errors.New("invalid hash format")
	}

	var err error
	if *h, err = HexToHash(trimmedData); err != nil {
		return err
	}
	return nil
}

// MarshalJSON converts hash to hex data
func (h *Hash) MarshalJSON() ([]byte, error) {
	return json.Marshal(h.String())
}

// HexToHash turns a 0x prefixed hex string into type Hash
func HexToHash(in string) (Hash, error) {
	if strings.Compare(in[:2], "0x") != 0 {
		return [32]byte{}, errors.New("could not byteify non 0x prefixed string")
	}
	in = in[2:]
	out, err := hex.DecodeString(in)
	if err != nil {
		return [32]byte{}, err
	}
	var buf = [32]byte{}
	copy(buf[:], out)
	return buf, err
}

// MustHexToHash turns a 0x prefixed hex string into type Hash
// it panics if it cannot turn the string into a Hash
func MustHexToHash(in string) Hash {
	if strings.Compare(in[:2], "0x") != 0 {
		panic("could not byteify non 0x prefixed string")
	}

	in = in[2:]
	out, err := hex.DecodeString(in)
	if err != nil {
		panic(err)
	}

	var buf = [32]byte{}
	copy(buf[:], out)
	return buf
}
