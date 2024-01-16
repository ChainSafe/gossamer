// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"math/big"
	"sync"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/libp2p/go-libp2p/core/peer"
	"golang.org/x/exp/maps"
)

type peerViewSet struct {
	mtx    sync.RWMutex
	view   map[peer.ID]peerView
	target uint
}

// getTarget takes the average of all peer views best number
func (p *peerViewSet) getTarget() uint {
	p.mtx.RLock()
	defer p.mtx.RUnlock()

	numbers := make([]uint, 0, len(p.view))
	// we are going to sort the data and remove the outliers then we will return the avg of all the valid elements
	for _, view := range maps.Values(p.view) {
		numbers = append(numbers, view.number)
	}

	sum, count := nonOutliersSumCount(numbers)
	quotientBigInt := uint(big.NewInt(0).Div(sum, big.NewInt(int64(count))).Uint64())

	if p.target >= quotientBigInt {
		return p.target
	}

	p.target = quotientBigInt // cache latest calculated target
	return p.target
}

func (p *peerViewSet) find(pID peer.ID) (view peerView, ok bool) {
	p.mtx.RLock()
	defer p.mtx.RUnlock()

	view, ok = p.view[pID]
	return view, ok
}

func (p *peerViewSet) size() int {
	p.mtx.RLock()
	defer p.mtx.RUnlock()

	return len(p.view)
}

func (p *peerViewSet) values() []peerView {
	p.mtx.RLock()
	defer p.mtx.RUnlock()

	return maps.Values(p.view)
}

func (p *peerViewSet) update(peerID peer.ID, hash common.Hash, number uint) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	newView := peerView{
		who:    peerID,
		hash:   hash,
		number: number,
	}

	view, ok := p.view[peerID]
	if !ok {
		p.view[peerID] = newView
		return
	}

	if view.number >= newView.number {
		return
	}

	p.view[peerID] = newView
}

func newPeerViewSet(cap int) *peerViewSet {
	return &peerViewSet{
		view: make(map[peer.ID]peerView, cap),
	}
}
