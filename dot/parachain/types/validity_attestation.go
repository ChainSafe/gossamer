// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachaintypes

import (
	"fmt"

	"github.com/ChainSafe/gossamer/pkg/scale"
)

type ValidityAttestationValues interface {
	Implicit | Explicit
}

// ValidityAttestation is an implicit or explicit attestation to the validity of a parachain
// candidate.
type ValidityAttestation struct {
	inner any
}

func setValidityAttestation[Value ValidityAttestationValues](mvdt *ValidityAttestation, value Value) {
	mvdt.inner = value
}

func (mvdt *ValidityAttestation) SetValue(value any) (err error) {
	switch value := value.(type) {
	case Implicit:
		setValidityAttestation(mvdt, value)
		return

	case Explicit:
		setValidityAttestation(mvdt, value)
		return

	default:
		return fmt.Errorf("unsupported type")
	}
}

func (mvdt ValidityAttestation) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case Implicit:
		return 1, mvdt.inner, nil

	case Explicit:
		return 2, mvdt.inner, nil

	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (mvdt ValidityAttestation) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}

func (mvdt ValidityAttestation) ValueAt(index uint) (value any, err error) {
	switch index {
	case 1:
		return *new(Implicit), nil

	case 2:
		return *new(Explicit), nil

	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}

// Implicit is for Implicit attestation.
type Implicit ValidatorSignature

func (i Implicit) String() string { //skipcq:SCC-U1000
	return fmt.Sprintf("implicit(%s)", ValidatorSignature(i))
}

// Explicit is for Explicit attestation.
type Explicit ValidatorSignature //skipcq

func (e Explicit) String() string { //skipcq:SCC-U1000
	return fmt.Sprintf("explicit(%s)", ValidatorSignature(e))
}

// NewValidityAttestation creates a ValidityAttestation varying data type.
func NewValidityAttestation() ValidityAttestation {
	return ValidityAttestation{}
}
