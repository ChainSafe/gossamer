package grandpa

import (
	"fmt"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/libp2p/go-libp2p/core/peer"
	"time"
)

// NeighborBroadcastPeriod See https://github.com/paritytech/polkadot-sdk/blob/08498f5473351c3d2f8eacbe1bfd7bc6d3a2ef8d/substrate/client/consensus/grandpa/src/communication/mod.rs#L73
const NeighborBroadcastPeriod = time.Minute * 2

type neighborData struct {
	peer        peer.ID
	neighborMsg *NeighbourPacketV1
}

type neighborState struct {
	setID            uint64
	round            uint64
	highestFinalized uint32
}

type NeighborTracker struct {
	grandpa *Service

	peerview         map[peer.ID]neighborState
	currentSetID     uint64
	currentRound     uint64
	highestFinalized uint32

	finalizationCha chan *types.FinalisationInfo
	neighborMsgChan chan neighborData
	stoppedNeighbor chan struct{}
}

func NewNeighborTracker(grandpa *Service, neighborChan chan neighborData) *NeighborTracker {
	return &NeighborTracker{
		grandpa:         grandpa,
		peerview:        make(map[peer.ID]neighborState),
		finalizationCha: grandpa.blockState.GetFinalisedNotifierChannel(),
		neighborMsgChan: neighborChan,
		stoppedNeighbor: make(chan struct{}),
	}
}

func (nt *NeighborTracker) Start() {
	go nt.run()
}

func (nt *NeighborTracker) Stop() {
	nt.grandpa.blockState.FreeFinalisedNotifierChannel(nt.finalizationCha)
	close(nt.stoppedNeighbor)
}

func (nt *NeighborTracker) run() {
	logger.Info("starting neighbor tracker")
	ticker := time.NewTicker(NeighborBroadcastPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			logger.Debugf("neighbor message broadcast triggered by ticker")
			err := nt.BroadcastNeighborMsg()
			if err != nil {
				logger.Errorf("broadcasting neighbor message: %v", err)
			}

		case block := <-nt.finalizationCha:
			if block != nil {
				err := nt.UpdateState(block.SetID, block.Round, uint32(block.Header.Number))
				if err != nil {
					logger.Errorf("updating neighbor state: %v", err)
				}
				err = nt.BroadcastNeighborMsg()
				if err != nil {
					logger.Errorf("broadcasting neighbor message: %v", err)
				}
				ticker.Reset(NeighborBroadcastPeriod)
			}
		case neighborData := <-nt.neighborMsgChan:
			if neighborData.neighborMsg.Number > nt.peerview[neighborData.peer].highestFinalized {
				err := nt.UpdatePeer(neighborData.peer, neighborData.neighborMsg.SetID, neighborData.neighborMsg.Round, neighborData.neighborMsg.Number)
				if err != nil {
					logger.Errorf("updating neighbor: %v", err)
				}
			}
		case <-nt.stoppedNeighbor:
			logger.Info("stopping neighbor tracker")
			return
		}
	}
}

func (nt *NeighborTracker) UpdateState(setID uint64, round uint64, highestFinalized uint32) error {
	if nt == nil {
		return fmt.Errorf("neighbor tracker is nil")
	}
	nt.currentSetID = setID
	nt.currentRound = round
	nt.highestFinalized = highestFinalized
	return nil
}

func (nt *NeighborTracker) UpdatePeer(p peer.ID, setID uint64, round uint64, highestFinalized uint32) error {
	if nt == nil {
		return fmt.Errorf("neighbor tracker is nil")
	}
	if nt.peerview == nil {
		return fmt.Errorf("neighbour tracker has nil peer tracker")
	}
	peerState := neighborState{setID, round, highestFinalized}
	nt.peerview[p] = peerState
	return nil
}

func (nt *NeighborTracker) BroadcastNeighborMsg() error {
	logger.Warnf("braodcasting neighbor message to relevant peers")
	packet := NeighbourPacketV1{
		Round:  nt.currentRound,
		SetID:  nt.currentSetID,
		Number: nt.highestFinalized,
	}

	cm, err := packet.ToConsensusMessage()
	if err != nil {
		return fmt.Errorf("converting NeighbourPacketV1 to network message: %w", err)
	}
	for id, peerState := range nt.peerview {
		if peerState.round >= nt.currentRound && peerState.setID >= nt.currentSetID {
			err = nt.grandpa.network.SendMessage(id, cm)
			if err != nil {
				return fmt.Errorf("sending message to peer: %v", id)
			}
		}
	}
	return nil
}
