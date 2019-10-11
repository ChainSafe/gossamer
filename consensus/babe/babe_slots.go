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
	"sort"
	"github.com/ChainSafe/gossamer/core/blocktree"
	log "github.com/ChainSafe/log15"
)
// used to calculate slot time current value of 1200 from spec suggestion
const SlotTail uint64 = 1200

// calculate the slot time for a given block in miliseconds, returns 0 if it can't be calculated
func (b *Session) slotTime(slot uint64, bt *blocktree.BlockTree) uint64 {
	var at []uint64
	dl := bt.DeepestLeaf()
	bn := new(big.Int).SetUint64(SlotTail)
	bn.Sub(dl.Number, bn)
	// check to make sure we have enough blocks before the deepest leaf to accurately calculate slot time
	if bn.Cmp(new(big.Int)) <= 0 {
		log.Debug("Cannot calculate slot time, deepest leaf block Number less than or equal to Slot Tail")
		return 0
	}
	s := bt.GetNodeFromBlockNumber(bn)
	sd := b.config.SlotDuration
	for _, Node:= range(bt.SubChain(dl.Hash, s.Hash)) {
		st := Node.ArrivalTime + (slotOffset(bt.ComputeSlotForBlock(Node, sd), slot) * sd)
		at = append(at, st)
	}
	return median(at)

}

// will need to implement own quickselect because of library contr	aints, this will do for now
func median(l []uint64) uint64 {
	// sort the list
	sort.Slice(l, func(i, j int) bool { return l[i] < l[j] })

	m := len(l)
	med := uint64(0)
	if (m == 0) {
		log.Debug("arrival times list is empty!")
		return 0
	} else if (m % 2 == 0){
		med = (l[(m/2)-1] + l[(m/2)+1])/2
	} else {
		med = l[m/2]
	}
	return med
}

// returns slotOffset
func slotOffset(start uint64, end uint64) uint64 {
	return (end - start)
}
