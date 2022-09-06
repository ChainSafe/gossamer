package grandpa

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
)

var (
	ErrInvalidRound = errors.New("invalid round")
)

type view struct {
	latestNeighborSent         *NeighbourPacketV1
	PrevoteSent, PrecommitSent bool
	Round                      uint64
	SetID                      uint64
	LastFinalizedBlock         uint32
}

type peerViewTracker struct {
	mutex sync.RWMutex
	peers map[peer.ID]view
}

// trackPeer will push the peer in the map
func (p *peerViewTracker) trackPeer(who peer.ID) bool {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	_, has := p.peers[who]
	if has {
		return false
	}

	p.peers[who] = view{}
	return true
}

func (p *peerViewTracker) updatePeerView(who peer.ID, msg *NeighbourPacketV1) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	peerView, has := p.peers[who]
	if has && peerView.Round > msg.Round {
		return fmt.Errorf("%w: expecting a round greater or equal to %d got %d",
			ErrInvalidRound, peerView.Round, msg.Round)
	}

	p.peers[who] = view{
		Round:              msg.Round,
		SetID:              msg.SetID,
		LastFinalizedBlock: msg.Number,
	}

	return nil
}

// Watch is a function that runs until done channel triggers
// the main popuporse of this function is to send through the returned channel
// the slice of peer IDs who satisfy the given condition
func (p *peerViewTracker) Watch(interval time.Duration, condition func(view) bool, done <-chan struct{}) <-chan peer.IDSlice {
	peersCh := make(chan peer.IDSlice)

	go func() {
		defer close(peersCh)

		lookupTicker := time.NewTicker(interval)
		defer lookupTicker.Stop()

		for {
			p.mutex.RLock()
			if len(p.peers) > 0 {
				peers := make(peer.IDSlice, 0, len(p.peers))
				for peer, v := range p.peers {
					if condition(v) {
						peers = append(peers, peer)
					}
				}

				peersCh <- peers
			}

			p.mutex.RUnlock()

			lookupTicker.Reset(interval)
			select {
			case <-done:
				return
			case <-lookupTicker.C:
			}
		}
	}()

	return peersCh
}

// Retrieve returns all peers that view satisfy predicate
func (p *peerViewTracker) Retrieve(predicate func(view) bool) (peers peer.IDSlice) {
	peers = make(peer.IDSlice, 0, len(p.peers))

	p.mutex.RLock()
	defer p.mutex.RUnlock()

	for peer, v := range p.peers {
		if predicate(v) {
			peers = append(peers, peer)
		}
	}

	return peers
}

func (p *peerViewTracker) forgetMessages(peer peer.ID) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	view, has := p.peers[peer]
	if !has {
		return
	}

	view.latestNeighborSent = nil
	view.PrecommitSent = false
	view.PrevoteSent = false

	p.peers[peer] = view
}

func (p *peerViewTracker) markNeighborAsSent(peers peer.IDSlice, neighborMessage *NeighbourPacketV1) {
	if len(peers) == 0 {
		return
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	for _, peer := range peers {
		view, has := p.peers[peer]
		if !has {
			continue
		}

		view.latestNeighborSent = neighborMessage
		p.peers[peer] = view
	}
}

func (p *peerViewTracker) markAsSent(peers peer.IDSlice, stage Subround) {
	if len(peers) == 0 {
		return
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	for _, peer := range peers {
		view, has := p.peers[peer]
		if !has {
			continue
		}

		switch stage {
		case precommit:
			view.PrecommitSent = true
		case prevote:
			view.PrevoteSent = true
		}

		p.peers[peer] = view
	}
}

func (p *peerViewTracker) removePeerView(who peer.ID) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	delete(p.peers, who)
}

func newPeerViewTracker() *peerViewTracker {
	return &peerViewTracker{
		peers: make(map[peer.ID]view),
	}
}
