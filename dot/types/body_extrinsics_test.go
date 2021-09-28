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

package types

import (
	"fmt"
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/require"
)

func TestBodyExtrinsicsToSCALEEncodedBody(t *testing.T) {
	exts := []Extrinsic{{1, 2, 3}, {7, 8, 9, 0}, {0xa, 0xb}}

	bodyExtrinsicsBefore := NewBody(exts)
	scaleEncodedBody, err := bodyExtrinsicsBefore.AsSCALEEncodedBody()
	require.NoError(t, err)

	bodyExtrinsicsAfter, err := NewBodyFromBytes(scaleEncodedBody)
	require.NoError(t, err)

	require.Equal(t, bodyExtrinsicsBefore, bodyExtrinsicsAfter)
}

func TestHasExtrinsics(t *testing.T) {
	exts := []Extrinsic{{1, 2, 3}, {7, 8, 9, 0}, {0xa, 0xb}}

	bodyExtrinsics := NewBody(exts)

	found, err := bodyExtrinsics.HasExtrinsic(Extrinsic{1, 2, 3})
	require.NoError(t, err)
	require.True(t, found)
}

func TestBodyExtrinsicsFromEncodedBytes(t *testing.T) {
	exts := []Extrinsic{{1, 2, 3}, {7, 8, 9, 0}, {0xa, 0xb}}

	bodyExtrinsicsBefore := NewBody(exts)

	encodeExtrinsics, err := bodyExtrinsicsBefore.AsEncodedExtrinsics()
	require.NoError(t, err)

	encodedBytes := ExtrinsicsArrayToBytesArray(encodeExtrinsics)

	bodyExtrinsicsAfter, err := NewBodyFromEncodedBytes(encodedBytes)
	require.NoError(t, err)

	require.Equal(t, bodyExtrinsicsBefore, bodyExtrinsicsAfter)
}

func TestBodyExtrinsicsFromExtrinsicStrings(t *testing.T) {
	exts := []Extrinsic{{1, 2, 3}, {7, 8, 9, 0}, {0xa, 0xb}}
	extStrings := []string{}

	for _, ext := range exts {
		extStrings = append(extStrings, common.BytesToHex(ext))
	}

	fmt.Println(extStrings)

	bodyFromByteExtrinsics := NewBody(exts)
	bodyFromStringExtrinsics, err := NewBodyFromExtrinsicStrings(extStrings)
	require.NoError(t, err)

	require.Equal(t, bodyFromByteExtrinsics, bodyFromStringExtrinsics)
}
