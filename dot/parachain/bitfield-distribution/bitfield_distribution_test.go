package bitfielddistribution

import (
	"errors"
	"sync"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	networkbridge "github.com/ChainSafe/gossamer/dot/parachain/network-bridge"

	"github.com/ChainSafe/gossamer/dot/parachain/grid"
	networkbridgeevents "github.com/ChainSafe/gossamer/dot/parachain/network-bridge/events"
	networkbridgemessages "github.com/ChainSafe/gossamer/dot/parachain/network-bridge/messages"
	provisionermessages "github.com/ChainSafe/gossamer/dot/parachain/provisioner/messages"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/dot/parachain/util"
	validationprotocol "github.com/ChainSafe/gossamer/dot/parachain/validation-protocol"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
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

	prpd := newPerRelayParentData(parachaintypes.SigningContext{
		SessionIndex: parachaintypes.SessionIndex(1),
		ParentHash:   common.Hash{0x01},
	}, validatorSet)

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

func TestBitfieldDistribution_ProcessPeerConnectedEvent(t *testing.T) {
	bd := NewBitfieldDistribution(make(chan<- any), nil)

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
	bd.processPeerConnectedEvent(signalWithVersion1)
	assert.Equal(t, 0, len(bd.peerViews))

	bd.processPeerConnectedEvent(signalWithVersion2)
	assert.Equal(t, 1, len(bd.peerViews))

	bd.processPeerConnectedEvent(signalWithVersion3)
	assert.Equal(t, 2, len(bd.peerViews))

	// race condition testing
	wg := sync.WaitGroup{}
	wg.Add(4)
	go func() {
		bd.processPeerConnectedEvent(signalWithPeer4)
		wg.Done()
	}()
	go func() {
		bd.processPeerConnectedEvent(signalWithPeer5)
		wg.Done()
	}()
	go func() {
		bd.processPeerConnectedEvent(signalWithPeer6)
		wg.Done()
	}()
	go func() {
		bd.processPeerConnectedEvent(signalWithPeer7)
		wg.Done()
	}()

	wg.Wait()
	assert.Equal(t, 6, len(bd.peerViews))
}

func TestBitfieldDistribution_ProcessPeerDisconnectedEvent(t *testing.T) {
	bd := NewBitfieldDistribution(make(chan<- any), nil)

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

	bd.processPeerConnectedEvent(signalWithVersion1)

	bd.processPeerConnectedEvent(signalWithVersion2)

	bd.processPeerConnectedEvent(signalWithVersion3)

	assert.Equal(t, 2, len(bd.peerViews))

	bd.processPeerDisconnectedEvent(networkbridgeevents.PeerDisconnected{
		PeerID: targetPeer,
	})

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
		result        map[peer.ID]struct{}
	}{
		{
			name: "test1_should_have_1_result",
			peers: map[peer.ID]uint32{
				p1: uint32(2),
				p2: uint32(3),
				p3: uint32(2),
				p4: uint32(3),
				p5: uint32(1),
			},
			targetVersion: uint32(1),
			result: map[peer.ID]struct{}{
				p5: {},
			},
		},
		{
			name: "test2_should_have_2_result",
			peers: map[peer.ID]uint32{
				p1: uint32(2),
				p2: uint32(3),
				p3: uint32(2),
				p4: uint32(3),
				p5: uint32(1),
			},
			targetVersion: uint32(2),
			result: map[peer.ID]struct{}{
				p3: {},
				p1: {},
			},
		},
		{
			name:          "test3_should_have_0_result",
			peers:         map[peer.ID]uint32{},
			targetVersion: uint32(2),
			result:        map[peer.ID]struct{}{},
		},
		{
			name: "test4_should_have_0_result",
			peers: map[peer.ID]uint32{
				p1: uint32(2),
				p2: uint32(3),
				p3: uint32(2),
				p4: uint32(3),
				p5: uint32(1),
			},
			targetVersion: uint32(4),
			result:        map[peer.ID]struct{}{},
		},
	}

	for _, testcase := range testcases {
		m := make(map[peer.ID]struct{})
		for _, val := range filterByPeerVersion(testcase.peers, testcase.targetVersion) {
			m[val] = struct{}{}
		}
		assert.Equal(t, testcase.result, m, testcase.name)
	}
}

func prepareForValidators() (parachaintypes.ValidatorIndex, []parachaintypes.ValidatorID) {
	// prepare for the relay message call params
	validatorIndex := 0
	validatorSet := []parachaintypes.ValidatorID{
		[sr25519.PublicKeyLength]byte{1},
		[sr25519.PublicKeyLength]byte{2},
	}
	return parachaintypes.ValidatorIndex(validatorIndex), validatorSet
}

func TestBitfieldDistribution_RelayMessage_InterestedPeersEmpty(t *testing.T) {
	peerA := generateDummyPeerID(t, -1)
	peerB := generateDummyPeerID(t, -2)
	assert.NotEqual(t, peerA, peerB)

	validatorIndex, validatorSet := prepareForValidators()

	vec, err := parachaintypes.NewBitVec([]bool{true, false})
	assert.Nil(t, err)

	relayParent := common.Hash{1, 1, 1}
	message := validationprotocol.CheckedBitfield{
		Hash: relayParent,
		CheckedSignedAvailabilityBitfield: parachaintypes.CheckedSignedAvailabilityBitfield{
			Payload:        vec,
			ValidatorIndex: validatorIndex,
			Signature:      [64]byte{0},
		},
	}

	jobData := newPerRelayParentData(parachaintypes.SigningContext{
		SessionIndex: parachaintypes.SessionIndex(1),
		ParentHash:   common.Hash{0x01},
	}, validatorSet)

	// control the peer protocol version here and the peer's Head
	peerViews := map[peer.ID]*networkbridge.PeerDataViewWithVersion{
		peerA: {
			View:            parachaintypes.View{},
			ProtocolVersion: 2,
		},
		peerB: {
			View:            parachaintypes.View{},
			ProtocolVersion: 3,
		},
	}

	subSystemToOverseer := make(chan any)
	b := &BitfieldDistribution{
		subSystemToOverseer: subSystemToOverseer,
		peerViews:           peerViews,
		ourView:             parachaintypes.View{},
		topologies:          &grid.SessionGridTopologyStorage{},
		perRelayParent: map[common.Hash]*perRelayParentData{
			relayParent: jobData,
		},
		reputation: util.NewReputationAggregator(func(rep util.UnifiedReputationChange) bool {
			return false
		}),
	}

	topologies := grid.NewEmptyGridNeighbours()
	requiredRouting := grid.RequiredRoutingAll
	validatorID := jobData.validatorsSet[validatorIndex]

	done := make(chan struct{})
	go func() {
		for {
			// check the overseer chan message for the ProvisionableDataBitfield message
			request := <-subSystemToOverseer
			switch request := request.(type) {
			case provisionermessages.ProvisionableDataBitfield:
				assert.EqualValues(t, message.Hash, request.RelayParent)
				assert.EqualValues(t, message.CheckedSignedAvailabilityBitfield, request.Bitfield)
			}
			done <- struct{}{}
		}
	}()

	// call the target method
	relayMessage(jobData, topologies, b.peerViews, validatorID, message, requiredRouting, b.subSystemToOverseer)

	// wait for the provisionable content checks
	select {
	case <-done:
		t.Log("Test completed")
	case <-time.After(2 * time.Second):
		t.Fatal("Test timed out")
	}
}

