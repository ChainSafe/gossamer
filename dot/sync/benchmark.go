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
}

func newSyncBenchmarker() *syncBenchmarker {
	return &syncBenchmarker{
		blocksPerSecond: []float64{},
	}
}

func (b *syncBenchmarker) begin(block uint64) {
	b.start = time.Now()
	b.startBlock = block
}

func (b *syncBenchmarker) end(block uint64) {
	duration := time.Since(b.start)
	blocks := block - b.startBlock
	if blocks == 0 {
		blocks = 1
	}
	bps := float64(blocks) / duration.Seconds()
	b.blocksPerSecond = append(b.blocksPerSecond, bps)
}

func (b *syncBenchmarker) average() float64 {
	sum := float64(0)
	for _, bps := range b.blocksPerSecond {
		sum += bps
	}
	return sum / float64(len(b.blocksPerSecond))
}

func (b *syncBenchmarker) mostRecentAverage() float64 {
	return b.blocksPerSecond[len(b.blocksPerSecond)-1]
}
