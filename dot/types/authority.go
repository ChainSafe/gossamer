// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package types

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
)

// Authority struct to hold authority data
type Authority struct {
	Key crypto.PublicKey
	// Weight exists for potential improvements in the protocol and could
	// have a use-case in the future. In polkadot all authorities have the weight = 1.
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

// DeepCopy creates a deep copy of the Authority
func (a *Authority) DeepCopy() *Authority {
	pk := a.Key.Encode()
	pkCopy, _ := sr25519.NewPublicKey(pk)
	return &Authority{
		Key:    pkCopy,
		Weight: a.Weight,
	}
}

// FromRawSr25519 sets the Authority given AuthorityRaw. It converts the byte representations of
// the authority public keys into a sr25519.PublicKey.
func (a *Authority) FromRawSr25519(raw *AuthorityRaw) error {
	id, err := sr25519.NewPublicKey(raw.Key[:])
	if err != nil {
		return err
	}

	_ = id.Hex()

	a.Key = id
	a.Weight = raw.Weight
	return nil
}

// AuthorityRaw struct to hold raw authority data
type AuthorityRaw struct {
	Key    [sr25519.PublicKeyLength]byte
	Weight uint64
}

func (a *AuthorityRaw) String() string {
	return fmt.Sprintf("AuthorityRaw Key=0x%x Weight=%d", a.Key, a.Weight)
}

// AuthoritiesToRaw converts an array of Authority in an array of AuthorityRaw
func AuthoritiesToRaw(auths []Authority) []AuthorityRaw {
	raw := make([]AuthorityRaw, len(auths))
	for i, auth := range auths {
		raw[i] = *auth.ToRaw()
	}
	return raw
}

// AuthorityAsAddress represents an Authority with their address instead of public key
type AuthorityAsAddress struct {
	Address common.Address
	Weight  uint64
}

// AuthoritiesRawToAuthorityAsAddress converts an array of AuthorityRaws into an array of AuthorityAsAddress
func AuthoritiesRawToAuthorityAsAddress(authsRaw []AuthorityRaw, kt crypto.KeyType) ([]AuthorityAsAddress, error) {
	auths := make([]AuthorityAsAddress, len(authsRaw))
	for i, authRaw := range authsRaw {
		var pk crypto.PublicKey
		var err error
		switch kt {
		case crypto.Ed25519Type:
			pk, err = ed25519.NewPublicKey(authRaw.Key[:])
		case crypto.Sr25519Type:
			pk, err = sr25519.NewPublicKey(authRaw.Key[:])
		}
		if err != nil {
			return nil, err
		}
		auths[i] = AuthorityAsAddress{
			Address: crypto.PublicKeyToAddress(pk),
			Weight:  authRaw.Weight,
		}
	}
	return auths, nil
}