func TestBitfieldDistribution_RelayMessage_FilterV2Peers(t *testing.T) {
	peerA := generateDummyPeerID(t, -1)
	peerB := generateDummyPeerID(t, -2)
	assert.NotEqual(t, peerA, peerB)

	validatorIndex, validatorSet := prepareForValidators()

	vec, err := parachaintypes.NewBitVec([]bool{true, false})
	assert.Nil(t, err)

	relayParent := common.Hash{1, 1, 1}
	message := validationprotocol.CheckedBitfield{
		Hash: relayParent,
		CheckedSignedAvailabilityBitfield: parachaintypes.CheckedSignedAvailabilityBitfield{
			Payload:        vec,
			ValidatorIndex: validatorIndex,
			Signature:      [64]byte{0},
		},
	}

	jobData := newPerRelayParentData(parachaintypes.SigningContext{
		SessionIndex: parachaintypes.SessionIndex(1),
		ParentHash:   common.Hash{0x01},
	}, validatorSet)

	// control the peer protocol version here and the peer's Head
	peerViews := map[peer.ID]*networkbridge.PeerDataViewWithVersion{
		peerA: {
			View: parachaintypes.View{
				Heads: []common.Hash{relayParent}, // valid relayParent hash
			},
			ProtocolVersion: 2, // V2
		},
		peerB: {
			View:            parachaintypes.View{},
			ProtocolVersion: 3, //V3
		},
	}

	subSystemToOverseer := make(chan any)
	b := &BitfieldDistribution{
		subSystemToOverseer: subSystemToOverseer,
		peerViews:           peerViews,
		ourView:             parachaintypes.View{},
		topologies:          &grid.SessionGridTopologyStorage{},
		perRelayParent: map[common.Hash]*perRelayParentData{
			relayParent: jobData,
		},
		reputation: util.NewReputationAggregator(func(rep util.UnifiedReputationChange) bool {
			return false
		}),
	}

	topologies := grid.NewEmptyGridNeighbours()
	requiredRouting := grid.RequiredRoutingAll
	validatorID := jobData.validatorsSet[validatorIndex]

	done := make(chan struct{})
	// check the overseer chan message for the SendValidationMessage for V2
	go func() {
		for {
			request := <-subSystemToOverseer
			switch request := request.(type) {
			case networkbridgemessages.SendValidationMessage:
				assert.EqualValues(t, []peer.ID{peerA}, request.To) // PeerA is v2

				i, v, err := request.ValidationProtocolMessage.IndexValue()
				assert.Nil(t, err)
				assert.EqualValues(t, 1, i)
				a := v.(validationprotocol.BitfieldDistribution)
				i, v, err = a.BitfieldDistributionMessage.IndexValue()
				assert.Nil(t, err)
				assert.EqualValues(t, 1, i)
				b := v.(validationprotocol.CheckedBitfield)

				assert.EqualValues(t, message.CheckedSignedAvailabilityBitfield, b.CheckedSignedAvailabilityBitfield)
				assert.EqualValues(t, message.Hash, b.Hash)

				done <- struct{}{}
			}
		}
	}()

	// call the target method
	relayMessage(jobData, topologies, b.peerViews, validatorID, message, requiredRouting, b.subSystemToOverseer)

	// wait for the v2 content SendValidationMessage checks
	select {
	case <-done:
		t.Log("Test completed")
	case <-time.After(2 * time.Second):
		t.Fatal("Test timed out")
	}
}

func TestBitfieldDistribution_RelayMessage_FilterV3Peers(t *testing.T) {
	peerA := generateDummyPeerID(t, -1)
	peerB := generateDummyPeerID(t, -2)
	assert.NotEqual(t, peerA, peerB)

	validatorIndex, validatorSet := prepareForValidators()

	vec, err := parachaintypes.NewBitVec([]bool{true, false})
	assert.Nil(t, err)

	relayParent := common.Hash{1, 1, 1}
	message := validationprotocol.CheckedBitfield{
		Hash: relayParent,
		CheckedSignedAvailabilityBitfield: parachaintypes.CheckedSignedAvailabilityBitfield{
			Payload:        vec,
			ValidatorIndex: validatorIndex,
			Signature:      [64]byte{0},
		},
	}

	jobData := newPerRelayParentData(parachaintypes.SigningContext{
		SessionIndex: parachaintypes.SessionIndex(1),
		ParentHash:   common.Hash{0x01},
	}, validatorSet)

	// control the peer protocol version here and the peer's Head
	peerViews := map[peer.ID]*networkbridge.PeerDataViewWithVersion{
		peerA: {
			View:            parachaintypes.View{},
			ProtocolVersion: 2,
		},
		peerB: {
			View: parachaintypes.View{
				Heads: []common.Hash{relayParent},
			},
			ProtocolVersion: 3,
		},
	}

	subSystemToOverseer := make(chan any)
	b := &BitfieldDistribution{
		subSystemToOverseer: subSystemToOverseer,
		peerViews:           peerViews,
		ourView:             parachaintypes.View{},
		topologies:          &grid.SessionGridTopologyStorage{},
		perRelayParent: map[common.Hash]*perRelayParentData{
			relayParent: jobData,
		},
		reputation: util.NewReputationAggregator(func(rep util.UnifiedReputationChange) bool {
			return false
		}),
	}

	topologies := grid.NewEmptyGridNeighbours()
	requiredRouting := grid.RequiredRoutingAll
	validatorID := jobData.validatorsSet[validatorIndex]

	done := make(chan struct{})
	// check the overseer chan message for the SendValidationMessage for v3
	go func() {
		for {
			request := <-subSystemToOverseer
			switch request := request.(type) {
			case networkbridgemessages.SendValidationMessage:
				assert.EqualValues(t, []peer.ID{peerB}, request.To) // PeerB is v3

				i, v, err := request.ValidationProtocolMessage.IndexValue()
				assert.Nil(t, err)
				assert.EqualValues(t, 1, i)
				a := v.(validationprotocol.BitfieldDistribution)
				i, v, err = a.BitfieldDistributionMessage.IndexValue()
				assert.Nil(t, err)
				assert.EqualValues(t, 1, i)
				b := v.(validationprotocol.CheckedBitfield)

				assert.EqualValues(t, message.CheckedSignedAvailabilityBitfield, b.CheckedSignedAvailabilityBitfield)
				assert.EqualValues(t, message.Hash, b.Hash)

				done <- struct{}{}
			}
		}
	}()

	// call the target method
	relayMessage(jobData, topologies, b.peerViews, validatorID, message, requiredRouting, b.subSystemToOverseer)

	// wait for the v3 content SendValidationMessage checks
	select {
	case <-done:
		t.Log("Test completed")
	case <-time.After(2 * time.Second):
		t.Fatal("Test timed out")
	}
}

func TestBitfieldDistribution_ProcessBitfieldDistributionMessage_NoRelayParentToWorkOn(t *testing.T) {
	overseerCh := make(chan any)
	b := NewBitfieldDistribution(overseerCh, nil)

	message := parachaintypes.DistributeBitfield{
		RelayParent: common.Hash{1, 2, 3},
	}

	// not supposed to work on relay parent related data
	err := b.processBitfieldDistributionMessage(message)
	assert.Nil(t, err)
}

