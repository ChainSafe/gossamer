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

	"github.com/ChainSafe/gossamer/dot/state"
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

func addBlocksToState(t *testing.T, babeService *Service, depth int, blockState BlockState, startTime time.Time) {
	previousHash := blockState.BestBlockHash()
	previousAT := startTime
	duration, err := time.ParseDuration("1s")
	require.NoError(t, err)

	for i := 1; i <= depth; i++ {
		// create proof that we can authorize this block
		babeService.epochData.threshold = maxThreshold
		babeService.epochData.authorityIndex = 0
		slotNumber := uint64(i)

		outAndProof, err := babeService.runLottery(slotNumber, testEpochIndex)
		require.NoError(t, err)
		require.NotNil(t, outAndProof, "proof was nil when over threshold")

		babeService.slotToProof[slotNumber] = outAndProof

		// create pre-digest
		slot := Slot{
			start:    time.Now(),
			duration: duration,
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

		arrivalTime := previousAT.Add(duration)
		previousHash = block.Header.Hash()
		previousAT = arrivalTime

		err = blockState.(*state.BlockState).AddBlockWithArrivalTime(block, arrivalTime)
		require.NoError(t, err)
	}
}

func TestSlotTime(t *testing.T) {
	babeService := createTestService(t, nil)
	addBlocksToState(t, babeService, 100, babeService.blockState, time.Now())

	res, err := babeService.slotTime(103, 20)
	require.NoError(t, err)

	dur, err := time.ParseDuration("127s")
	require.NoError(t, err)

	expected := time.Now().Add(dur)
	if int64(res) > expected.Unix()+3 && int64(res) < expected.Unix()-3 {
		t.Fatalf("Fail: got %d expected %d", res, expected.Unix())
	}
}

func TestEstimateCurrentSlot(t *testing.T) {
	babeService := createTestService(t, nil)
	// create proof that we can authorize this block
	babeService.epochData.threshold = maxThreshold
	babeService.epochData.authorityIndex = 0
	slotNumber := uint64(17)

	outAndProof, err := babeService.runLottery(slotNumber, testEpochIndex)
	require.NoError(t, err)
	require.NotNil(t, outAndProof, "proof was nil when over threshold")

	babeService.slotToProof[slotNumber] = outAndProof

	// create pre-digest
	slot := Slot{
		start:    time.Now(),
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

	arrivalTime := time.Now().UnixNano() - slot.duration.Nanoseconds()

	err = babeService.blockState.(*state.BlockState).AddBlockWithArrivalTime(block, time.Unix(0, arrivalTime))
	require.NoError(t, err)

	estimatedSlot, err := babeService.estimateCurrentSlot()
	require.NoError(t, err)
	if estimatedSlot > slotNumber+2 && estimatedSlot < slotNumber-2 {
		t.Fatalf("Fail: got %d expected %d", estimatedSlot, slotNumber)
	}
}

func TestGetCurrentSlot(t *testing.T) {
	babeService := createTestService(t, nil)

	before, err := time.ParseDuration("300s")
	require.NoError(t, err)
	beforeSecs := time.Now().Unix() - int64(before.Seconds())

	addBlocksToState(t, babeService, 100, babeService.blockState, time.Unix(beforeSecs, 0))

	res, err := babeService.getCurrentSlot()
	require.NoError(t, err)

	expected := uint64(167)

	if res > expected+2 && res < expected-2 {
		t.Fatalf("Fail: got %d expected %d", res, expected)
	}
}
