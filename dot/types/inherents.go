// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package types

import (
	"fmt"

	"github.com/ChainSafe/gossamer/pkg/scale"
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

// InherentsData contains a mapping of inherent keys to values
// keys must be 8 bytes, values are a scale-encoded byte array
type InherentsData struct {
	Data map[[8]byte][]byte
}

// NewInherentsData returns InherentsData
func NewInherentsData() *InherentsData {
	return &InherentsData{
		Data: make(map[[8]byte]([]byte)),
	}
}

func (d *InherentsData) String() string {
	str := ""
	for k, v := range d.Data {
		str = str + fmt.Sprintf("key=%v\tvalue=%v\n", k, v)
	}
	return str
}

// SetInherent sets a inherent.
func (d *InherentsData) SetInherent(inherentIdentifier InherentIdentifier, value any) error {
	data, err := scale.Marshal(value)
	if err != nil {
		return err
	}

	d.Data[inherentIdentifier.Bytes()] = data

	return nil
}
