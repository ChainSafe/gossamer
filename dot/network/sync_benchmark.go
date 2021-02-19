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

package network

import (
	"time"
)

type syncBenchmarker struct {
	start           time.Time
	startBlock      uint64
	blocksPerSecond []float64
	syncing         bool
}

func newSyncBenchmarker() *syncBenchmarker {
	return &syncBenchmarker{
		blocksPerSecond: []float64{},
	}
}

func (b *syncBenchmarker) begin(block uint64) {
	if b.syncing {
		return
	}

	b.start = time.Now()
	b.startBlock = block
	b.syncing = true
}

func (b *syncBenchmarker) end(block uint64) {
	if !b.syncing {
		return
	}

	duration := time.Since(b.start)
	blocks := block - b.startBlock
	if blocks == 0 {
		blocks = 1
	}
	bps := float64(blocks) / duration.Seconds()
	b.blocksPerSecond = append(b.blocksPerSecond, bps)
	b.syncing = false
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