func TestBitfieldDistribution_ProcessBitfieldDistributionMessage_ValidatorSetEmpty(t *testing.T) {
	jobData := newPerRelayParentData(parachaintypes.SigningContext{
		SessionIndex: parachaintypes.SessionIndex(1),
		ParentHash:   common.Hash{0x01},
	}, nil)
	peerViews := map[peer.ID]*networkbridge.PeerDataViewWithVersion{}
	overseerCh := make(chan any)
	b := &BitfieldDistribution{
		subSystemToOverseer: overseerCh,
		peerViews:           peerViews,
		ourView:             parachaintypes.View{},
		topologies:          &grid.SessionGridTopologyStorage{},
		perRelayParent: map[common.Hash]*perRelayParentData{
			{1, 2, 3}: jobData,
		},
		reputation: util.NewReputationAggregator(func(rep util.UnifiedReputationChange) bool {
			return false
		}),
	}

	message := parachaintypes.DistributeBitfield{
		RelayParent: common.Hash{1, 2, 3},
	}

	// validator set is empty
	err := b.processBitfieldDistributionMessage(message)
	assert.Nil(t, err)
}

func TestBitfieldDistribution_ProcessBitfieldDistributionMessage_ValidatorIdxInvalid(t *testing.T) {
	validatorIndex, validatorSet := prepareForValidators()
	jobData := newPerRelayParentData(parachaintypes.SigningContext{
		SessionIndex: parachaintypes.SessionIndex(1),
		ParentHash:   common.Hash{0x01},
	}, validatorSet)
	peerViews := map[peer.ID]*networkbridge.PeerDataViewWithVersion{}
	overseerCh := make(chan any)
	b := &BitfieldDistribution{
		subSystemToOverseer: overseerCh,
		peerViews:           peerViews,
		ourView:             parachaintypes.View{},
		topologies:          &grid.SessionGridTopologyStorage{},
		perRelayParent: map[common.Hash]*perRelayParentData{
			{1, 2, 3}: jobData,
		},
		reputation: util.NewReputationAggregator(func(rep util.UnifiedReputationChange) bool {
			return false
		}),
	}

	message := parachaintypes.DistributeBitfield{
		RelayParent: common.Hash{1, 2, 3},
		Bitfield: parachaintypes.UncheckedSignedAvailabilityBitfield{
			ValidatorIndex: validatorIndex + 10, // make them different
		},
	}

	// validator set is empty
	err := b.processBitfieldDistributionMessage(message)
	assert.Nil(t, err)
}

func TestBitfieldDistribution_ProcessBitfieldDistributionMessage_CheckSignedAvailabilityBitfield(t *testing.T) {
	validatorIndex := 0

	keyring, err := keystore.NewSr25519Keyring()
	assert.Nil(t, err)
	aliceKeypair := keyring.Alice().(*sr25519.Keypair)

	validatorSet := []parachaintypes.ValidatorID{
		parachaintypes.ValidatorID(aliceKeypair.Public().Encode()),
		[sr25519.PublicKeyLength]byte{2},
	}
	jobData := newPerRelayParentData(parachaintypes.SigningContext{
		SessionIndex: parachaintypes.SessionIndex(1),
		ParentHash:   common.Hash{0x01},
	}, validatorSet)
	peerViews := map[peer.ID]*networkbridge.PeerDataViewWithVersion{}

	gt := grid.NewSessionGridTopology([]uint{1, 2, 3}, []grid.TopologyPeerInfo{{
		Peers:          []peer.ID{"peer1", "peer2"},
		ValidatorIndex: parachaintypes.ValidatorIndex(1),
		DiscoveryID:    types.AuthorityID{1},
	}})
	sgte := &grid.SessionGridTopologyEntry{
		Topology:     gt,
		LocalIndex:   10,
		SessionIndex: 99,
	}

	overseerCh := make(chan any)
	b := &BitfieldDistribution{
		subSystemToOverseer: overseerCh,
		peerViews:           peerViews,
		ourView:             parachaintypes.View{},
		topologies: &grid.SessionGridTopologyStorage{
			CurrentTopology: sgte,
			PrevTopology:    sgte,
		},
		perRelayParent: map[common.Hash]*perRelayParentData{
			{1, 2, 3}: jobData,
		},
		reputation: util.NewReputationAggregator(func(rep util.UnifiedReputationChange) bool {
			return false
		}),
	}

	bitfield, err := parachaintypes.NewBitVec([]bool{true, true, false})
	assert.Nil(t, err)
	data, err := bitfield.MarshalSCALE()
	assert.Nil(t, err)

	signature, err := aliceKeypair.Sign(data)
	assert.Nil(t, err)

	message := parachaintypes.DistributeBitfield{
		RelayParent: common.Hash{1, 2, 3},
		Bitfield: parachaintypes.UncheckedSignedAvailabilityBitfield{
			Payload:        bitfield,
			ValidatorIndex: parachaintypes.ValidatorIndex(validatorIndex),
			Signature:      parachaintypes.ValidatorSignature(signature),
		},
	}

	done := make(chan struct{})
	go func() {
		for {
			request := <-overseerCh
			switch request.(type) {
			// only provisionermessages.ProvisionableDataBitfield should be received here
			case provisionermessages.ProvisionableDataBitfield:
				done <- struct{}{}
			default:
				t.Error("should not receive other type of message")
			}
		}
	}()

	err = b.processBitfieldDistributionMessage(message)
	assert.Nil(t, err)

	select {
	case <-done:
		t.Log("expected message received from overseerCh")
	case <-time.After(2 * time.Second):
		t.Fatal("Test timed out")
	}
}

func TestBitfieldDistribution_ProcessIncomingPeerMessageEvent_InvalidSignalType(t *testing.T) {
	peerA := generateDummyPeerID(t, -1)

	message := validationprotocol.Statement{
		Hash: common.Hash{1, 2, 3},
		UncheckedSignedFullStatement: parachaintypes.UncheckedSignedFullStatement{
			Payload:        parachaintypes.StatementVDT{},
			ValidatorIndex: 0,
			Signature:      parachaintypes.ValidatorSignature([64]byte{1}),
		},
	}

	vb := &validationprotocol.StatementDistributionMessage{}
	err := vb.SetValue(message)
	assert.Nil(t, err)

	vp := &validationprotocol.ValidationProtocol{}
	err = vp.SetValue(validationprotocol.StatementDistribution{
		StatementDistributionMessage: *vb,
	})
	assert.Nil(t, err)

	signal := networkbridgeevents.PeerMessage[validationprotocol.ValidationProtocol]{
		PeerID:  peerA,
		Message: *vp,
	}

	overseerCh := make(chan any)
	b := NewBitfieldDistribution(overseerCh, nil)

	err = b.processIncomingPeerMessageEvent(signal)
	assert.NotNil(t, err)
	assert.EqualValues(t, err.Error(), "invalid value for peer message handler, supporsed to be "+
		"validationprotocol.BitfieldDistribution")
}

