// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"time"
)

type syncBenchmarker struct {
	start           time.Time
	startBlock      uint64
	blocksPerSecond []float64
	samplesToKeep   int
}

func newSyncBenchmarker(samplesToKeep int) *syncBenchmarker {
	return &syncBenchmarker{
		blocksPerSecond: make([]float64, 0, samplesToKeep),
		samplesToKeep:   samplesToKeep,
	}
}

func (b *syncBenchmarker) begin(now time.Time, block uint64) {
	b.start = now
	b.startBlock = block
}

func (b *syncBenchmarker) end(now time.Time, block uint64) {
	duration := now.Sub(b.start)
	blocks := block - b.startBlock
	bps := float64(blocks) / duration.Seconds()

	if len(b.blocksPerSecond) == b.samplesToKeep {
		b.blocksPerSecond = b.blocksPerSecond[1:]
	}

	b.blocksPerSecond = append(b.blocksPerSecond, bps)
}

func (b *syncBenchmarker) average() float64 {
	var sum float64
	for _, bps := range b.blocksPerSecond {
		sum += bps
	}
	return sum / float64(len(b.blocksPerSecond))
}

func (b *syncBenchmarker) mostRecentAverage() float64 {
	return b.blocksPerSecond[len(b.blocksPerSecond)-1]
}
