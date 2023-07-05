// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package types

import (
	"bytes"
	"fmt"
	"math/big"
	"sort"

	"github.com/ChainSafe/gossamer/pkg/scale"
	"golang.org/x/exp/maps"
)

// InherentIdentifier is an identifier for an inherent.
type InherentIdentifier uint

const (
	// Timstap0 is the identifier for the `timestamp` inherent.
	Timstap0 InherentIdentifier = iota
	// Babeslot is the BABE inherent identifier.
	Babeslot
	// Uncles00 is the identifier for the `uncles` inherent.
	Uncles00
	// Parachn0 is an inherent key for parachains inherent.
	Parachn0
	// Newheads is an inherent key for new minimally-attested parachain heads.
	Newheads
)

// Bytes returns a byte array of given inherent identifier.
func (ii InherentIdentifier) Bytes() [8]byte {

	kb := [8]byte{}
	switch ii {
	case Timstap0:
		copy(kb[:], []byte("timstap0"))
	case Babeslot:
		copy(kb[:], []byte("babeslot"))
	case Uncles00:
		copy(kb[:], []byte("uncles00"))
	case Parachn0:
		copy(kb[:], []byte("parachn0"))
	case Newheads:
		copy(kb[:], []byte("newheads"))
	default:
		panic("invalid inherent identifier")
	}

	return kb
}

// InherentData contains a mapping of inherent keys to values
// keys must be 8 bytes, values are a scale-encoded byte array
type InherentData struct {
	Data map[[8]byte][]byte
}

// NewInherentData returns InherentData
func NewInherentData() *InherentData {
	return &InherentData{
		Data: make(map[[8]byte][]byte),
	}
}

func (d *InherentData) String() string {
	str := ""
	for k, v := range d.Data {
		str = str + fmt.Sprintf("key=%v\tvalue=%v\n", k, v)
	}
	return str
}

// SetInherent sets a inherent.
func (d *InherentData) SetInherent(inherentIdentifier InherentIdentifier, value any) error {
	data, err := scale.Marshal(value)
	if err != nil {
		return err
	}

	d.Data[inherentIdentifier.Bytes()] = data

	return nil
}

// Encode will encode a given []byte using scale.Encode
func (d *InherentData) Encode() ([]byte, error) {
	length := big.NewInt(int64(len(d.Data)))
	buffer := bytes.Buffer{}

	l, err := scale.Marshal(length)
	if err != nil {
		return nil, err
	}

	_, err = buffer.Write(l)
	if err != nil {
		return nil, err
	}

	keys := maps.Keys(d.Data)

	sort.Slice(keys, func(i, j int) bool {
		return bytes.Compare(keys[i][:], keys[j][:]) < 0
	})

	for _, key := range keys {
		v := d.Data[key]

		_, err = buffer.Write(key[:])
		if err != nil {
			return nil, err
		}

		venc, err := scale.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("scale encoding encoded value: %w", err)
		}
		_, err = buffer.Write(venc)
		if err != nil {
			return nil, err
		}
	}

	return buffer.Bytes(), nil
}
