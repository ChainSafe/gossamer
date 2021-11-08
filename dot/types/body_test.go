// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package types

import (
	"fmt"
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
)

var exts = []Extrinsic{{1, 2, 3}, {7, 8, 9, 0}, {0xa, 0xb}}

func TestBodyToSCALEEncodedBody(t *testing.T) {
	bodyBefore := NewBody(exts)
	scaleEncodedBody, err := scale.Marshal(*bodyBefore)
	require.NoError(t, err)

	bodyAfter, err := NewBodyFromBytes(scaleEncodedBody)
	require.NoError(t, err)

	require.Equal(t, bodyBefore, bodyAfter)
}

func TestHasExtrinsics(t *testing.T) {
	body := NewBody(exts)

	found, err := body.HasExtrinsic(Extrinsic{1, 2, 3})
	require.NoError(t, err)
	require.True(t, found)
}

func TestBodyFromEncodedBytes(t *testing.T) {
	bodyBefore := NewBody(exts)

	encodeExtrinsics, err := bodyBefore.AsEncodedExtrinsics()
	require.NoError(t, err)

	encodedBytes := ExtrinsicsArrayToBytesArray(encodeExtrinsics)

	bodyAfter, err := NewBodyFromEncodedBytes(encodedBytes)
	require.NoError(t, err)

	require.Equal(t, bodyBefore, bodyAfter)
}

func TestBodyFromExtrinsicStrings(t *testing.T) {
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
