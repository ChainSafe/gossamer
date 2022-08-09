// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package runtime

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ChainSafe/gossamer/pkg/scale"
)

// APIItem struct to hold runtime API Name and Version
type APIItem struct {
	Name [8]byte
	Ver  uint32
}

// Version is the runtime version info.
type Version struct {
	SpecName           []byte
	ImplName           []byte
	AuthoringVersion   uint32
	SpecVersion        uint32
	ImplVersion        uint32
	APIItems           []APIItem
	TransactionVersion uint32
	legacy             bool
}

type legacyData struct {
	SpecName         []byte
	ImplName         []byte
	AuthoringVersion uint32
	SpecVersion      uint32
	ImplVersion      uint32
	APIItems         []APIItem
}

// Encode returns the scale encoding of the version.
func (v *Version) Encode() (encoded []byte, err error) {
	if !v.legacy {
		return scale.Marshal(*v)
	}

	toEncode := legacyData{
		SpecName:         v.SpecName,
		ImplName:         v.ImplName,
		AuthoringVersion: v.AuthoringVersion,
		SpecVersion:      v.SpecVersion,
		ImplVersion:      v.ImplVersion,
		APIItems:         v.APIItems,
	}
	return scale.Marshal(toEncode)
}

var (
	ErrDecodingVersion       = errors.New("decoding version")
	ErrDecodingLegacyVersion = errors.New("decoding legacy version")
)

// DecodeVersion scale decodes the encoded version data and returns an error.
// It first tries to decode the data using the current version format.
// If that fails with an EOF error, it then tries to decode the data
// using the legacy version format (for Kusama).
func DecodeVersion(encoded []byte) (version Version, err error) {
	err = scale.Unmarshal(encoded, &version)
	if err == nil {
		return version, nil
	}

	if !strings.Contains(err.Error(), "EOF") {
		// TODO io.EOF should be wrapped in scale and
		// ErrDecodingVersion should be removed once this is done.
		return version, fmt.Errorf("%w: %s", ErrDecodingVersion, err)
	}

	// TODO: kusama seems to use the legacy version format
	var legacy legacyData
	err = scale.Unmarshal(encoded, &legacy)
	if err != nil {
		// TODO sentinel error should be wrapped in scale and
		// ErrDecodingLegacyVersion should be removed once this is done.
		return version, fmt.Errorf("%w: %s", ErrDecodingLegacyVersion, err)
	}

	return Version{
		SpecName:         legacy.SpecName,
		ImplName:         legacy.ImplName,
		AuthoringVersion: legacy.AuthoringVersion,
		SpecVersion:      legacy.SpecVersion,
		ImplVersion:      legacy.ImplVersion,
		APIItems:         legacy.APIItems,
		legacy:           true,
	}, nil
}

// WithLegacy sets the legacy boolean (for Kusama)
// and is only used for tests.
func (v Version) WithLegacy() Version {
	v.legacy = true //skipcq: RVV-B0006
	return v
}