func TestBitfieldDistribution_ProcessIncomingPeerMessageEvent_CheckedBitfield(t *testing.T) {
	peerA := generateDummyPeerID(t, -1)
	peerB := generateDummyPeerID(t, -2)
	assert.NotEqual(t, peerA, peerB)

	payload, err := parachaintypes.NewBitVec([]bool{true, false})
	assert.Nil(t, err)

	message := validationprotocol.CheckedBitfield{
		Hash: common.Hash{1, 2, 3},
		CheckedSignedAvailabilityBitfield: parachaintypes.CheckedSignedAvailabilityBitfield{
			Payload:        payload,
			ValidatorIndex: 0,
			Signature:      parachaintypes.ValidatorSignature([64]byte{1}),
		},
	}

	vb := &validationprotocol.BitfieldDistributionMessage{}
	err = vb.SetValue(message)
	assert.Nil(t, err)

	vp := &validationprotocol.ValidationProtocol{}
	err = vp.SetValue(validationprotocol.BitfieldDistribution{
		BitfieldDistributionMessage: *vb,
	})
	assert.Nil(t, err)

	signal := networkbridgeevents.PeerMessage[validationprotocol.ValidationProtocol]{
		PeerID:  peerA,
		Message: *vp,
	}

	overseerCh := make(chan any)
	b := NewBitfieldDistribution(overseerCh, nil)

	err = b.processIncomingPeerMessageEvent(signal)
	assert.NotNil(t, err)
	assert.EqualValues(t, err.Error(), "invalid value for uncheckedBitfield")
}

func TestBitfieldDistribution_ProcessIncomingPeerMessageEvent_NotContainTheView(t *testing.T) {
	peerA := generateDummyPeerID(t, -1)
	peerB := generateDummyPeerID(t, -2)
	assert.NotEqual(t, peerA, peerB)

	payload, err := parachaintypes.NewBitVec([]bool{true, false})
	assert.Nil(t, err)

	relayParent := common.Hash{1, 2, 3}

	message := validationprotocol.UncheckedBitfield{
		Hash: relayParent,
		UncheckedSignedAvailabilityBitfield: parachaintypes.UncheckedSignedAvailabilityBitfield{
			Payload:        payload,
			ValidatorIndex: 0,
			Signature:      parachaintypes.ValidatorSignature([64]byte{1}),
		},
	}

	vb := &validationprotocol.BitfieldDistributionMessage{}
	err = vb.SetValue(message)
	assert.Nil(t, err)

	vp := &validationprotocol.ValidationProtocol{}
	err = vp.SetValue(validationprotocol.BitfieldDistribution{
		BitfieldDistributionMessage: *vb,
	})
	assert.Nil(t, err)

	signal := networkbridgeevents.PeerMessage[validationprotocol.ValidationProtocol]{
		PeerID:  peerA,
		Message: *vp,
	}

	overseerCh := make(chan any)

	peerViews := map[peer.ID]*networkbridge.PeerDataViewWithVersion{}
	b := &BitfieldDistribution{
		subSystemToOverseer: overseerCh,
		peerViews:           peerViews,
		ourView:             parachaintypes.View{},
		topologies:          &grid.SessionGridTopologyStorage{},
		perRelayParent:      map[common.Hash]*perRelayParentData{},
		reputation: util.NewReputationAggregator(func(rep util.UnifiedReputationChange) bool {
			return false
		}),
	}

	err = b.processIncomingPeerMessageEvent(signal)
	assert.Nil(t, err)
}

func TestBitfieldDistribution_ProcessIncomingPeerMessageEvent_JobDataEmpty(t *testing.T) {
	peerA := generateDummyPeerID(t, -1)
	peerB := generateDummyPeerID(t, -2)
	assert.NotEqual(t, peerA, peerB)

	payload, err := parachaintypes.NewBitVec([]bool{true, false})
	assert.Nil(t, err)

	relayParent := common.Hash{1, 2, 3}

	message := validationprotocol.UncheckedBitfield{
		Hash: relayParent,
		UncheckedSignedAvailabilityBitfield: parachaintypes.UncheckedSignedAvailabilityBitfield{
			Payload:        payload,
			ValidatorIndex: 0,
			Signature:      parachaintypes.ValidatorSignature([64]byte{1}),
		},
	}

	vb := &validationprotocol.BitfieldDistributionMessage{}
	err = vb.SetValue(message)
	assert.Nil(t, err)

	vp := &validationprotocol.ValidationProtocol{}
	err = vp.SetValue(validationprotocol.BitfieldDistribution{
		BitfieldDistributionMessage: *vb,
	})
	assert.Nil(t, err)

	signal := networkbridgeevents.PeerMessage[validationprotocol.ValidationProtocol]{
		PeerID:  peerA,
		Message: *vp,
	}

	overseerCh := make(chan any)

	peerViews := map[peer.ID]*networkbridge.PeerDataViewWithVersion{}
	b := &BitfieldDistribution{
		subSystemToOverseer: overseerCh,
		peerViews:           peerViews,
		ourView: parachaintypes.View{
			Heads: []common.Hash{relayParent},
		},
		topologies:     &grid.SessionGridTopologyStorage{},
		perRelayParent: map[common.Hash]*perRelayParentData{},
		reputation: util.NewReputationAggregator(func(rep util.UnifiedReputationChange) bool {
			return false
		}),
	}

	err = b.processIncomingPeerMessageEvent(signal)
	assert.Nil(t, err)
}

func TestBitfieldDistribution_ProcessIncomingPeerMessageEvent_ValidatorSetEmpty(t *testing.T) {
	peerA := generateDummyPeerID(t, -1)
	peerB := generateDummyPeerID(t, -2)
	assert.NotEqual(t, peerA, peerB)

	payload, err := parachaintypes.NewBitVec([]bool{true, false})
	assert.Nil(t, err)

	relayParent := common.Hash{1, 2, 3}

	message := validationprotocol.UncheckedBitfield{
		Hash: relayParent,
		UncheckedSignedAvailabilityBitfield: parachaintypes.UncheckedSignedAvailabilityBitfield{
			Payload:        payload,
			ValidatorIndex: 0,
			Signature:      parachaintypes.ValidatorSignature([64]byte{1}),
		},
	}

	vb := &validationprotocol.BitfieldDistributionMessage{}
	err = vb.SetValue(message)
	assert.Nil(t, err)

	vp := &validationprotocol.ValidationProtocol{}
	err = vp.SetValue(validationprotocol.BitfieldDistribution{
		BitfieldDistributionMessage: *vb,
	})
	assert.Nil(t, err)

	signal := networkbridgeevents.PeerMessage[validationprotocol.ValidationProtocol]{
		PeerID:  peerA,
		Message: *vp,
	}

	jobData := newPerRelayParentData(parachaintypes.SigningContext{
		SessionIndex: parachaintypes.SessionIndex(1),
		ParentHash:   common.Hash{0x01},
	}, nil)

	overseerCh := make(chan any)
	peerViews := map[peer.ID]*networkbridge.PeerDataViewWithVersion{}
	b := &BitfieldDistribution{
		subSystemToOverseer: overseerCh,
		peerViews:           peerViews,
		ourView: parachaintypes.View{
			Heads: []common.Hash{relayParent},
		},
		topologies: &grid.SessionGridTopologyStorage{},
		perRelayParent: map[common.Hash]*perRelayParentData{
			{1, 2, 3}: jobData,
		},
		reputation: util.NewReputationAggregator(func(rep util.UnifiedReputationChange) bool {
			return false
		}),
	}

	err = b.processIncomingPeerMessageEvent(signal)
	assert.Nil(t, err)
}

