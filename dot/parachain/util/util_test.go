// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package util

import (
	"testing"
	"time"

	networkbridgemessages "github.com/ChainSafe/gossamer/dot/parachain/network-bridge/messages"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
)

func TestReputationAggregator_SendImmediately(t *testing.T) {

	overseerCh := make(chan any, 1)

	// Create a new aggregator with immediate send logic for Malicious type
	aggregator := NewReputationAggregator(func(rep UnifiedReputationChange) bool {
		return rep.Type == Malicious
	})

	// Mock peer and reputation change
	peerID := peer.ID("peer1")
	repChange := UnifiedReputationChange{
		Type:   Malicious,
		Reason: "Detected malicious behaviour",
	}

	// Modify the aggregator
	aggregator.Modify(overseerCh, peerID, repChange)

	// Verify the message is sent immediately

	select {
	case msg := <-overseerCh:
		switch m := msg.(type) {
		case networkbridgemessages.ReportPeer: // immediately send, so no batch here
			assert.EqualValues(t, m.PeerID, peerID)
			assert.EqualValues(t, repChange.CostOrBenefit(), m.ReputationChange.Value)
		default:
			t.Error("Expected immediate message, but wrong one was sent")
		}
	default:
		t.Error("Expected immediate message, but none was sent")
	}
}

func TestReputationAggregator_BatchSend(t *testing.T) {
	overseerCh := make(chan any, 1)

	// Create a new aggregator with no immediate send logic
	aggregator := NewReputationAggregator(func(rep UnifiedReputationChange) bool {
		return false // Always accumulate
	})

	// Add multiple reputation changes
	peerID1 := peer.ID("peer1")
	peerID2 := peer.ID("peer2")
	aggregator.Modify(overseerCh, peerID1, UnifiedReputationChange{Type: BenefitMinor, Reason: "Good behaviour"})
	aggregator.Modify(overseerCh, peerID2, UnifiedReputationChange{Type: BenefitMajor, Reason: "Excellent behaviour"})

	// Verify no messages were sent yet
	select {
	case <-overseerCh:
		t.Error("Expected no message to be sent, but one was sent")
	default:
	}

	// Verify the batch message
	numOfRep := make([]networkbridgemessages.ReportPeer, 0)
	go func() {
		for msg := range overseerCh {
			switch m := msg.(type) {
			case networkbridgemessages.ReportPeer:
				numOfRep = append(numOfRep, m)
			default:
				t.Error("Expected no networkbridgemessages to be sent, but wrong one was sent")
			}
		}
	}()

	// Call Send to flush changes
	aggregator.Send(overseerCh)

	time.Sleep(1 * time.Second)

	assert.Len(t, numOfRep, 2)
	assert.Equal(t, int32(210_000), int32(numOfRep[0].ReputationChange.Value+numOfRep[1].ReputationChange.Value))
}

func TestReputationAggregator_ClearAfterSend(t *testing.T) {

	overseerCh := make(chan any, 1)

	// Create a new aggregator
	aggregator := NewReputationAggregator(func(rep UnifiedReputationChange) bool {
		return false // Always accumulate
	})

	// Add a reputation change
	peerID := peer.ID("peer1")
	aggregator.Modify(overseerCh, peerID, UnifiedReputationChange{Type: BenefitMinor, Reason: "Positive contribution"})

	// Call Send to flush changes
	aggregator.Send(overseerCh)

	// Verify the batch message
	select {
	case <-overseerCh:
		// Expected message sent
	default:
		t.Error("Expected batch message, but none was sent")
	}

	// Verify the internal state is cleared
	assert.Empty(t, aggregator.byPeer)
}

func TestReputationAggregator_ConflictResolution(t *testing.T) {

	overseerCh := make(chan any, 1)

	// Create a new aggregator
	aggregator := NewReputationAggregator(func(rep UnifiedReputationChange) bool {
		return false // Always accumulate
	})

	// Add multiple reputation changes for the same peer
	peerID := peer.ID("peer1")
	aggregator.Modify(overseerCh, peerID, UnifiedReputationChange{Type: BenefitMajor, Reason: "Helpful behaviour"})
	aggregator.Modify(overseerCh, peerID, UnifiedReputationChange{Type: CostMinor, Reason: "Minor issue"})

	// Verify no messages were sent yet
	select {
	case <-overseerCh:
		t.Error("Expected no message to be sent, but one was sent")
	default:
	}

	numOfRep := make([]networkbridgemessages.ReportPeer, 0)
	go func() {
		for msg := range overseerCh {
			switch m := msg.(type) {
			case networkbridgemessages.ReportPeer:
				numOfRep = append(numOfRep, m)
			default:
				t.Error("Expected no networkbridgemessages to be sent, but wrong one was sent")
			}
		}
	}()

	// Call Send to flush changes
	// there is only one report at this moment
	aggregator.Send(overseerCh)

	time.Sleep(1 * time.Second)

	// Verify the accumulated result
	assert.Len(t, numOfRep, 1)
	assert.Equal(t, int32(100_000), int32(numOfRep[0].ReputationChange.Value)) // 200_000 + (-100_000) = 100_000
}

func TestReputationAggregator_NoActionWithoutChanges(t *testing.T) {

	overseerCh := make(chan any, 1)

	// Create a new aggregator
	aggregator := NewReputationAggregator(func(rep UnifiedReputationChange) bool {
		return false
	})

	// Call Send without any changes
	aggregator.Send(overseerCh)

	// Verify no messages were sent
	select {
	case <-overseerCh:
		t.Error("Expected no message, but one was sent")
	default:
		// Expected behaviour
	}
}
