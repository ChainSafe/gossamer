// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package crypto

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/ChainSafe/gossamer/internal/primitives/core/hashing"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// The root phrase for our publicly known keys.
const DevPhrase = "bottom drive obey lake curtain smoke basket hold race lonely fit walk"

// A since derivation junction description. It is the single parameter used when creating
// a new secret key from an existing secret key and, in the case of `SoftRaw` and `SoftIndex`
// a new public key from an existing public key.
type DeriveJunction struct {
	inner any
}
type DeriveJunctions interface {
	DeriveJunctionSoft | DeriveJunctionHard
}

func (dj DeriveJunction) Value() any {
	if dj.inner == nil {
		panic("nil inner for DeriveJunction")
	}
	return dj.inner
}

// Soft (vanilla) derivation. Public keys have a correspondent derivation.
type DeriveJunctionSoft [32]byte

// Hard ("hardened") derivation. Public keys do not have a correspondent derivation.
type DeriveJunctionHard [32]byte

// Consume self to return a hard derive junction with the same chain code.
func (dj *DeriveJunction) Harden() DeriveJunction {
	switch inner := dj.inner.(type) {
	case DeriveJunctionSoft:
		dj.inner = DeriveJunctionHard(inner)
	}
	return *dj
}

// Create a new soft (vanilla) DeriveJunction from a given, encodable, value.
func NewDeriveJunctionSoft(index any) (DeriveJunctionSoft, error) {
	var cc = [32]byte{}
	data, err := scale.Marshal(index)
	if err != nil {
		return DeriveJunctionSoft{}, err
	}

	if len(data) > 32 {
		cc = hashing.Blake2_256(data)
	} else {
		copy(cc[:], data)
	}
	return DeriveJunctionSoft(cc), nil
}

func NewDeriveJunctionFromString(j string) DeriveJunction {
	hard := false
	trimmed := strings.TrimPrefix(j, "/")
	if trimmed != j {
		hard = true
	}
	code := trimmed

	var res DeriveJunction
	n, err := strconv.Atoi(code)
	if err == nil {
		soft, err := NewDeriveJunctionSoft(n)
		if err != nil {
			panic(err)
		}
		res = DeriveJunction{
			inner: soft,
		}
	} else {
		soft, err := NewDeriveJunctionSoft(code)
		if err != nil {
			panic(err)
		}
		res = DeriveJunction{
			inner: soft,
		}
	}

	if hard {
		return res.Harden()
	} else {
		return res
	}
}

func NewDeriveJunction[V DeriveJunctions](value V) DeriveJunction {
	return DeriveJunction{
		inner: value,
	}
}

var secretPhraseRegex = regexp.MustCompile(`^(?P<phrase>[\d\w ]+)?(?P<path>(//?[^/]+)*)(///(?P<password>.*))?$`)

var junctionRegex = regexp.MustCompile(`/(/?[^/]+)`)

// Trait used for types that are really just a fixed-length array.
type ByteArray interface {
	// Return a `Vec<u8>` filled with raw data.
	ToRawVec() []byte
}

// Trait suitable for typical cryptographic key public type.
type Public[Signature any] interface {
	ByteArray

	// Verify a signature on a message. Returns true if the signature is good.
	Verify(sig Signature, message []byte) bool
}