func TestBitfieldDistribution_ProcessIncomingPeerMessageEvent_ValidatorNotExist(t *testing.T) {
	peerA := generateDummyPeerID(t, -1)
	peerB := generateDummyPeerID(t, -2)
	assert.NotEqual(t, peerA, peerB)

	payload, err := parachaintypes.NewBitVec([]bool{true, false})
	assert.Nil(t, err)

	relayParent := common.Hash{1, 2, 3}

	message := validationprotocol.UncheckedBitfield{
		Hash: relayParent,
		UncheckedSignedAvailabilityBitfield: parachaintypes.UncheckedSignedAvailabilityBitfield{
			Payload:        payload,
			ValidatorIndex: 10, // uint32(validatorIdx) >= uint32(len(jobData.validatorsSet))
			Signature:      parachaintypes.ValidatorSignature([64]byte{1}),
		},
	}

	vb := &validationprotocol.BitfieldDistributionMessage{}
	err = vb.SetValue(message)
	assert.Nil(t, err)

	vp := &validationprotocol.ValidationProtocol{}
	err = vp.SetValue(validationprotocol.BitfieldDistribution{
		BitfieldDistributionMessage: *vb,
	})
	assert.Nil(t, err)

	signal := networkbridgeevents.PeerMessage[validationprotocol.ValidationProtocol]{
		PeerID:  peerA,
		Message: *vp,
	}

	_, validatorSet := prepareForValidators()
	jobData := newPerRelayParentData(parachaintypes.SigningContext{
		SessionIndex: parachaintypes.SessionIndex(1),
		ParentHash:   common.Hash{0x01},
	}, validatorSet)

	overseerCh := make(chan any)
	peerViews := map[peer.ID]*networkbridge.PeerDataViewWithVersion{}
	b := &BitfieldDistribution{
		subSystemToOverseer: overseerCh,
		peerViews:           peerViews,
		ourView: parachaintypes.View{
			Heads: []common.Hash{relayParent},
		},
		topologies: &grid.SessionGridTopologyStorage{},
		perRelayParent: map[common.Hash]*perRelayParentData{
			{1, 2, 3}: jobData,
		},
		reputation: util.NewReputationAggregator(func(rep util.UnifiedReputationChange) bool {
			return false
		}),
	}

	err = b.processIncomingPeerMessageEvent(signal)
	assert.Nil(t, err)
}

func TestBitfieldDistribution_ProcessIncomingPeerMessageEvent_ReceivedSetNotEmpty(t *testing.T) {
	peerA := generateDummyPeerID(t, -1)
	peerB := generateDummyPeerID(t, -2)
	assert.NotEqual(t, peerA, peerB)

	payload, err := parachaintypes.NewBitVec([]bool{true, false})
	assert.Nil(t, err)

	relayParent := common.Hash{1, 2, 3}

	message := validationprotocol.UncheckedBitfield{
		Hash: relayParent,
		UncheckedSignedAvailabilityBitfield: parachaintypes.UncheckedSignedAvailabilityBitfield{
			Payload:        payload,
			ValidatorIndex: 0,
			Signature:      parachaintypes.ValidatorSignature([64]byte{1}),
		},
	}

	vb := &validationprotocol.BitfieldDistributionMessage{}
	err = vb.SetValue(message)
	assert.Nil(t, err)

	vp := &validationprotocol.ValidationProtocol{}
	err = vp.SetValue(validationprotocol.BitfieldDistribution{
		BitfieldDistributionMessage: *vb,
	})
	assert.Nil(t, err)

	signal := networkbridgeevents.PeerMessage[validationprotocol.ValidationProtocol]{
		PeerID:  peerA,
		Message: *vp,
	}

	_, validatorSet := prepareForValidators()
	jobData := newPerRelayParentData(parachaintypes.SigningContext{
		SessionIndex: parachaintypes.SessionIndex(1),
		ParentHash:   common.Hash{0x01},
	}, validatorSet)

	overseerCh := make(chan any)
	peerViews := map[peer.ID]*networkbridge.PeerDataViewWithVersion{}
	b := &BitfieldDistribution{
		subSystemToOverseer: overseerCh,
		peerViews:           peerViews,
		ourView: parachaintypes.View{
			Heads: []common.Hash{relayParent},
		},
		topologies: &grid.SessionGridTopologyStorage{},
		perRelayParent: map[common.Hash]*perRelayParentData{
			{1, 2, 3}: jobData,
		},
		reputation: util.NewReputationAggregator(func(rep util.UnifiedReputationChange) bool {
			return false
		}),
	}

	pretendReceive := func(state *perRelayParentData, sourcePeer peer.ID, signedBy parachaintypes.ValidatorID) {
		state.messageReceivedFromPeer[sourcePeer] = map[parachaintypes.ValidatorID]struct{}{signedBy: {}}
	}
	pretendReceive(jobData, peerA, validatorSet[0])

	err = b.processIncomingPeerMessageEvent(signal)
	assert.Nil(t, err)
}

func TestBitfieldDistribution_ProcessIncomingPeerMessageEvent_DuplicatedMessages(t *testing.T) {
	peerA := generateDummyPeerID(t, -1)
	peerB := generateDummyPeerID(t, -2)
	assert.NotEqual(t, peerA, peerB)

	payload, err := parachaintypes.NewBitVec([]bool{true, false})
	assert.Nil(t, err)

	relayParent := common.Hash{1, 2, 3}

	message := validationprotocol.UncheckedBitfield{
		Hash: relayParent,
		UncheckedSignedAvailabilityBitfield: parachaintypes.UncheckedSignedAvailabilityBitfield{
			Payload:        payload,
			ValidatorIndex: 0,
			Signature:      parachaintypes.ValidatorSignature([64]byte{1}),
		},
	}

	vb := &validationprotocol.BitfieldDistributionMessage{}
	err = vb.SetValue(message)
	assert.Nil(t, err)

	vp := &validationprotocol.ValidationProtocol{}
	err = vp.SetValue(validationprotocol.BitfieldDistribution{
		BitfieldDistributionMessage: *vb,
	})
	assert.Nil(t, err)

	signal := networkbridgeevents.PeerMessage[validationprotocol.ValidationProtocol]{
		PeerID:  peerA,
		Message: *vp,
	}

	validatorIndex, validatorSet := prepareForValidators()
	jobData := newPerRelayParentData(parachaintypes.SigningContext{
		SessionIndex: parachaintypes.SessionIndex(1),
		ParentHash:   common.Hash{0x01},
	}, validatorSet)

	jobData.onePerValidator = map[parachaintypes.ValidatorID]*validationprotocol.BitfieldDistributionMessage{
		validatorSet[validatorIndex]: vb,
	}

	overseerCh := make(chan any)
	peerViews := map[peer.ID]*networkbridge.PeerDataViewWithVersion{}
	b := &BitfieldDistribution{
		subSystemToOverseer: overseerCh,
		peerViews:           peerViews,
		ourView: parachaintypes.View{
			Heads: []common.Hash{relayParent},
		},
		topologies: &grid.SessionGridTopologyStorage{},
		perRelayParent: map[common.Hash]*perRelayParentData{
			{1, 2, 3}: jobData,
		},
		reputation: util.NewReputationAggregator(func(rep util.UnifiedReputationChange) bool {
			return false
		}),
	}

	err = b.processIncomingPeerMessageEvent(signal)
	assert.Nil(t, err)
}

