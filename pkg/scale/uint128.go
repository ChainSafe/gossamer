// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package scale

import (
	"encoding/binary"
	"fmt"
	"math/big"
)

// Uint128 represents an unsigned 128 bit integer
type Uint128 struct {
	Upper uint64
	Lower uint64
}

// MaxUint128 is the maximum uint128 value
var MaxUint128 = &Uint128{
	Upper: ^uint64(0),
	Lower: ^uint64(0),
}

// MustNewUint128 will panic if NewUint128 returns an error
func MustNewUint128(in interface{}, order ...binary.ByteOrder) (u *Uint128) {
	u, err := NewUint128(in, order...)
	if err != nil {
		panic(err)
	}
	return
}

func padBytes(b []byte, order binary.ByteOrder) []byte {
	for len(b) != 16 {
		switch order {
		case binary.BigEndian:
			b = append([]byte{0}, b...)
		case binary.LittleEndian:
			b = append(b, 0)
		}
	}
	return b
}

// NewUint128 is constructor for Uint128 that accepts an option binary.ByteOrder
// option is only used when inputted interface{} is of type []byte
// by default binary.LittleEndian is used for []byte since this is SCALE
func NewUint128(in interface{}, order ...binary.ByteOrder) (u *Uint128, err error) {
	switch in := in.(type) {
	case *big.Int:
		bytes := in.Bytes()
		if len(bytes) < 16 {
			bytes = padBytes(bytes, binary.BigEndian)
		}
		u = &Uint128{
			Upper: binary.BigEndian.Uint64(bytes[:8]),
			Lower: binary.BigEndian.Uint64(bytes[8:]),
		}
	case []byte:
		var o binary.ByteOrder = binary.LittleEndian
		if len(order) > 0 {
			o = order[0]
		}
		if len(in) < 16 {
			in = padBytes(in, o)
		}
		u = &Uint128{
			Upper: o.Uint64(in[8:]),
			Lower: o.Uint64(in[:8]),
		}
	default:
		err = fmt.Errorf("unsupported type: %T", in)
	}
	return
}

// Bytes returns the Uint128 in little endian format by default.  A variadic parameter
// order can be used to specify the binary.ByteOrder used
func (u *Uint128) Bytes(order ...binary.ByteOrder) (b []byte) {
	var o binary.ByteOrder = binary.LittleEndian
	if len(order) > 0 {
		o = order[0]
	}
	b = make([]byte, 16)
	switch o {
	case binary.LittleEndian:
		o.PutUint64(b[:8], u.Lower)
		o.PutUint64(b[8:], u.Upper)
		b = u.trimBytes(b, o)
	case binary.BigEndian:
		o.PutUint64(b[:8], u.Upper)
		o.PutUint64(b[8:], u.Lower)
		b = u.trimBytes(b, o)
	}
	return
}

// String returns the string format from the Uint128 value
func (u *Uint128) String() string {
	return fmt.Sprintf("%d", big.NewInt(0).SetBytes(u.Bytes()))
}

// Compare returns 1 if the receiver is greater than other, 0 if they are equal, and -1 otherwise.
func (u *Uint128) Compare(other *Uint128) int {
	switch {
	case u.Upper > other.Upper:
		return 1
	case u.Upper < other.Upper:
		return -1
	case u.Upper == other.Upper:
		switch {
		case u.Lower > other.Lower:
			return 1
		case u.Lower < other.Lower:
			return -1
		}
	}
	return 0
}

func (u *Uint128) trimBytes(b []byte, order binary.ByteOrder) []byte {
	switch order {
	case binary.LittleEndian:
		for {
			if len(b) == 0 {
				return b
			}
			if b[len(b)-1] == 0 {
				b = b[:len(b)-1]
			} else {
				break
			}
		}
	case binary.BigEndian:
		for {
			if len(b) == 0 {
				return b
			}
			if b[0] == 0 {
				b = b[1:]
			} else {
				break
			}
		}
	}
	return b
}

// UnmarshalJSON converts data to Uint128.
func (u *Uint128) UnmarshalJSON(data []byte) error {
	intVal, ok := big.NewInt(0).SetString(string(data), 10)
	if !ok {
		return fmt.Errorf("failed to unmarshal Uint128")
	}

	dec, err := NewUint128(intVal)
	if err != nil {
		return err
	}
	u.Upper = dec.Upper
	u.Lower = dec.Lower
	return nil
}
