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
	"fmt"
	"math/big"
	"github.com/ChainSafe/gossamer/core/blocktree"
	log "github.com/ChainSafe/log15"
	"time"
)
// used to calculate slot time current value of 1200 from spec suggestion
const uint64 SlotTail = 1200

// calculate the slot time for a given block in miliseconds, returns nil if it can't be calculated
func (b *Session) slotTime(slot uint64, bt *blockTree) *time.Time {
	at = []uint64
	dl = bt.DeepestLeaf()
	bn = big.NewInt(SlotTail)
	bn.Sub(dl.number, bn)
	// check to make sure we have enough blocks before the deepest leaf to accurately calculate slot time
	if n.Cmp(0) <= 0 {
		log.Debug("Cannot calculate slot time, deepest leaf block number less than or equal to Slot Tail")
		return nil
	}
	s = bt.GetNodeFromBlockNumber(bn)
	sd = b.Config.SlotDuration
	for _, node in bt.SubChain(dl, s) {
		st = node.arrivalTime + (slotOffset(bt.computeSlotForBlock(node, sd), slot) * sd)
		at = append(st, at)
	}

	return median(at)

}

// will need to implement own quickselect because of library contraints, this will do for now
func median(l []uint64) uint64 {
	// sort the list
	sort.Slice(l, func(i, j int) bool { return l[i] < l[j] })

	m = len(l)
	if (m == 0) {
		log.Debug("arrival times list is empty!")
		return nil
	} else if (m % 2 == 0){
		median = (l[(m/2)-1] + l[(m/2)+1])/2
	} else {
		median = l[m/2]
	}


	

}

// returns slotOffset
func slotOffset(start uint64, end uint64) uint64 {
	return (end - start)
}

// computes the slot for a block from genesis
// helper for now, there's a better way to do this
func (bt *blockTree) computeSlotForBlock(n *node, sd uint64) uint64 {
	gt = bt.head.arrivalTime
	nt = n.arrivalTime
	
	sp = 0
	for gt < nt {
		gt += sd
		sp += 1
	}

	return sp
}