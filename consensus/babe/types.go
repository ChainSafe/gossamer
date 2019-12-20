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

package babe

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math/big"

	scale "github.com/ChainSafe/gossamer/codec"
	"github.com/ChainSafe/gossamer/crypto/sr25519"
)

// BabeConfiguration contains the starting data needed for Babe
// see: https://github.com/paritytech/substrate/blob/426c26b8bddfcdbaf8d29f45b128e0864b57de1c/core/consensus/babe/primitives/src/lib.rs#L132
type BabeConfiguration struct {
	SlotDuration       uint64 // milliseconds
	EpochLength        uint64 // duration of epoch in slots
	C1                 uint64 // (1-(c1/c2)) is the probability of a slot being empty
	C2                 uint64
	GenesisAuthorities []AuthorityDataRaw
	Randomness         byte
	SecondarySlots     bool
}

type AuthorityDataRaw struct {
	Id     [32]byte
	Weight uint64
}

//nolint:structcheck
type AuthorityData struct {
	id     *sr25519.PublicKey
	weight uint64
}

var Timstap0 = []byte("timstap0")
var Babeslot = []byte("babeslot")

// InherentsData contains a mapping of inherent keys to values
// Keys must be 8 bytes, values are a variable-length byte array
type InherentsData struct {
	data map[[8]byte]([]byte)
}

func NewInherentsData() *InherentsData {
	return &InherentsData{
		data: make(map[[8]byte]([]byte)),
	}
}

func (d *InherentsData) SetInt64Inherent(key []byte, data uint64) error {
	if len(key) != 8 {
		return errors.New("inherent key must be 8 bytes")
	}

	val := make([]byte, 8)
	binary.LittleEndian.PutUint64(val, data)

	venc, err := scale.Encode(val)
	if err != nil {
		return err
	}

	kb := [8]byte{}
	copy(kb[:], key)

	d.data[kb] = venc
	return nil
}

func (d *InherentsData) Encode() ([]byte, error) {
	length := big.NewInt(int64(len(d.data)))

	buffer := bytes.Buffer{}
	se := scale.Encoder{Writer: &buffer}

	_, err := se.Encode(length)
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

type Slot struct {
	start    uint64
	duration uint64
	number   uint64
}
