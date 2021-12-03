// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package types

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"

	"github.com/ChainSafe/gossamer/pkg/scale"
)

var (
	// Timstap0 is an inherent key.
	Timstap0 = []byte("timstap0")
	// Babeslot is an inherent key.
	Babeslot = []byte("babeslot")
	// Uncles00 is an inherent key.
	Uncles00 = []byte("uncles00")
)

// InherentsData contains a mapping of inherent keys to values
// keys must be 8 bytes, values are a scale-encoded byte array
type InherentsData struct {
	data map[[8]byte]([]byte)
}

// NewInherentsData returns InherentsData
func NewInherentsData() *InherentsData {
	return &InherentsData{
		data: make(map[[8]byte]([]byte)),
	}
}

func (d *InherentsData) String() string {
	str := ""
	for k, v := range d.data {
		str = str + fmt.Sprintf("key=%v\tvalue=%v\n", k, v)
	}
	return str
}

// SetInt64Inherent set an inherent of type uint64
func (d *InherentsData) SetInt64Inherent(key []byte, data uint64) error {
	if len(key) != 8 {
		return errors.New("inherent key must be 8 bytes")
	}

	val := make([]byte, 8)
	binary.LittleEndian.PutUint64(val, data)

	venc, err := scale.Marshal(val)
	if err != nil {
		return err
	}

	kb := [8]byte{}
	copy(kb[:], key)

	d.data[kb] = venc
	return nil
}

// Encode will encode a given []byte using scale.Encode
func (d *InherentsData) Encode() ([]byte, error) {
	length := big.NewInt(int64(len(d.data)))
	buffer := bytes.Buffer{}

	l, err := scale.Marshal(length)
	if err != nil {
		return nil, err
	}

	_, err = buffer.Write(l[:])
	if err != nil {
		return nil, err
	}

	for k, v := range d.data {
		_, err = buffer.Write(k[:])
		if err != nil {
			return nil, err
		}
		_, err = buffer.Write(v)
		if err != nil {
			return nil, err
		}
	}
	return buffer.Bytes(), nil
}
