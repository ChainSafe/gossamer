// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

//go:build integration
// +build integration

package babe

import (
	"testing"

	"github.com/stretchr/testify/require"
)
 
func TestRunLottery(t *testing.T) {
	babeService := createTestService(t, nil)

	babeService.epochData.threshold = maxThreshold

	outAndProof, err := babeService.runLottery(0, testEpochIndex)
	require.NoError(t, err)
	require.NotNil(t, outAndProof)
}

func TestRunLottery_False(t *testing.T) {
	babeService := createTestService(t, nil)
	babeService.epochData.threshold = minThreshold

	outAndProof, err := babeService.runLottery(0, testEpochIndex)
	require.NoError(t, err)
	require.Nil(t, outAndProof)
}

func TestCalculateThreshold_Failing(t *testing.T) {
	var C1 uint64 = 5
	var C2 uint64 = 4

	_, err := CalculateThreshold(C1, C2, 3)
	require.NotNil(t, err)
}
