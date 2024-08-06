package sync

import (
	"math/big"
	"sort"
	"sync"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/libp2p/go-libp2p/core/peer"
	"golang.org/x/exp/maps"
)

type peerView struct {
	bestBlockNumber uint32
	bestBlockHash   common.Hash
}

type peerViewSet struct {
	mtx    sync.RWMutex
	view   map[peer.ID]peerView
	target uint32
}

func (p *peerViewSet) get(peerID peer.ID) peerView {
	p.mtx.RLock()
	defer p.mtx.RUnlock()
	return p.view[peerID]
}

func (p *peerViewSet) update(peerID peer.ID, bestHash common.Hash, bestNumber uint32) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	newView := peerView{
		bestBlockHash:   bestHash,
		bestBlockNumber: bestNumber,
	}

	view, ok := p.view[peerID]
	if ok && view.bestBlockNumber >= newView.bestBlockNumber {
		return
	}

	p.view[peerID] = newView
}

// getTarget takes the average of all peer views best number
func (p *peerViewSet) getTarget() uint32 {
	p.mtx.RLock()
	defer p.mtx.RUnlock()

	if len(p.view) == 0 {
		return p.target
	}

	numbers := make([]uint32, 0, len(p.view))
	// we are going to sort the data and remove the outliers then we will return the avg of all the valid elements
	for _, view := range maps.Values(p.view) {
		numbers = append(numbers, view.bestBlockNumber)
	}

	sum, count := nonOutliersSumCount(numbers)
	quotientBigInt := uint32(big.NewInt(0).Div(sum, big.NewInt(int64(count))).Uint64())

	if p.target >= quotientBigInt {
		return p.target
	}

	p.target = quotientBigInt // cache latest calculated target
	return p.target
}

// nonOutliersSumCount calculates the sum and count of non-outlier elements
// Explanation:
// IQR outlier detection
// Q25 = 25th_percentile
// Q75 = 75th_percentile
// IQR = Q75 - Q25         // inter-quartile range
// If x >  Q75  + 1.5 * IQR or  x   < Q25 - 1.5 * IQR THEN  x is a mild outlier
// If x >  Q75  + 3.0 * IQR or  x   < Q25 – 3.0 * IQR THEN  x is a extreme outlier
// Ref: http://www.mathwords.com/o/outlier.htm
//
// returns: sum and count of all the non-outliers elements
func nonOutliersSumCount(dataArrUint []uint32) (sum *big.Int, count uint) {
	dataArr := make([]*big.Int, len(dataArrUint))
	for i, v := range dataArrUint {
		dataArr[i] = big.NewInt(int64(v))
	}

	length := len(dataArr)

	switch length {
	case 0:
		return big.NewInt(0), 0
	case 1:
		return dataArr[0], 1
	case 2:
		return big.NewInt(0).Add(dataArr[0], dataArr[1]), 2
	}

	sort.Slice(dataArr, func(i, j int) bool {
		return dataArr[i].Cmp(dataArr[j]) < 0
	})

	half := length / 2
	firstHalf := dataArr[:half]
	var secondHalf []*big.Int

	if length%2 == 0 {
		secondHalf = dataArr[half:]
	} else {
		secondHalf = dataArr[half+1:]
	}

	q1 := getMedian(firstHalf)
	q3 := getMedian(secondHalf)

	iqr := big.NewInt(0).Sub(q3, q1)
	iqr1_5 := big.NewInt(0).Mul(iqr, big.NewInt(2)) // instead of 1.5 it is 2.0 due to the rounding
	lower := big.NewInt(0).Sub(q1, iqr1_5)
	upper := big.NewInt(0).Add(q3, iqr1_5)

	sum = big.NewInt(0)
	for _, v := range dataArr {
		// collect valid (non-outlier) values
		lowPass := v.Cmp(lower)
		highPass := v.Cmp(upper)
		if lowPass >= 0 && highPass <= 0 {
			sum.Add(sum, v)
			count++
		}
	}

	return sum, count
}

func getMedian(data []*big.Int) *big.Int {
	length := len(data)
	half := length / 2
	if length%2 == 0 {
		sum := big.NewInt(0).Add(data[half], data[half-1])
		return sum.Div(sum, big.NewInt(2))
	}

	return data[half]
}
