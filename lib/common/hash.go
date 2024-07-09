// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package common

import (
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

var EmptyHash = Hash{}

// Hash used to store a blake2b hash
type Hash [32]byte

// NewHash casts a byte array to a Hash
// if the input is longer than 32 bytes, it takes the first 32 bytes
func NewHash(in []byte) (res Hash) {
	res = [32]byte{}
	copy(res[:], in)
	return res
}

// ToBytes turns a hash to a byte array
func (h Hash) ToBytes() []byte { //skipcq: GO-W1029
	b := [32]byte(h)
	return b[:]
}

// ToBytes turns a hash to a byte array
func (h Hash) Bytes() []byte { //skipcq: GO-W1029
	b := [32]byte(h)
	return b[:]
}

// HashValidator validates hash fields
func HashValidator(field reflect.Value) interface{} {
	// Try to convert to hash type.
	if valuer, ok := field.Interface().(Hash); ok {
		// Check if the hash is empty.
		if valuer == (EmptyHash) {
			return ""
		}
		return valuer.ToBytes()
	}
	return ""
}

// IsEmpty returns true if the hash is empty, false otherwise.
func (h Hash) IsEmpty() bool { //skipcq: GO-W1029
	return h == EmptyHash
}

// String returns the hex string for the hash
func (h Hash) String() string { //skipcq: GO-W1029
	return fmt.Sprintf("0x%x", h[:])
}

// Short returns the first 4 bytes and the last 4 bytes of the hex string for the hash
func (h Hash) Short() string { //skipcq: GO-W1029
	const nBytes = 4
	return fmt.Sprintf("0x%x...%x", h[:nBytes], h[len(h)-nBytes:])
}

// SetBytes sets the hash to the value of b.
// If b is larger than len(h), b will be cropped from the left.
func (h *Hash) SetBytes(b []byte) { //skipcq: GO-W1029
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
		return EmptyHash, err
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
func (h *Hash) UnmarshalJSON(data []byte) error { //skipcq: GO-W1029
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
func (h Hash) MarshalJSON() ([]byte, error) { //skipcq: GO-W1029
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
