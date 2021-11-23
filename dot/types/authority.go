// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package types

import (
	"encoding/binary"
	"io"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
)

// Authority struct to hold authority data
type Authority struct {
	Key    crypto.PublicKey
	Weight uint64
}

// NewAuthority function to create Authority object
func NewAuthority(pub crypto.PublicKey, weight uint64) *Authority {
	return &Authority{
		Key:    pub,
		Weight: weight,
	}
}

// Encode returns the SCALE encoding of the BABEAuthorities.
func (a *Authority) Encode() ([]byte, error) {
	raw := a.ToRaw()

	enc := raw.Key[:]

	weightBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(weightBytes, raw.Weight)

	return append(enc, weightBytes...), nil
}

// Decode sets the
func (a *Authority) Decode(r io.Reader) error {
	return a.DecodeSr25519(r)
}

// DecodeSr25519 sets the Authority to the SCALE decoded input for Authority containing SR25519 Keys.
func (a *Authority) DecodeSr25519(r io.Reader) error {
	id, err := common.Read32Bytes(r)
	if err != nil {
		return err
	}

	weight, err := common.ReadUint64(r)
	if err != nil {
		return err
	}

	raw := &AuthorityRaw{
		Key:    id,
		Weight: weight,
	}

	return a.FromRawSr25519(raw)
}

// ToRaw returns the BABEAuthorities as BABEAuthoritiesRaw. It encodes the authority public keys.
func (a *Authority) ToRaw() *AuthorityRaw {
	raw := new(AuthorityRaw)

	id := a.Key.Encode()
	copy(raw.Key[:], id)

	raw.Weight = a.Weight
	return raw
}

// FromRawSr25519 sets the Authority given AuthorityRaw. It converts the byte representations of
// the authority public keys into a sr25519.PublicKey.
func (a *Authority) FromRawSr25519(raw *AuthorityRaw) error {
	id, err := sr25519.NewPublicKey(raw.Key[:])
	if err != nil {
		return err
	}

	a.Key = id
	a.Weight = raw.Weight
	return nil
}

// AuthorityRaw struct to hold raw authority data
type AuthorityRaw struct {
	Key    [sr25519.PublicKeyLength]byte
	Weight uint64
}

// AuthoritiesToRaw converts an array of Authority in an array of AuthorityRaw
func AuthoritiesToRaw(auths []Authority) []AuthorityRaw {
	raw := make([]AuthorityRaw, len(auths))
	for i, auth := range auths {
		raw[i] = *auth.ToRaw()
	}
	return raw
}