func TestBitfieldDistribution_ProcessIncomingPeerMessageEvent_Success(t *testing.T) {
	peerA := generateDummyPeerID(t, -1)
	peerB := generateDummyPeerID(t, -2)
	assert.NotEqual(t, peerA, peerB)

	payload, err := parachaintypes.NewBitVec([]bool{true, false})
	assert.Nil(t, err)

	keyring, err := keystore.NewSr25519Keyring()
	assert.Nil(t, err)
	aliceKeypair := keyring.Alice().(*sr25519.Keypair)

	validatorIndex := 0
	validatorSet := []parachaintypes.ValidatorID{
		parachaintypes.ValidatorID(aliceKeypair.Public().Encode()),
		[sr25519.PublicKeyLength]byte{2},
	}

	data, err := payload.MarshalSCALE()
	assert.Nil(t, err)

	signature, err := aliceKeypair.Sign(data)
	assert.Nil(t, err)

	relayParent := common.Hash{1, 2, 3}

	message := validationprotocol.UncheckedBitfield{
		Hash: relayParent,
		UncheckedSignedAvailabilityBitfield: parachaintypes.UncheckedSignedAvailabilityBitfield{
			Payload:        payload,
			ValidatorIndex: 0,
			Signature:      parachaintypes.ValidatorSignature(signature),
		},
	}

	vb := &validationprotocol.BitfieldDistributionMessage{}
	err = vb.SetValue(message)
	assert.Nil(t, err)

	vp := &validationprotocol.ValidationProtocol{}
	err = vp.SetValue(validationprotocol.BitfieldDistribution{
		BitfieldDistributionMessage: *vb,
	})
	assert.Nil(t, err)

	signal := networkbridgeevents.PeerMessage[validationprotocol.ValidationProtocol]{
		PeerID:  peerA,
		Message: *vp,
	}

	jobData := newPerRelayParentData(parachaintypes.SigningContext{
		SessionIndex: parachaintypes.SessionIndex(1),
		ParentHash:   common.Hash{0x01},
	}, validatorSet)

	storedMessage := validationprotocol.UncheckedBitfield{
		Hash: relayParent,
		UncheckedSignedAvailabilityBitfield: parachaintypes.UncheckedSignedAvailabilityBitfield{
			Payload:        payload,
			ValidatorIndex: 1,
			Signature:      parachaintypes.ValidatorSignature([64]byte{2}),
		},
	}
	messageStoredInState := &validationprotocol.BitfieldDistributionMessage{}
	err = messageStoredInState.SetValue(storedMessage)
	assert.Nil(t, err)

	jobData.onePerValidator = map[parachaintypes.ValidatorID]*validationprotocol.BitfieldDistributionMessage{
		validatorSet[validatorIndex]: messageStoredInState,
	}

	overseerCh := make(chan any)
	peerViews := map[peer.ID]*networkbridge.PeerDataViewWithVersion{}
	gt := grid.NewSessionGridTopology([]uint{1, 2, 3}, []grid.TopologyPeerInfo{{
		Peers:          []peer.ID{"peer1", "peer2"},
		ValidatorIndex: parachaintypes.ValidatorIndex(1),
		DiscoveryID:    types.AuthorityID{1},
	}})
	sgte := &grid.SessionGridTopologyEntry{
		Topology:        gt,
		LocalNeighbours: grid.NewEmptyGridNeighbours(),
		LocalIndex:      10,
		SessionIndex:    99,
	}
	b := &BitfieldDistribution{
		subSystemToOverseer: overseerCh,
		peerViews:           peerViews,
		ourView: parachaintypes.View{
			Heads: []common.Hash{relayParent},
		},
		topologies: &grid.SessionGridTopologyStorage{
			CurrentTopology: sgte,
			PrevTopology:    sgte,
		},
		perRelayParent: map[common.Hash]*perRelayParentData{
			{1, 2, 3}: jobData,
		},
		reputation: util.NewReputationAggregator(func(rep util.UnifiedReputationChange) bool {
			return false
		}),
	}

	done := make(chan struct{})
	go func() {
		for {
			request := <-overseerCh
			switch request.(type) {
			// only provisionermessages.ProvisionableDataBitfield in this case
			case provisionermessages.ProvisionableDataBitfield:
				done <- struct{}{}
			default:
				t.Error("should not receive other type of message")
			}
		}
	}()

	err = b.processIncomingPeerMessageEvent(signal)
	assert.Nil(t, err)

	select {
	case <-done:
		t.Log("expected message received from overseerCh")
	case <-time.After(2 * time.Second):
		t.Fatal("Test timed out")
	}
}

func TestBitfieldDistribution_SendTrackedGossipMessage_noRelayParent(t *testing.T) {
	overseerCh := make(chan any)

	b := NewBitfieldDistribution(overseerCh, nil)

	dest := peer.ID("tester")

	validatorIndex, validatorSet := prepareForValidators()

	vec, err := parachaintypes.NewBitVec([]bool{true, false})
	assert.Nil(t, err)

	// perRelayParent data not exist for this hash
	relayParent := common.Hash{1, 1, 1}
	message := validationprotocol.CheckedBitfield{
		Hash: relayParent,
		CheckedSignedAvailabilityBitfield: parachaintypes.CheckedSignedAvailabilityBitfield{
			Payload:        vec,
			ValidatorIndex: validatorIndex,
			Signature:      [64]byte{0},
		},
	}

	done := make(chan struct{})
	go func() {
		// here we expect there is nothing send over via overseerCh
		b.sendTrackedGossipMessage(dest, validatorSet[0], message)
		done <- struct{}{}
	}()

	select {
	case <-overseerCh:
		t.Fatal("should not receive any type of message")
	case <-time.After(2 * time.Second):
		t.Fatal("test time out")
	case <-done:
		t.Log("test complete")
	}
}

func TestBitfieldDistribution_SendTrackedGossipMessage_noPeerView(t *testing.T) {
	overseerCh := make(chan any)

	b := NewBitfieldDistribution(overseerCh, nil)

	dest := peer.ID("tester")

	validatorIndex, validatorSet := prepareForValidators()

	vec, err := parachaintypes.NewBitVec([]bool{true, false})
	assert.Nil(t, err)

	relayParent := common.Hash{1, 1, 1}

	perRelayParent := newPerRelayParentData(parachaintypes.SigningContext{
		SessionIndex: parachaintypes.SessionIndex(1),
		ParentHash:   common.Hash{0x01},
	}, validatorSet)
	b.perRelayParent[relayParent] = perRelayParent

	message := validationprotocol.CheckedBitfield{
		Hash: relayParent,
		CheckedSignedAvailabilityBitfield: parachaintypes.CheckedSignedAvailabilityBitfield{
			Payload:        vec,
			ValidatorIndex: validatorIndex,
			Signature:      [64]byte{0},
		},
	}

	done := make(chan struct{})
	go func() {
		b.sendTrackedGossipMessage(dest, validatorSet[0], message)
		done <- struct{}{}
	}()

	select {
	case <-overseerCh:
		t.Fatal("should not receive any type of message")
	case <-time.After(2 * time.Second):
		t.Fatal("test time out")
	case <-done:
		t.Log("test complete")
	}
}

