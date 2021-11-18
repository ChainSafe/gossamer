// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package variadic

import (
	"encoding/binary"
	"errors"
	"io"

	"github.com/ChainSafe/gossamer/lib/common"
)

// Uint64OrHash represents an optional interface type (int,hash).
type Uint64OrHash struct {
	value interface{}
}

// NewUint64OrHash returns a new variadic.Uint64OrHash given an int, uint64, or Hash
func NewUint64OrHash(value interface{}) (*Uint64OrHash, error) {
	switch v := value.(type) {
	case int:
		return &Uint64OrHash{
			value: uint64(v),
		}, nil
	case uint64:
		return &Uint64OrHash{
			value: v,
		}, nil
	case common.Hash:
		return &Uint64OrHash{
			value: v,
		}, nil
	default:
		return nil, errors.New("value is not uint64 or common.Hash")
	}
}

// MustNewUint64OrHash returns a new variadic.Uint64OrHash given an int, uint64, or Hash
// It panics if the input value is invalid
func MustNewUint64OrHash(value interface{}) *Uint64OrHash {
	val, err := NewUint64OrHash(value)
	if err != nil {
		panic(err)
	}

	return val
}

// NewUint64OrHashFromBytes returns a new variadic.Uint64OrHash from an encoded variadic uint64 or hash
func NewUint64OrHashFromBytes(data []byte) *Uint64OrHash {
	firstByte := data[0]
	if firstByte == 0 {
		return &Uint64OrHash{
			value: common.NewHash(data[1:]),
		}
	} else if firstByte == 1 {
		num := data[1:]
		if len(num) < 8 {
			num = common.AppendZeroes(num, 8)
		}
		return &Uint64OrHash{
			value: binary.LittleEndian.Uint64(num),
		}
	} else {
		return nil
	}
}

// Value returns the interface value.
func (x *Uint64OrHash) Value() interface{} {
	if x == nil {
		return nil
	}
	return x.value
}

// IsHash returns true if the value is a hash
func (x *Uint64OrHash) IsHash() bool {
	if x == nil {
		return false
	}
	_, is := x.value.(common.Hash)
	return is
}

// Hash returns the value as a common.Hash. it panics if the value is not a hash.
func (x *Uint64OrHash) Hash() common.Hash {
	if !x.IsHash() {
		panic("value is not common.Hash")
	}

	return x.value.(common.Hash)
}

// IsUint64 returns true if the value is a hash
func (x *Uint64OrHash) IsUint64() bool {
	if x == nil {
		return false
	}
	_, is := x.value.(uint64)
	return is
}

// Uint64 returns the value as a uint64. it panics if the value is not a hash.
func (x *Uint64OrHash) Uint64() uint64 {
	if !x.IsUint64() {
		panic("value is not uint64")
	}

	return x.value.(uint64)
}

// Encode will encode a uint64 or hash into the SCALE spec
func (x *Uint64OrHash) Encode() ([]byte, error) {
	var encMsg []byte
	switch c := x.Value().(type) {
	case uint64:
		startingBlockByteArray := make([]byte, 8)
		binary.LittleEndian.PutUint64(startingBlockByteArray, c)

		encMsg = append(encMsg, append([]byte{1}, startingBlockByteArray...)...)
	case common.Hash:
		encMsg = append(encMsg, append([]byte{0}, c.ToBytes()...)...)
	}
	return encMsg, nil
}

// Decode will decode the Uint64OrHash into a hash or uint64
func (x *Uint64OrHash) Decode(r io.Reader) error {
	startingBlockType, err := common.ReadByte(r)
	if err != nil {
		return err
	}
	if startingBlockType == 0 {
		hash := make([]byte, 32)
		_, err = r.Read(hash)
		if err != nil {
			return err
		}
		x.value = common.NewHash(hash)
	} else {
		num := make([]byte, 8)
		_, err = r.Read(num)
		if err != nil {
			return err
		}
		x.value = binary.LittleEndian.Uint64(num)
	}
	return nil
}