// A secret uri (`SURI`) that can be used to generate a key pair.
//
// The `SURI` can be parsed from a string. The string is interpreted in the following way:
//
// - If `string` is a possibly `0x` prefixed 64-digit hex string, then it will be interpreted
// directly as a `MiniSecretKey` (aka "seed" in `subkey`).
// - If `string` is a valid BIP-39 key phrase of 12, 15, 18, 21 or 24 words, then the key will
// be derived from it. In this case:
//   - the phrase may be followed by one or more items delimited by `/` characters.
//   - the path may be followed by `///`, in which case everything after the `///` is treated
//
// as a password.
//   - If `string` begins with a `/` character it is prefixed with the Substrate public `DEV_PHRASE`
//     and interpreted as above.
//
// In this case they are interpreted as HDKD junctions; purely numeric items are interpreted as
// integers, non-numeric items as strings. Junctions prefixed with `/` are interpreted as soft
// junctions, and with `//` as hard junctions.
//
// There is no correspondence mapping between `SURI` strings and the keys they represent.
// Two different non-identical strings can actually lead to the same secret being derived.
// Notably, integer junction indices may be legally prefixed with arbitrary number of zeros.
// Similarly an empty password (ending the `SURI` with `///`) is perfectly valid and will
// generally be equivalent to no password at all.
//
// # Example
//
// Parse [`DEV_PHRASE`] secret uri with junction:
//
// ```
// # use sp_core::crypto::{SecretUri, DeriveJunction, DEV_PHRASE, ExposeSecret};
// # use std::str::FromStr;
// let suri = SecretUri::from_str("//Alice").expect("Parse SURI");
//
// assert_eq!(vec![DeriveJunction::from("Alice").harden()], suri.junctions);
// assert_eq!(DEV_PHRASE, suri.phrase.expose_secret());
// assert!(suri.password.is_none());
// ```
//
// Parse [`DEV_PHRASE`] secret ui with junction and password:
//
// ```
// # use sp_core::crypto::{SecretUri, DeriveJunction, DEV_PHRASE, ExposeSecret};
// # use std::str::FromStr;
// let suri = SecretUri::from_str("//Alice///SECRET_PASSWORD").expect("Parse SURI");
//
// assert_eq!(vec![DeriveJunction::from("Alice").harden()], suri.junctions);
// assert_eq!(DEV_PHRASE, suri.phrase.expose_secret());
// assert_eq!("SECRET_PASSWORD", suri.password.unwrap().expose_secret());
// ```
//
// Parse [`DEV_PHRASE`] secret ui with hex phrase and junction:
//
// ```
// # use sp_core::crypto::{SecretUri, DeriveJunction, DEV_PHRASE, ExposeSecret};
// # use std::str::FromStr;
// let suri = SecretUri::from_str("0xe5be9a5092b81bca64be81d212e7f2f9eba183bb7a90954f7b76361f6edb5c0a//Alice").expect("Parse SURI");
//
// assert_eq!(vec![DeriveJunction::from("Alice").harden()], suri.junctions);
// assert_eq!("0xe5be9a5092b81bca64be81d212e7f2f9eba183bb7a90954f7b76361f6edb5c0a", suri.phrase.expose_secret());
// assert!(suri.password.is_none());
// ```
type SecretURI struct {
	// The phrase to derive the private key.
	// This can either be a 64-bit hex string or a BIP-39 key phrase.
	Phrase string
	// Optional password as given as part of the uri.
	Password *string
	// The junctions as part of the uri.
	Junctions []DeriveJunction
}

func NewSecretURI(s string) (SecretURI, error) {
	matches := secretPhraseRegex.FindStringSubmatch(s)
	if matches == nil {
		return SecretURI{}, fmt.Errorf("invalid format")
	}

	var (
		junctions []DeriveJunction
		phrase    = DevPhrase
		password  *string
	)
	for i, name := range secretPhraseRegex.SubexpNames() {
		if i == 0 {
			continue
		}
		switch name {
		case "path":
			junctionMatches := junctionRegex.FindAllString(matches[i], -1)
			for _, jm := range junctionMatches {
				junctions = append(junctions, NewDeriveJunctionFromString(jm))
			}
		case "phrase":
			if matches[i] != "" {
				phrase = matches[i]
			}
		case "password":
			if matches[i] != "" {
				pw := matches[i]
				password = &pw
			}
		}
	}
	return SecretURI{
		Phrase:    phrase,
		Password:  password,
		Junctions: junctions,
	}, nil
}

// Trait suitable for typical cryptographic PKI key pair type.
//
// For now it just specifies how to create a key from a phrase and derivation path.
type Pair[Seed, Signature any] interface {
	// Derive a child key from a series of given junctions.
	Derive(path []DeriveJunction, seed *Seed) (Pair[Seed, Signature], Seed, error)

	// Sign a message.
	Sign(message []byte) Signature

	// Get the public key.
	Public() Public[Signature]
}