func TestBitfieldDistribution_SendTrackedGossipMessage_noMessageSentToPeer(t *testing.T) {
	overseerCh := make(chan any)

	b := NewBitfieldDistribution(overseerCh, nil)

	dest := peer.ID("tester")

	validatorIndex, validatorSet := prepareForValidators()

	vec, err := parachaintypes.NewBitVec([]bool{true, false})
	assert.Nil(t, err)

	relayParent := common.Hash{1, 1, 1}

	perRelayParent := newPerRelayParentData(parachaintypes.SigningContext{
		SessionIndex: parachaintypes.SessionIndex(1),
		ParentHash:   common.Hash{0x01},
	}, validatorSet)
	b.perRelayParent[relayParent] = perRelayParent

	b.peerViews[dest] = &networkbridge.PeerDataViewWithVersion{
		View:            parachaintypes.View{},
		ProtocolVersion: 2,
	}

	message := validationprotocol.CheckedBitfield{
		Hash: relayParent,
		CheckedSignedAvailabilityBitfield: parachaintypes.CheckedSignedAvailabilityBitfield{
			Payload:        vec,
			ValidatorIndex: validatorIndex,
			Signature:      [64]byte{0},
		},
	}

	done := make(chan struct{})
	go func() {
		for {
			request := <-overseerCh
			switch request.(type) {
			// only networkbridgemessages.SendValidationMessage should be received here
			case networkbridgemessages.SendValidationMessage:
				done <- struct{}{}
			default:
				t.Error("should not receive other type of message")
			}
		}
	}()

	b.sendTrackedGossipMessage(dest, validatorSet[0], message)

	select {
	case <-done:
		t.Log("test complete")
	case <-time.After(2 * time.Second):
		t.Fatal("test time out")
	}
}

func TestBitfieldDistribution_handlePeerViewChange_noPeerView(t *testing.T) {
	overseerCh := make(chan any)
	origin := peer.ID("tester")
	_, validatorSet := prepareForValidators()
	b := NewBitfieldDistribution(overseerCh, nil)
	relayParent := common.Hash{1, 1, 1}
	perRelayParent := newPerRelayParentData(parachaintypes.SigningContext{
		SessionIndex: parachaintypes.SessionIndex(1),
		ParentHash:   common.Hash{0x01},
	}, validatorSet)
	b.perRelayParent[relayParent] = perRelayParent
	view := parachaintypes.View{}

	done := make(chan struct{})
	go func() {
		b.handlePeerViewChange(origin, view)
		done <- struct{}{}
	}()

	select {
	case <-overseerCh:
		t.Fatal("should not receive any type of message")
	case <-time.After(2 * time.Second):
		t.Fatal("test time out")
	case <-done:
		t.Log("test complete")
	}
}

func TestBitfieldDistribution_handlePeerViewChange_noGossipPeer(t *testing.T) {
	overseerCh := make(chan any)
	origin := peer.ID("tester")
	_, validatorSet := prepareForValidators()

	b := NewBitfieldDistribution(overseerCh, nil)
	relayParent := common.Hash{1, 1, 1}
	perRelayParent := newPerRelayParentData(parachaintypes.SigningContext{
		SessionIndex: parachaintypes.SessionIndex(1),
		ParentHash:   common.Hash{0x01},
	}, validatorSet)
	b.perRelayParent[relayParent] = perRelayParent

	// old view
	b.peerViews[origin] = &networkbridge.PeerDataViewWithVersion{
		View: parachaintypes.View{
			Heads: []common.Hash{{0x01}, {0x02}, {0x03}, {0x04}},
		},
		ProtocolVersion: 2,
	}
	// new view
	newView := parachaintypes.View{
		Heads: []common.Hash{{0x01}, {0x02}, {0x05}, {0x06}},
	}

	done := make(chan struct{})
	go func() {
		b.handlePeerViewChange(origin, newView)
		done <- struct{}{}
	}()

	select {
	case <-overseerCh:
		t.Fatal("should not receive any type of message")
	case <-time.After(2 * time.Second):
		t.Fatal("test time out")
	case <-done:
		t.Log("test complete")
	}
}

func TestBitfieldDistribution_handlePeerViewChange_GossipPeer(t *testing.T) {
	overseerCh := make(chan any)
	origin := peer.ID("tester")
	validatorIndex, validatorSet := prepareForValidators()
	vec, err := parachaintypes.NewBitVec([]bool{true, false})
	assert.Nil(t, err)

	// {0x05} does not have jobData, so it will cover the jobData == nil branch
	// {0x06} has dummy jobData, so it will go through all the logic and send out the message to overseerCh
	relayParent := common.Hash{0x06}
	message := validationprotocol.CheckedBitfield{
		Hash: relayParent,
		CheckedSignedAvailabilityBitfield: parachaintypes.CheckedSignedAvailabilityBitfield{
			Payload:        vec,
			ValidatorIndex: validatorIndex,
			Signature:      [64]byte{0},
		},
	}
	bvdt := validationprotocol.NewBitfieldDistributionMessageVDT()
	err = bvdt.SetValue(message)
	assert.Nil(t, err)

	perRelayParent := newPerRelayParentData(parachaintypes.SigningContext{
		SessionIndex: parachaintypes.SessionIndex(1),
		ParentHash:   common.Hash{0x01},
	}, validatorSet)
	perRelayParent.onePerValidator[validatorSet[0]] = &bvdt
	b := NewBitfieldDistribution(overseerCh, nil)
	b.perRelayParent[relayParent] = perRelayParent
	b.topologies.CurrentTopology.LocalNeighbours.PeersRow[origin] = struct{}{}

	// old view
	b.peerViews[origin] = &networkbridge.PeerDataViewWithVersion{
		View: parachaintypes.View{
			Heads: []common.Hash{{0x01}, {0x02}, {0x03}, {0x04}},
		},
		ProtocolVersion: 2,
	}
	// new view
	newView := parachaintypes.View{
		Heads: []common.Hash{{0x01}, {0x02}, {0x05}, {0x06}},
	}

	done := make(chan any)
	go func() {
		for {
			request := <-overseerCh
			switch request.(type) {
			case networkbridgemessages.SendValidationMessage:
				done <- struct{}{}
			default:
				t.Error("should not receive other type of message")
			}
		}
	}()

	b.handlePeerViewChange(origin, newView)

	select {
	case <-done:
		t.Log("test complete")
	case <-time.After(2 * time.Second):
		t.Fatal("test time out")
	}
}

func TestBitfieldDistribution_ProcessOurViewChangeEvent_NewViewNotExistInState(t *testing.T) {
	oldView := parachaintypes.View{
		Heads:           []common.Hash{{0x02}},
		FinalizedNumber: 2,
	}
	newView := networkbridgeevents.OurViewChange{
		View: parachaintypes.View{
			Heads:           []common.Hash{{0x01}},
			FinalizedNumber: 1,
		},
	}

	b := NewBitfieldDistribution(nil, nil)
	b.ourView = oldView

	// before the override
	assert.Equal(t, parachaintypes.View{
		Heads:           []common.Hash{{0x02}},
		FinalizedNumber: 2,
	}, b.ourView)

	b.processOurViewChangeEvent(newView)

	// after the override
	// in the new view but not in the old view, so we should get {0x01}
	assert.Equal(t, parachaintypes.View{
		Heads:           []common.Hash{{0x01}},
		FinalizedNumber: 1,
	}, b.ourView)
}

