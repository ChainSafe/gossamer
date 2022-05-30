// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"container/ring"
	"time"
)

type syncBenchmarker struct {
	start           time.Time
	startBlock      uint
	blocksPerSecond *ring.Ring
	samplesToKeep   int
}

func newSyncBenchmarker(samplesToKeep int) *syncBenchmarker {
	if samplesToKeep == 0 {
		panic("cannot have 0 samples to keep")
	}

	return &syncBenchmarker{
		blocksPerSecond: ring.New(samplesToKeep),
		samplesToKeep:   samplesToKeep,
	}
}

func (b *syncBenchmarker) begin(now time.Time, block uint) {
	b.start = now
	b.startBlock = block
}

func (b *syncBenchmarker) end(now time.Time, block uint) {
	duration := now.Sub(b.start)
	blocks := block - b.startBlock
	bps := float64(blocks) / duration.Seconds()
	b.blocksPerSecond.Value = bps
	b.blocksPerSecond = b.blocksPerSecond.Next()
}

func (b *syncBenchmarker) average() float64 {
	var sum float64
	var elementsSet int
	b.blocksPerSecond.Do(func(x interface{}) {
		if x == nil {
			return
		}
		bps := x.(float64)
		sum += bps
		elementsSet++
	})

	if elementsSet == 0 {
		return 0
	}

	return sum / float64(elementsSet)
}

func (b *syncBenchmarker) mostRecentAverage() float64 {
	value := b.blocksPerSecond.Prev().Value
	if value == nil {
		return 0
	}
	return value.(float64)
}
