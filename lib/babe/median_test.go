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

package babe

import (
	"math/big"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/stretchr/testify/require"
)

func TestMedian_OddLength(t *testing.T) {
	us := []uint64{3, 2, 1, 4, 5}
	res, err := median(us)
	require.NoError(t, err)

	var expected uint64 = 3
	require.Equal(t, expected, res)
}

func TestMedian_EvenLength(t *testing.T) {
	us := []uint64{1, 4, 2, 4, 5, 6}
	res, err := median(us)
	require.NoError(t, err)

	var expected uint64 = 4
	require.Equal(t, expected, res)
}

func TestSlotOffset_Failing(t *testing.T) {
	var st uint64 = 1000001
	var se uint64 = 1000000

	_, err := slotOffset(st, se)
	require.NotNil(t, err)
}

func TestSlotOffset(t *testing.T) {
	var st uint64 = 1000000
	var se uint64 = 1000001

	res, err := slotOffset(st, se)
	require.NoError(t, err)

	var expected uint64 = 1
	require.Equal(t, expected, res)
}

func addBlocksToState(t *testing.T, babeService *Service, depth int, blockState BlockState, startTime uint64) {
	previousHash := blockState.BestBlockHash()
	previousAT := startTime

	for i := 1; i <= depth; i++ {
		// create proof that we can authorize this block
		babeService.epochData.threshold = maxThreshold
		babeService.epochData.authorityIndex = 0
		slotNumber := uint64(i)

		outAndProof, err := babeService.runLottery(slotNumber)
		require.NoError(t, err)
		require.NotNil(t, outAndProof, "proof was nil when over threshold")

		babeService.slotToProof[slotNumber] = outAndProof

		// create pre-digest
		slot := Slot{
			start:    uint64(time.Now().Unix()),
			duration: uint64(1000),
			number:   slotNumber,
		}

		predigest, err := babeService.buildBlockPreDigest(slot)
		require.NoError(t, err)

		block := &types.Block{
			Header: &types.Header{
				ParentHash: previousHash,
				Number:     big.NewInt(int64(i)),
				Digest:     types.Digest{predigest},
			},
			Body: &types.Body{},
		}

		arrivalTime := previousAT + uint64(1)
		previousHash = block.Header.Hash()
		previousAT = arrivalTime

		err = blockState.AddBlockWithArrivalTime(block, arrivalTime)
		require.NoError(t, err)
	}
}

func TestSlotTime(t *testing.T) {
	babeService := createTestService(t, nil)
	addBlocksToState(t, babeService, 100, babeService.blockState, uint64(0))

	res, err := babeService.slotTime(103, 20)
	require.NoError(t, err)

	expected := uint64(129)
	require.Equal(t, expected, res)
}

func TestEstimateCurrentSlot(t *testing.T) {
	babeService := createTestService(t, nil)
	// create proof that we can authorize this block
	babeService.epochData.threshold = maxThreshold
	babeService.epochData.authorityIndex = 0
	slotNumber := uint64(17)

	outAndProof, err := babeService.runLottery(slotNumber)
	require.NoError(t, err)
	require.NotNil(t, outAndProof, "proof was nil when over threshold")

	babeService.slotToProof[slotNumber] = outAndProof

	// create pre-digest
	slot := Slot{
		start:    uint64(time.Now().Unix()),
		duration: babeService.slotDuration,
		number:   slotNumber,
	}

	predigest, err := babeService.buildBlockPreDigest(slot)
	require.NoError(t, err)

	block := &types.Block{
		Header: &types.Header{
			ParentHash: genesisHeader.Hash(),
			Number:     big.NewInt(int64(1)),
			Digest:     types.Digest{predigest},
		},
		Body: &types.Body{},
	}

	arrivalTime := uint64(time.Now().Unix()) - slot.duration

	err = babeService.blockState.AddBlockWithArrivalTime(block, arrivalTime)
	require.NoError(t, err)

	estimatedSlot, err := babeService.estimateCurrentSlot()
	require.NoError(t, err)
	require.Equal(t, slotNumber+1, estimatedSlot)
}

func TestGetCurrentSlot(t *testing.T) {
	babeService := createTestService(t, nil)

	// 100 blocks / 1000 ms/s
	addBlocksToState(t, babeService, 100, babeService.blockState, uint64(time.Now().Unix())-(babeService.slotDuration/10))

	res, err := babeService.getCurrentSlot()
	require.NoError(t, err)

	expected := uint64(162)

	if res != expected && res != expected+1 {
		t.Fatalf("Fail: got %d expected %d", res, expected)
	}
}