func TestBitfieldDistribution_ProcessOurViewChangeEventRemoveView(t *testing.T) {
	oldView := parachaintypes.View{
		Heads:           []common.Hash{{0x02}},
		FinalizedNumber: 2,
	}
	newView := networkbridgeevents.OurViewChange{
		View: parachaintypes.View{
			Heads:           []common.Hash{{0x01}},
			FinalizedNumber: 1,
		},
	}

	b := NewBitfieldDistribution(nil, nil)
	b.ourView = oldView
	b.perRelayParent[common.Hash{0x02}] = newPerRelayParentData(parachaintypes.SigningContext{
		SessionIndex: parachaintypes.SessionIndex(1),
		ParentHash:   common.Hash{0x01},
	}, nil)

	// before the override
	assert.Equal(t, parachaintypes.View{
		Heads:           []common.Hash{{0x02}},
		FinalizedNumber: 2,
	}, b.ourView)
	assert.NotNil(t, b.perRelayParent[common.Hash{0x02}])

	b.processOurViewChangeEvent(newView)

	// after the override
	// in the old view but not in the new view, so we should get {0x02} and remove its perRelayParentData
	assert.Equal(t, parachaintypes.View{
		Heads:           []common.Hash{{0x01}},
		FinalizedNumber: 1,
	}, b.ourView)

	assert.Nil(t, b.perRelayParent[common.Hash{0x02}])
}

func TestBitfieldDistribution_processUpdatedAuthorityIDsEvent(t *testing.T) {
	gt := grid.NewSessionGridTopology([]uint{1}, []grid.TopologyPeerInfo{{
		Peers:          []peer.ID{"peer1", "peer2"},
		ValidatorIndex: parachaintypes.ValidatorIndex(1),
		DiscoveryID:    types.AuthorityID{2},
	},
	})
	assert.Len(t, gt.Peers, 2)
	sgte := &grid.SessionGridTopologyEntry{
		Topology: gt,
		//LocalNeighbours: gn,
		LocalIndex:   0,
		SessionIndex: 99,
	}

	b := NewBitfieldDistribution(nil, nil)
	b.topologies = &grid.SessionGridTopologyStorage{
		CurrentTopology: sgte,
	}

	updated := networkbridgeevents.UpdatedAuthorityIDs{
		PeerID: peer.ID("peer2"),
		AuthorityDiscoveryIDs: []parachaintypes.AuthorityDiscoveryID{
			{1},
		},
	}
	err := b.processUpdatedAuthorityIDsEvent(updated)
	assert.Nil(t, err)

	updated = networkbridgeevents.UpdatedAuthorityIDs{
		PeerID: peer.ID("peer10"),
		AuthorityDiscoveryIDs: []parachaintypes.AuthorityDiscoveryID{
			{2},
		},
	}
	err = b.processUpdatedAuthorityIDsEvent(updated)
	assert.NotNil(t, err)
}

func TestBitfieldDistribution_ProcessActiveLeavesUpdateSignal_activatedLeafIsNil(t *testing.T) {
	signal := parachaintypes.ActiveLeavesUpdateSignal{
		Activated: nil,
	}

	b := NewBitfieldDistribution(nil, nil)
	err := b.ProcessActiveLeavesUpdateSignal(signal)
	assert.NoError(t, err)
}

func TestBitfieldDistribution_ProcessActiveLeavesUpdateSignal_QueryRuntimeError(t *testing.T) {
	ctrl := gomock.NewController(t)
	blockAPIMock := NewMockBlockState(ctrl)

	blockAPIMock.EXPECT().GetRuntime(common.Hash{0x01}).Return(
		nil, errors.New("something is off")).Times(1)

	signal := parachaintypes.ActiveLeavesUpdateSignal{
		Activated: &parachaintypes.ActivatedLeaf{
			Hash: common.Hash{0x01},
		},
	}

	b := NewBitfieldDistribution(nil, blockAPIMock)
	err := b.ProcessActiveLeavesUpdateSignal(signal)
	assert.EqualError(t, err, "something is off")
}

func TestBitfieldDistribution_ProcessActiveLeavesUpdateSignal_QueryParachainHostValidatorsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	blockAPIMock := NewMockBlockState(ctrl)
	runtimeMock := NewMockInstance(ctrl)

	blockAPIMock.EXPECT().GetRuntime(common.Hash{0x01}).Return(runtimeMock, nil).Times(1)
	runtimeMock.EXPECT().ParachainHostValidators().Return(nil, errors.New("something is off")).Times(1)

	signal := parachaintypes.ActiveLeavesUpdateSignal{
		Activated: &parachaintypes.ActivatedLeaf{
			Hash: common.Hash{0x01},
		},
	}

	b := NewBitfieldDistribution(nil, blockAPIMock)
	err := b.ProcessActiveLeavesUpdateSignal(signal)
	assert.EqualError(t, err, "something is off")
}

func TestBitfieldDistribution_ProcessActiveLeavesUpdateSignal_QueryParachainHostSessionIndexForChildError(
	t *testing.T,
) {
	ctrl := gomock.NewController(t)
	blockAPIMock := NewMockBlockState(ctrl)
	runtimeMock := NewMockInstance(ctrl)

	validatorSet := []parachaintypes.ValidatorID{
		[sr25519.PublicKeyLength]byte{1},
		[sr25519.PublicKeyLength]byte{2},
	}

	blockAPIMock.EXPECT().GetRuntime(common.Hash{0x01}).Return(runtimeMock, nil).Times(1)
	runtimeMock.EXPECT().ParachainHostValidators().Return(validatorSet, nil).Times(1)
	runtimeMock.EXPECT().ParachainHostSessionIndexForChild().Return(
		parachaintypes.SessionIndex(0), errors.New("something is off")).Times(1)

	signal := parachaintypes.ActiveLeavesUpdateSignal{
		Activated: &parachaintypes.ActivatedLeaf{
			Hash: common.Hash{0x01},
		},
	}

	b := NewBitfieldDistribution(nil, blockAPIMock)
	err := b.ProcessActiveLeavesUpdateSignal(signal)
	assert.EqualError(t, err, "something is off")
}

func TestBitfieldDistribution_ProcessActiveLeavesUpdateSignal_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	blockAPIMock := NewMockBlockState(ctrl)
	runtimeMock := NewMockInstance(ctrl)

	validatorSet := []parachaintypes.ValidatorID{
		[sr25519.PublicKeyLength]byte{1},
		[sr25519.PublicKeyLength]byte{2},
	}

	blockAPIMock.EXPECT().GetRuntime(common.Hash{0x01}).Return(runtimeMock, nil).Times(1)
	runtimeMock.EXPECT().ParachainHostValidators().Return(validatorSet, nil).Times(1)
	runtimeMock.EXPECT().ParachainHostSessionIndexForChild().Return(
		parachaintypes.SessionIndex(1), nil).Times(1)

	signal := parachaintypes.ActiveLeavesUpdateSignal{
		Activated: &parachaintypes.ActivatedLeaf{
			Hash: common.Hash{0x01},
		},
	}

	b := NewBitfieldDistribution(nil, blockAPIMock)

	// perRelayParent for relayParent 0x01 should be nil at this moment
	assert.Nil(t, nil, b.perRelayParent[common.Hash{0x01}])

	err := b.ProcessActiveLeavesUpdateSignal(signal)
	assert.NoError(t, err)

	signingContext := &parachaintypes.SigningContext{
		SessionIndex: parachaintypes.SessionIndex(1),
		ParentHash:   common.Hash{0x01},
	}

	// perRelayParent for relayParent 0x01 should be not nil at this moment
	assert.EqualValues(t, newPerRelayParentData(*signingContext, validatorSet), b.perRelayParent[common.Hash{0x01}])
}
