// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package types

import (
	"errors"
	"fmt"
)

var ErrNoFirstPreDigest = errors.New("first digest item is not pre-digest")

// RandomnessLength is the length of the epoch randomness (32 bytes)
const RandomnessLength = 32

// AllowedSlots tells in what ways a slot can be claimed.
type AllowedSlots byte

// https://github.com/paritytech/substrate/blob/ded44948e2d5a398abcb4e342b0513cb690961bb/primitives/consensus/babe/src/lib.rs#L219-L226
const (
	// PrimarySlots only allows primary slots.
	PrimarySlots AllowedSlots = iota
	// PrimaryAndSecondaryPlainSlots allow primary and secondary plain slots.
	PrimaryAndSecondaryPlainSlots
	// PrimaryAndSecondaryVRFSlots allows primary and secondary VRF slots.
	PrimaryAndSecondaryVRFSlots
)

var (
	ErrChainHeadMissingDigest = errors.New("chain head missing digest")
	ErrGenesisHeader          = errors.New("genesis header doesn't have a slot")
)

// BabeConfiguration contains the genesis data for BABE
// See https://github.com/paritytech/substrate/blob/426c26b8bddfcdbaf8d29f45b128e0864b57de1c/core/consensus/babe/primitives/src/lib.rs#L132
type BabeConfiguration struct {
	SlotDuration       uint64 // milliseconds
	EpochLength        uint64 // duration of epoch in slots
	C1                 uint64 // (1-(c1/c2)) is the probability of a slot being empty
	C2                 uint64
	GenesisAuthorities []AuthorityRaw
	Randomness         [RandomnessLength]byte
	SecondarySlots     byte
}

// BABEAuthorityRawToAuthority turns a slice of BABE AuthorityRaw into a slice of Authority
func BABEAuthorityRawToAuthority(adr []AuthorityRaw) ([]Authority, error) {
	ad := make([]Authority, len(adr))
	for i := range adr {
		ad[i] = Authority{}
		err := ad[i].FromRawSr25519(&adr[i])
		if err != nil {
			return nil, err
		}
	}

	return ad, nil
}

// EpochData is the data provided for a BABE epoch
type EpochData struct {
	Authorities []Authority
	Randomness  [RandomnessLength]byte
}

// ToEpochDataRaw returns the EpochData as an EpochDataRaw, converting the Authority to AuthorityRaw
func (d *EpochData) ToEpochDataRaw() *EpochDataRaw {
	raw := &EpochDataRaw{
		Randomness: d.Randomness,
	}

	rawAuths := make([]AuthorityRaw, len(d.Authorities))
	for i, auth := range d.Authorities {
		rawAuths[i] = *auth.ToRaw()
	}

	raw.Authorities = rawAuths
	return raw
}

// EpochDataRaw is the data provided for an epoch, with Authority as AuthorityRaw
type EpochDataRaw struct {
	Authorities []AuthorityRaw
	Randomness  [RandomnessLength]byte
}

// ToEpochData returns the EpochDataRaw as EpochData
func (d *EpochDataRaw) ToEpochData() (*EpochData, error) {
	epochData := &EpochData{
		Randomness: d.Randomness,
	}

	auths, err := BABEAuthorityRawToAuthority(d.Authorities)
	if err != nil {
		return nil, err
	}

	epochData.Authorities = auths
	return epochData, nil
}

// ConfigData represents a BABE configuration update
type ConfigData struct {
	C1             uint64
	C2             uint64
	SecondarySlots byte
}

// GetSlotFromHeader returns the BABE slot from the given header
func GetSlotFromHeader(header *Header) (uint64, error) {
	if header.Number == 0 {
		return 0, ErrGenesisHeader
	}

	if len(header.Digest) == 0 {
		return 0, ErrChainHeadMissingDigest
	}

	digestValue, err := header.Digest[0].Value()
	if err != nil {
		return 0, fmt.Errorf("getting first digest type value: %w", err)
	}
	preDigest, ok := digestValue.(PreRuntimeDigest)
	if !ok {
		return 0, fmt.Errorf("%w: got %T", ErrNoFirstPreDigest, digestValue)
	}

	digest, err := DecodeBabePreDigest(preDigest.Data)
	if err != nil {
		return 0, fmt.Errorf("cannot decode BabePreDigest from pre-digest: %s", err)
	}

	var slotNumber uint64
	switch d := digest.(type) {
	case BabePrimaryPreDigest:
		slotNumber = d.SlotNumber
	case BabeSecondaryVRFPreDigest:
		slotNumber = d.SlotNumber
	case BabeSecondaryPlainPreDigest:
		slotNumber = d.SlotNumber
	}

	return slotNumber, nil
}

// IsPrimary returns true if the block was authored in a primary slot, false otherwise.
func IsPrimary(header *Header) (bool, error) {
	if header == nil {
		return false, fmt.Errorf("cannot have nil header")
	}

	if len(header.Digest) == 0 {
		return false, ErrChainHeadMissingDigest
	}

	digestValue, err := header.Digest[0].Value()
	if err != nil {
		return false, fmt.Errorf("getting first digest type value: %w", err)
	}
	preDigest, ok := digestValue.(PreRuntimeDigest)
	if !ok {
		return false, fmt.Errorf("%w: got %T", ErrNoFirstPreDigest, digestValue)
	}

	digest, err := DecodeBabePreDigest(preDigest.Data)
	if err != nil {
		return false, fmt.Errorf("cannot decode BabePreDigest from pre-digest: %s", err)
	}

	switch digest.(type) {
	case BabePrimaryPreDigest:
		return true, nil
	default:
		return false, nil
	}
}
