package bitfielddistribution

import (
	networkbridgeevents "github.com/ChainSafe/gossamer/dot/parachain/network-bridge/events"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func generateDummyPeerID(t *testing.T, bits int) peer.ID {
	_, pubk, err := crypto.GenerateKeyPair(crypto.Ed25519, bits)
	assert.Nil(t, err)
	peerID, err := peer.IDFromPublicKey(pubk)
	assert.Nil(t, err)

	return peerID
}

func TestMessageFromValidatorNeededByPeer(t *testing.T) {
	validatorSet := []parachaintypes.ValidatorID{
		[sr25519.PublicKeyLength]byte{1},
		[sr25519.PublicKeyLength]byte{2},
	}

	prpd := newPerRelayParentData(parachaintypes.SessionIndex(1), validatorSet)

	peerA := generateDummyPeerID(t, -1)
	peerB := generateDummyPeerID(t, -2)

	assert.NotEqual(t, peerA, peerB)

	pretendSent := func(state *perRelayParentData, destPeer peer.ID, signedBy parachaintypes.ValidatorID) bool {
		if state.messageFromValidatorNeededByPeer(destPeer, signedBy) {
			state.messageSentToPeer[destPeer] = map[parachaintypes.ValidatorID]struct{}{signedBy: {}}
			return true
		} else {
			return false
		}
	}

	pretendReceive := func(state *perRelayParentData, sourcePeer peer.ID, signedBy parachaintypes.ValidatorID) {
		state.messageReceivedFromPeer[sourcePeer] = map[parachaintypes.ValidatorID]struct{}{signedBy: {}}
	}

	assert.True(t, pretendSent(prpd, peerA, validatorSet[0]))
	assert.True(t, pretendSent(prpd, peerB, validatorSet[1]))

	assert.False(t, pretendSent(prpd, peerA, validatorSet[0]))

	pretendReceive(prpd, peerA, validatorSet[0])

	assert.False(t, pretendSent(prpd, peerA, validatorSet[0]))
	assert.False(t, pretendSent(prpd, peerB, validatorSet[1]))

	pretendReceive(prpd, peerB, validatorSet[1])

	assert.False(t, pretendSent(prpd, peerA, validatorSet[0]))
	assert.False(t, pretendSent(prpd, peerB, validatorSet[1]))
}

func TestBitfieldDistribution_ProcessPeerConnectedSignal(t *testing.T) {
	bd := NewBitfieldDistribution(make(chan<- any))

	assert.Equal(t, 0, len(bd.peerViews))

	signalWithVersion1 := networkbridgeevents.PeerConnected{
		PeerID:          generateDummyPeerID(t, -1),
		ProtocolVersion: uint32(1),
	}
	signalWithVersion2 := networkbridgeevents.PeerConnected{
		PeerID:          generateDummyPeerID(t, -2),
		ProtocolVersion: uint32(2),
	}
	signalWithVersion3 := networkbridgeevents.PeerConnected{
		PeerID:          generateDummyPeerID(t, -3),
		ProtocolVersion: uint32(3),
	}
	signalWithPeer4 := networkbridgeevents.PeerConnected{
		PeerID:          generateDummyPeerID(t, -4),
		ProtocolVersion: uint32(2),
	}
	signalWithPeer5 := networkbridgeevents.PeerConnected{
		PeerID:          generateDummyPeerID(t, -5),
		ProtocolVersion: uint32(2),
	}
	signalWithPeer6 := networkbridgeevents.PeerConnected{
		PeerID:          generateDummyPeerID(t, -6),
		ProtocolVersion: uint32(2),
	}
	signalWithPeer7 := networkbridgeevents.PeerConnected{
		PeerID:          generateDummyPeerID(t, -7),
		ProtocolVersion: uint32(2),
	}

	// protocol version support testing
	err := bd.ProcessPeerConnectedSignal(signalWithVersion1)
	assert.Nil(t, err)
	time.Sleep(1 * time.Second)
	assert.Equal(t, 0, len(bd.peerViews))

	err = bd.ProcessPeerConnectedSignal(signalWithVersion2)
	assert.Nil(t, err)
	time.Sleep(1 * time.Second)
	assert.Equal(t, 1, len(bd.peerViews))

	err = bd.ProcessPeerConnectedSignal(signalWithVersion3)
	assert.Nil(t, err)
	time.Sleep(1 * time.Second)
	assert.Equal(t, 2, len(bd.peerViews))

	// race condition testing
	err = bd.ProcessPeerConnectedSignal(signalWithPeer4)
	assert.Nil(t, err)
	err = bd.ProcessPeerConnectedSignal(signalWithPeer5)
	assert.Nil(t, err)
	err = bd.ProcessPeerConnectedSignal(signalWithPeer6)
	assert.Nil(t, err)
	err = bd.ProcessPeerConnectedSignal(signalWithPeer7)
	assert.Nil(t, err)

	time.Sleep(3 * time.Second)
	assert.Equal(t, 6, len(bd.peerViews))
}

func TestBitfieldDistribution_ProcessPeerDisconnectedSignal(t *testing.T) {
	bd := NewBitfieldDistribution(make(chan<- any))

	targetPeer := generateDummyPeerID(t, -2)
	signalWithVersion1 := networkbridgeevents.PeerConnected{
		PeerID:          generateDummyPeerID(t, -1),
		ProtocolVersion: uint32(1),
	}
	signalWithVersion2 := networkbridgeevents.PeerConnected{
		PeerID:          targetPeer,
		ProtocolVersion: uint32(2),
	}
	signalWithVersion3 := networkbridgeevents.PeerConnected{
		PeerID:          generateDummyPeerID(t, -3),
		ProtocolVersion: uint32(3),
	}

	err := bd.ProcessPeerConnectedSignal(signalWithVersion1)
	assert.Nil(t, err)
	err = bd.ProcessPeerConnectedSignal(signalWithVersion2)
	assert.Nil(t, err)
	err = bd.ProcessPeerConnectedSignal(signalWithVersion3)
	assert.Nil(t, err)

	time.Sleep(1 * time.Second)
	assert.Equal(t, 2, len(bd.peerViews))

	err = bd.ProcessPeerDisconnectedSignal(networkbridgeevents.PeerDisconnected{
		PeerID: targetPeer,
	})
	assert.Nil(t, err)

	time.Sleep(1 * time.Second)
	assert.Equal(t, 1, len(bd.peerViews))
}

func TestBitfieldDistribution_FilterByPeerVersion(t *testing.T) {
	p1 := generateDummyPeerID(t, -1)
	p2 := generateDummyPeerID(t, -2)
	p3 := generateDummyPeerID(t, -3)
	p4 := generateDummyPeerID(t, -4)
	p5 := generateDummyPeerID(t, -5)

	testcases := []struct {
		name          string
		peers         map[peer.ID]uint32
		targetVersion uint32
		result        []peer.ID
	}{
		{
			name: "test1",
			peers: map[peer.ID]uint32{
				p1: uint32(2),
				p2: uint32(3),
				p3: uint32(2),
				p4: uint32(3),
				p5: uint32(1),
			},
			targetVersion: uint32(1),
			result:        []peer.ID{p5},
		},
		{
			name: "test2",
			peers: map[peer.ID]uint32{
				p1: uint32(2),
				p2: uint32(3),
				p3: uint32(2),
				p4: uint32(3),
				p5: uint32(1),
			},
			targetVersion: uint32(2),
			result:        []peer.ID{p1, p3},
		},
		{
			name:          "test3",
			peers:         map[peer.ID]uint32{},
			targetVersion: uint32(2),
			result:        []peer.ID{},
		},
		{
			name: "test4",
			peers: map[peer.ID]uint32{
				p1: uint32(2),
				p2: uint32(3),
				p3: uint32(2),
				p4: uint32(3),
				p5: uint32(1),
			},
			targetVersion: uint32(4),
			result:        []peer.ID{},
		},
	}

	for _, testcase := range testcases {
		re := filterByPeerVersion(testcase.peers, testcase.targetVersion)
		assert.Equal(t, testcase.result, re, testcase.name)
	}
}
