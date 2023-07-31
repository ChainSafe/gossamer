// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"container/ring"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_newSyncBenchmarker(t *testing.T) {
	t.Parallel()

	t.Run("10_samples_to_keep", func(t *testing.T) {
		t.Parallel()
		const samplesToKeep = 10
		actual := newSyncBenchmarker(samplesToKeep)

		expected := &syncBenchmarker{
			blocksPerSecond: ring.New(samplesToKeep),
			samplesToKeep:   samplesToKeep,
		}

		assert.Equal(t, expected, actual)
	})

	t.Run("panics_on_0_sample_to_keep", func(t *testing.T) {
		t.Parallel()
		const samplesToKeep = 0
		assert.PanicsWithValue(t, "cannot have 0 samples to keep", func() {
			newSyncBenchmarker(samplesToKeep)
		})
	})
}

func Test_syncBenchmarker_begin(t *testing.T) {
	t.Parallel()

	const startSec = 1000
	start := time.Unix(startSec, 0)
	const startBlock = 10

	b := syncBenchmarker{}
	b.begin(start, startBlock)

	expected := syncBenchmarker{
		start:      start,
		startBlock: startBlock,
	}

	assert.Equal(t, expected, b)
}

func Test_syncBenchmarker_end(t *testing.T) {
	t.Parallel()

	const startSec = 1000
	start := time.Unix(startSec, 0)

	const nowSec = 1010
	now := time.Unix(nowSec, 0)

	const (
		startBlock = 10
		endBlock   = 12
	)

	const ringCap = 3

	blocksPerSecond := ring.New(ringCap)
	blocksPerSecond.Value = 1.00
	blocksPerSecond = blocksPerSecond.Next()

	b := syncBenchmarker{
		start:           start,
		startBlock:      startBlock,
		blocksPerSecond: blocksPerSecond,
	}
	b.end(now, endBlock)

	expectedBlocksPerSecond := ring.New(ringCap)
	expectedBlocksPerSecond.Value = 1.00
	expectedBlocksPerSecond = expectedBlocksPerSecond.Next()
	expectedBlocksPerSecond.Value = 0.2
	expectedBlocksPerSecond = expectedBlocksPerSecond.Next()

	expected := syncBenchmarker{
		start:           start,
		startBlock:      startBlock,
		blocksPerSecond: expectedBlocksPerSecond,
	}

	assert.Equal(t, expected, b)
}

func Test_syncBenchmarker_average(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		values  []float64
		ringCap int
		average float64
	}{
		// zero size ring is not possible due to constructor check
		"empty_ring": {
			ringCap: 1,
		},
		"single_element_in_one-size_ring": {
			values:  []float64{1.1},
			ringCap: 1,
			average: 1.1,
		},
		"single_element_in_two-size_ring": {
			values:  []float64{1.1},
			ringCap: 2,
			average: 1.1,
		},
		"two_elements_in_two-size_ring": {
			values:  []float64{1.0, 2.0},
			ringCap: 2,
			average: 1.5,
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			blocksPerSecond := ring.New(testCase.ringCap)
			for _, value := range testCase.values {
				blocksPerSecond.Value = value
				blocksPerSecond = blocksPerSecond.Next()
			}

			benchmarker := syncBenchmarker{
				blocksPerSecond: blocksPerSecond,
				samplesToKeep:   testCase.ringCap,
			}

			avg := benchmarker.average()

			assert.Equal(t, testCase.average, avg)
		})
	}
}

func Test_syncBenchmarker_mostRecentAverage(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		values  []float64
		ringCap int
		average float64
	}{
		// zero size ring is not possible due to constructor check
		"empty_ring": {
			ringCap: 1,
		},
		"single_element_in_one-size_ring": {
			values:  []float64{1.1},
			ringCap: 1,
			average: 1.1,
		},
		"single_element_in_two-size_ring": {
			values:  []float64{1.1},
			ringCap: 2,
			average: 1.1,
		},
		"two_elements_in_two-size_ring": {
			values:  []float64{1.0, 2.0},
			ringCap: 2,
			average: 2.0,
		},
		"three_elements_in_two-size_ring": {
			values:  []float64{1.0, 2.0, 3.0},
			ringCap: 2,
			average: 3.0,
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			blocksPerSecond := ring.New(testCase.ringCap)
			for _, value := range testCase.values {
				blocksPerSecond.Value = value
				blocksPerSecond = blocksPerSecond.Next()
			}

			benchmarker := syncBenchmarker{
				blocksPerSecond: blocksPerSecond,
			}

			avg := benchmarker.mostRecentAverage()

			assert.Equal(t, testCase.average, avg)
		})
	}
}

func Test_syncBenchmarker(t *testing.T) {
	t.Parallel()

	const samplesToKeep = 5
	benchmarker := newSyncBenchmarker(samplesToKeep)

	const initialBlock = 10
	timeZero := time.Unix(0, 0)
	const timeIncrement = time.Second
	const baseBlocksIncrement uint = 1

	startTime := timeZero
	endTime := startTime.Add(timeIncrement)
	var block uint = initialBlock

	const samples = 10
	for i := 0; i < samples; i++ {
		benchmarker.begin(startTime, block)
		block += baseBlocksIncrement + uint(i)
		benchmarker.end(endTime, block)

		startTime = startTime.Add(timeIncrement)
		endTime = startTime.Add(timeIncrement)
	}

	avg := benchmarker.average()
	const expectedAvg = 8.0
	assert.Equal(t, expectedAvg, avg)

	mostRecentAvg := benchmarker.mostRecentAverage()
	const expectedMostRecentAvg = 10.0
	assert.Equal(t, expectedMostRecentAvg, mostRecentAvg)
}
