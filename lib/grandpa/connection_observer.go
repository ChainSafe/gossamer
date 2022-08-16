package grandpa

import (
	"errors"
	"fmt"
	"sync"

	"github.com/libp2p/go-libp2p-core/peer"
)

var (
	ErrInvalidRound = errors.New("invalid round")
)

type view struct {
	Round              uint64
	SetID              uint64
	LastFinalizedBlock uint32
}

type peerViewTracker struct {
	mutex sync.RWMutex
	peers map[peer.ID]view
}

func (p *peerViewTracker) updatePeerView(who peer.ID, msg *NeighbourPacketV1) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	v, has := p.peers[who]
	if has && v.Round > msg.Round {
		return fmt.Errorf("%w: expecting a round greater or equal to %d got %d",
			ErrInvalidRound, v.Round, msg.Round)
	}

	p.peers[who] = view{
		Round:              msg.Round,
		SetID:              msg.SetID,
		LastFinalizedBlock: msg.Number,
	}

	return nil
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

// OnPeerConnected will send neighbour message
func (s *Service) OnPeerConnected(who peer.ID) {
	s.roundLock.Lock()
	neighbourMessage := &NeighbourPacketV1{
		Round:  s.state.round,
		SetID:  s.state.setID,
		Number: uint32(s.head.Number),
	}
	s.roundLock.Unlock()

	logger.Debugf("peer %s connected: sending neighbour message: %v",
		who, neighbourMessage)
	s.sendNeighbourMessageTo(who, neighbourMessage)
}

func (s *Service) OnPeerDisconnected(who peer.ID) {
	logger.Debugf("peer %s disconnected: removing its view", who)
	s.viewTracker.removePeerView(who)
}
