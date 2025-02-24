package util

import (
	"testing"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
)

func TestReputationAggregator_SendImmediately(t *testing.T) {

	overseerCh := make(chan NetworkBridgeTxMessage, 1)

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
		assert.Len(t, msg.ReportPeerMessageBatch, 1)
		assert.Equal(t, repChange.CostOrBenefit(), msg.ReportPeerMessageBatch[peerID])
	default:
		t.Error("Expected immediate message, but none was sent")
	}
}

func TestReputationAggregator_BatchSend(t *testing.T) {

	overseerCh := make(chan NetworkBridgeTxMessage, 1)

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

	// Call Send to flush changes
	aggregator.Send(overseerCh)

	// Verify the batch message
	select {
	case msg := <-overseerCh:
		assert.Len(t, msg.ReportPeerMessageBatch, 2)
		assert.Equal(t, int32(10_000), msg.ReportPeerMessageBatch[peerID1])  // BenefitMinor
		assert.Equal(t, int32(200_000), msg.ReportPeerMessageBatch[peerID2]) // BenefitMajor
	default:
		t.Error("Expected batch message, but none was sent")
	}
}

func TestReputationAggregator_ClearAfterSend(t *testing.T) {

	overseerCh := make(chan NetworkBridgeTxMessage, 1)

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

	overseerCh := make(chan NetworkBridgeTxMessage, 1)

	// Create a new aggregator
	aggregator := NewReputationAggregator(func(rep UnifiedReputationChange) bool {
		return false // Always accumulate
	})

	// Add multiple reputation changes for the same peer
	peerID := peer.ID("peer1")
	aggregator.Modify(overseerCh, peerID, UnifiedReputationChange{Type: BenefitMajor, Reason: "Helpful behaviour"})
	aggregator.Modify(overseerCh, peerID, UnifiedReputationChange{Type: CostMinor, Reason: "Minor issue"})

	// Call Send to flush changes
	aggregator.Send(overseerCh)

	// Verify the accumulated result
	select {
	case msg := <-overseerCh:
		assert.Len(t, msg.ReportPeerMessageBatch, 1)
		assert.Equal(t, int32(100_000), msg.ReportPeerMessageBatch[peerID]) // 200_000 + (-100_000) = 100_000
	default:
		t.Error("Expected batch message, but none was sent")
	}
}

func TestReputationAggregator_NoActionWithoutChanges(t *testing.T) {

	overseerCh := make(chan NetworkBridgeTxMessage, 1)

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
