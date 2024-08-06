// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package variadic

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/ChainSafe/gossamer/lib/common"
)

// Uint32OrHash represents a variadic type that is either uint32 or common.Hash.
type Uint32OrHash struct {
	value interface{}
}

func FromHash(hash common.Hash) *Uint32OrHash {
	return &Uint32OrHash{
		value: hash,
	}
}

func FromUint32(value uint32) *Uint32OrHash {
	return &Uint32OrHash{
		value: value,
	}
}

// NewUint32OrHash returns a new variadic.Uint32OrHash given an int, uint32, or Hash
func NewUint32OrHash(value interface{}) (*Uint32OrHash, error) {
	switch v := value.(type) {
	case int: // in order to accept constants int such as `NewUint32OrHash(1)`
		return &Uint32OrHash{
			value: uint32(v),
		}, nil
	case uint:
		return &Uint32OrHash{
			value: uint32(v),
		}, nil
	case uint32:
		return &Uint32OrHash{
			value: v,
		}, nil
	case common.Hash:
		return &Uint32OrHash{
			value: v,
		}, nil
	default:
		return nil, errors.New("value is not uint32 or common.Hash")
	}
}

// MustNewUint32OrHash returns a new variadic.Uint32OrHash given an int, uint32, or Hash
// It panics if the input value is invalid
func MustNewUint32OrHash(value interface{}) *Uint32OrHash {
	val, err := NewUint32OrHash(value)
	if err != nil {
		panic(err)
	}

	return val
}

// NewUint32OrHashFromBytes returns a new variadic.Uint32OrHash from an encoded variadic uint32 or hash
func NewUint32OrHashFromBytes(data []byte) *Uint32OrHash {
	firstByte := data[0]
	if firstByte == 0 {
		return &Uint32OrHash{
			value: common.NewHash(data[1:]),
		}
	} else if firstByte == 1 {
		num := data[1:]
		if len(num) < 4 {
			num = common.AppendZeroes(num, 4)
		}
		return &Uint32OrHash{
			value: binary.LittleEndian.Uint32(num),
		}
	} else {
		return nil
	}
}

// Value returns the interface value.
func (x *Uint32OrHash) Value() interface{} {
	if x == nil {
		return nil
	}
	return x.value
}

// IsHash returns true if the value is a hash
func (x *Uint32OrHash) IsHash() bool {
	if x == nil {
		return false
	}
	_, is := x.value.(common.Hash)
	return is
}

// Hash returns the value as a common.Hash. It panics if the value is not a hash.
func (x *Uint32OrHash) Hash() common.Hash {
	if !x.IsHash() {
		panic("value is not common.Hash")
	}

	return x.value.(common.Hash)
}

func (x *Uint32OrHash) String() string {
	if x.IsHash() {
		return x.Hash().String()
	}

	return fmt.Sprintf("%d", x.value)
}

// IsUint32 returns true if the value is a uint32
func (x *Uint32OrHash) IsUint32() bool {
	if x == nil {
		return false
	}

	_, is := x.value.(uint32)
	return is
}

// Uint32 returns the value as a uint32. It panics if the value is not a uint32.
func (x *Uint32OrHash) Uint32() uint32 {
	if !x.IsUint32() {
		panic("value is not uint32")
	}

	return x.value.(uint32)
}

// Encode will encode a Uint32OrHash using SCALE
func (x *Uint32OrHash) Encode() ([]byte, error) {
	var encMsg []byte
	switch c := x.Value().(type) {
	case uint32:
		startingBlockByteArray := make([]byte, 4)
		binary.LittleEndian.PutUint32(startingBlockByteArray, c)
		encMsg = append(encMsg, append([]byte{1}, startingBlockByteArray...)...)
	case common.Hash:
		encMsg = append(encMsg, append([]byte{0}, c.ToBytes()...)...)
	}
	return encMsg, nil
}

// Decode decodes a value into a Uint32OrHash
func (x *Uint32OrHash) Decode(r io.Reader) error {
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
		num := make([]byte, 4)
		_, err = r.Read(num)
		if err != nil {
			return err
		}
		x.value = binary.LittleEndian.Uint32(num)
	}
	return nil
}
