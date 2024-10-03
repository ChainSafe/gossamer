package grandpa

import (
	"fmt"
	"github.com/libp2p/go-libp2p/core/peer"
	"time"
)

type neighborState struct {
	setID uint64
	round uint64
	//highestFinalized uint32 not sure if i need this or not
}

type NeighborTracker struct {
	//grandpa *Service
	network Network

	peerview     map[peer.ID]neighborState
	currentSetID uint64
	currentRound uint64
	highestFinalized uint32
}

func NewNeighborTracker(network Network) *NeighborTracker {
	return &NeighborTracker{
		network: network,
		peerview: make(map[peer.ID]neighborState),
	}
}

func (nt *NeighborTracker) UpdateState(setID uint64, round uint64, highestFinalized uint32) {
	nt.currentSetID = setID
	nt.currentRound = round
	nt.highestFinalized = highestFinalized
}


func (nt *NeighborTracker) UpdatePeer(p peer.ID, setID uint64, round uint64) {
	peerState := neighborState{setID, round}
	nt.peerview[p] = peerState
}

func (nt *NeighborTracker) BroadcastNeighborMsg() error {
	for id, peerState := range nt.peerview {
		if !(peerState.round < nt.currentRound || peerState.setID < nt.currentSetID) {
			// Send msg
			packet := NeighbourPacketV1{
				Round:  nt.currentRound,
				SetID:  nt.currentSetID,
				Number: nt.highestFinalized,
			}

			cm, err := packet.ToConsensusMessage()
			if err != nil {
				return fmt.Errorf("converting NeighbourPacketV1 to network message: %w", err)
			}

			err = nt.network.SendMessage(id, cm)
			if err != nil {
				return fmt.Errorf("sending message to peer: %v", id)
			}
		}
	}
	return nil
}

func (nt *NeighborTracker) initCatchUp() {
	logger.Warnf("Initializing catch up")
	const duration = time.Second * 30
	ticker := time.NewTicker(duration)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			logger.Warnf("Ready to catch up again")

		//case <-c.shutdownCatchup:
		//	logger.Warnf("Closing catch up")
		//	return
		//}
	}
}

//type catchUp struct {
//	lock sync.Mutex
//
//	grandpa         *Service
//	readyToCatchUp  *atomic.Bool
//	shutdownCatchup chan struct{}
//}
//
//func newCatchUp(grandpa *Service) *catchUp {
//	c := &catchUp{
//		readyToCatchUp:  &atomic.Bool{},
//		grandpa:         grandpa,
//		shutdownCatchup: make(chan struct{}),
//	}
//	c.readyToCatchUp.Store(true)
//	return c
//}

//func (c *catchUp) tryCatchUp(round uint64, setID uint64, peer peer.ID) error {
//	logger.Warnf("Trying to catch up")
//	//c.lock.Lock()
//	if !c.readyToCatchUp.Load() {
//		// Fine we just skip
//		return nil
//	}
//	//c.lock.Lock()
//	catchUpRequest := newCatchUpRequest(round, setID)
//	cm, err := catchUpRequest.ToConsensusMessage()
//	if err != nil {
//		return fmt.Errorf("converting catchUpRequest to network message: %w", err)
//	}
//
//	logger.Warnf("sending catchup request message: %v", catchUpRequest)
//	err = c.grandpa.network.SendMessage(peer, cm)
//	if err != nil {
//		return fmt.Errorf("sending catchUpRequest to network message: %w", err)
//	}
//	c.readyToCatchUp.Store(false)
//	logger.Warnf("successfully tryed catch up")
//	return nil
//}
