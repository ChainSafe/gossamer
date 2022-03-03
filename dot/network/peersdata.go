// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"sync"

	"github.com/libp2p/go-libp2p-core/peer"
)

type peersData struct {
	mutexesMu  sync.RWMutex
	mutexes    map[peer.ID]*sync.Mutex
	inboundMu  sync.RWMutex
	inbound    map[peer.ID]*handshakeData
	outboundMu sync.RWMutex
	outbound   map[peer.ID]*handshakeData
}

func newPeersData() *peersData {
	return &peersData{
		mutexes:  make(map[peer.ID]*sync.Mutex),
		inbound:  make(map[peer.ID]*handshakeData),
		outbound: make(map[peer.ID]*handshakeData),
	}
}

func (p *peersData) setMutex(peerID peer.ID) {
	p.mutexesMu.Lock()
	defer p.mutexesMu.Unlock()
	p.mutexes[peerID] = new(sync.Mutex)
}

func (p *peersData) getMutex(peerID peer.ID) *sync.Mutex {
	p.mutexesMu.RLock()
	defer p.mutexesMu.RUnlock()
	return p.mutexes[peerID]
}

func (p *peersData) deleteMutex(peerID peer.ID) {
	p.mutexesMu.Lock()
	defer p.mutexesMu.Unlock()
	delete(p.mutexes, peerID)
}

func (p *peersData) getInbound(peerID peer.ID) (data *handshakeData) {
	p.inboundMu.RLock()
	defer p.inboundMu.RUnlock()
	return p.inbound[peerID]
}

func (p *peersData) setInbound(peerID peer.ID, data *handshakeData) {
	p.inboundMu.Lock()
	defer p.inboundMu.Unlock()
	p.inbound[peerID] = data
}

func (p *peersData) deleteInbound(peerID peer.ID) {
	p.inboundMu.Lock()
	defer p.inboundMu.Unlock()
	delete(p.inbound, peerID)
}

func (p *peersData) countInboundStreams() (count int64) {
	p.inboundMu.RLock()
	defer p.inboundMu.RUnlock()
	for _, data := range p.inbound {
		if data.stream != nil {
			count++
		}
	}
	return count
}

func (p *peersData) getOutbound(peerID peer.ID) (data *handshakeData) {
	p.outboundMu.RLock()
	defer p.outboundMu.RUnlock()
	return p.outbound[peerID]
}

func (p *peersData) setOutbound(peerID peer.ID, data *handshakeData) {
	p.outboundMu.Lock()
	defer p.outboundMu.Unlock()
	p.outbound[peerID] = data
}

func (p *peersData) deleteOutbound(peerID peer.ID) {
	p.outboundMu.Lock()
	defer p.outboundMu.Unlock()
	delete(p.outbound, peerID)
}

func (p *peersData) countOutboundStreams() (count int64) {
	p.outboundMu.RLock()
	defer p.outboundMu.RUnlock()
	for _, data := range p.outbound {
		if data.stream != nil {
			count++
		}
	}
	return count
}
