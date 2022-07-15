// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package runtime

import (
	"testing"

	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
)

func Test_VersionData_Scale(t *testing.T) {
	version := VersionData{
		SpecName:         []byte{1},
		ImplName:         []byte{2},
		AuthoringVersion: 3,
		SpecVersion:      4,
		ImplVersion:      5,
		APIItems: []APIItem{{
			Name: [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
			Ver:  6,
		}},
		TransactionVersion: 7,
	}

	expectedScale := []byte{
		0x4, 0x1, 0x4, 0x2, 0x3, 0x0, 0x0, 0x0, 0x4, 0x0, 0x0, 0x0,
		0x5, 0x0, 0x0, 0x0, 0x4, 0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7,
		0x8, 0x6, 0x0, 0x0, 0x0, 0x7, 0x0, 0x0, 0x0}

	scaleEncoded, err := scale.Marshal(version)
	require.NoError(t, err)
	require.Equal(t, expectedScale, scaleEncoded)

	encoding, err := version.Encode()
	require.NoError(t, err)
	require.Equal(t, expectedScale, encoding)

	var decoded VersionData
	err = scale.Unmarshal(scaleEncoded, &decoded)
	require.NoError(t, err)
	require.Equal(t, version, decoded)
}

func Test_LegacyVersionData_Scale(t *testing.T) {
	version := LegacyVersionData{
		SpecName:         []byte{1},
		ImplName:         []byte{2},
		AuthoringVersion: 3,
		SpecVersion:      4,
		ImplVersion:      5,
		APIItems: []APIItem{{
			Name: [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
			Ver:  6,
		}},
	}

	expectedScale := []byte{
		0x4, 0x1, 0x4, 0x2, 0x3, 0x0, 0x0, 0x0, 0x4, 0x0, 0x0, 0x0,
		0x5, 0x0, 0x0, 0x0, 0x4, 0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7,
		0x8, 0x6, 0x0, 0x0, 0x0}

	scaleEncoded, err := scale.Marshal(version)
	require.NoError(t, err)
	require.Equal(t, expectedScale, scaleEncoded)

	encoding, err := version.Encode()
	require.NoError(t, err)
	require.Equal(t, expectedScale, encoding)

	var decoded LegacyVersionData
	err = scale.Unmarshal(scaleEncoded, &decoded)
	require.NoError(t, err)
	require.Equal(t, version, decoded)
}
