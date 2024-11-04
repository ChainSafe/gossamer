package parachaintypes

import (
	"reflect"
	"sort"

	"github.com/ChainSafe/gossamer/lib/common"
)

// A succinct representation of a peer's view. This consists of a bounded amount of chain heads
// and the highest known finalized block number.
//
// Up to `N` (5?) chain heads.
type View struct {
	// a bounded amount of chain Heads
	Heads []common.Hash
	// the highest known finalized number
	FinalizedNumber uint32
}

// Difference returns hashes present in v View but not in v2 View.
func (v View) Difference(v2 View) []common.Hash {

	vHeads := SortableHeads(v.Heads)
	v2Heads := SortableHeads(v2.Heads)

	sort.Sort(vHeads)
	sort.Sort(v2Heads)

	var diff []common.Hash
	i, j := 0, 0
	for i < len(vHeads) && j < len(v2Heads) {
		if vHeads[i] == v2Heads[j] {
			i++
			j++
		} else if vHeads[i].String() < v2Heads[j].String() {
			diff = append(diff, vHeads[i])
			i++
		} else {
			j++
		}
	}

	for i < len(vHeads) {
		diff = append(diff, vHeads[i])
		i++
	}

	return diff
}

// CheckHeadsEqual checks if the heads of the view are equal to the heads of the other view.
func (v View) CheckHeadsEqual(other View) bool {
	if len(v.Heads) != len(other.Heads) {
		return false
	}

	localHeads := v.Heads
	sort.Sort(SortableHeads(localHeads))
	otherHeads := other.Heads
	sort.Sort(SortableHeads(otherHeads))

	return reflect.DeepEqual(localHeads, otherHeads)
}

type SortableHeads []common.Hash

func (s SortableHeads) Len() int {
	return len(s)
}

func (s SortableHeads) Less(i, j int) bool {
	return s[i].String() > s[j].String()
}

func (s SortableHeads) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func ConstructView(liveHeads map[common.Hash]struct{}, finalizedNumber uint32) View {
	heads := make([]common.Hash, 0, len(liveHeads))
	for head := range liveHeads {
		heads = append(heads, head)
	}

	if len(heads) >= 5 {
		heads = heads[:5]
	}

	return View{
		Heads:           heads,
		FinalizedNumber: finalizedNumber,
	}
}
