// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"fmt"
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/libp2p/go-libp2p/core/peer"
)

// How often neighbour messages should be rebroadcast in the case where no new packets are created
const neighbourBroadcastPeriod = time.Minute * 2

type neighborData struct {
	peer        peer.ID
	neighborMsg *NeighbourPacketV1
}

type neighborState struct {
	setID            uint64
	round            uint64
	highestFinalized uint32
}

type neighborTracker struct {
	sync.Mutex
	grandpa *Service

	peerview         map[peer.ID]neighborState
	currentSetID     uint64
	currentRound     uint64
	highestFinalized uint32

	finalizationCha chan *types.FinalisationInfo
	neighborMsgChan chan neighborData
	stoppedNeighbor chan struct{}
	wg              sync.WaitGroup
}

func newNeighborTracker(grandpa *Service, neighborChan chan neighborData) *neighborTracker {
	return &neighborTracker{
		grandpa:         grandpa,
		peerview:        make(map[peer.ID]neighborState),
		finalizationCha: grandpa.blockState.GetFinalisedNotifierChannel(),
		neighborMsgChan: neighborChan,
		stoppedNeighbor: make(chan struct{}),
	}
}

func (nt *neighborTracker) Start() {
	nt.wg.Add(1)
	go nt.run()
}

func (nt *neighborTracker) Stop() {
	nt.grandpa.blockState.FreeFinalisedNotifierChannel(nt.finalizationCha)
	close(nt.stoppedNeighbor)

	nt.wg.Wait()
}

func (nt *neighborTracker) run() {
	logger.Info("starting neighbour tracker")
	ticker := time.NewTicker(neighbourBroadcastPeriod)
	defer func() {
		ticker.Stop()
		nt.wg.Done()
	}()

	for {
		select {
		case <-ticker.C:
			logger.Debugf("neighbour message broadcast triggered by ticker")
			err := nt.BroadcastNeighborMsg()
			if err != nil {
				logger.Errorf("broadcasting neighbour message: %v", err)
			}

		case block := <-nt.finalizationCha:
			if block != nil {
				nt.updateState(block.SetID, block.Round, uint32(block.Header.Number)) //nolint
				err := nt.BroadcastNeighborMsg()
				if err != nil {
					logger.Errorf("broadcasting neighbour message: %v", err)
				}
				ticker.Reset(neighbourBroadcastPeriod)
			}
		case neighborData := <-nt.neighborMsgChan:
			if neighborData.neighborMsg.Number > nt.peerview[neighborData.peer].highestFinalized {
				nt.updatePeer(
					neighborData.peer,
					neighborData.neighborMsg.SetID,
					neighborData.neighborMsg.Round,
					neighborData.neighborMsg.Number,
				)
			}
		case <-nt.stoppedNeighbor:
			logger.Info("stopping neighbour tracker")
			return
		}
	}
}

func (nt *neighborTracker) updateState(setID uint64, round uint64, highestFinalized uint32) {
	nt.Lock()
	defer nt.Unlock()

	nt.currentSetID = setID
	nt.currentRound = round
	nt.highestFinalized = highestFinalized
}

func (nt *neighborTracker) updatePeer(p peer.ID, setID uint64, round uint64, highestFinalized uint32) {
	nt.Lock()
	defer nt.Unlock()
	peerState := neighborState{setID, round, highestFinalized}
	nt.peerview[p] = peerState
}

func (nt *neighborTracker) getPeer(p peer.ID) neighborState {
	nt.Lock()
	defer nt.Unlock()
	return nt.peerview[p]
}

func (nt *neighborTracker) BroadcastNeighborMsg() error {
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
