package bitfielddistribution

import (
	"fmt"
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

	jobData := newPerRelayParentData(parachaintypes.SessionIndex(1), validatorSet)

	// control the peer protocol version here and the peer's Head
	peerViews := map[peer.ID]struct {
		view            parachaintypes.View
		protocolVersion uint32
	}{
		peerA: {
			view:            parachaintypes.View{},
			protocolVersion: 2,
		},
		peerB: {
			view:            parachaintypes.View{},
			protocolVersion: 3,
		},
	}

	subSystemToOverseer := make(chan any)
	b := &BitfieldDistribution{
		subSystemToOverseer: subSystemToOverseer,
		peerViews:           peerViews,
		ourView:             parachaintypes.View{},
		topologies:          grid.SessionGridTopologyStorage{},
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

	// call the target method
	relayMessage(jobData, topologies, b.peerViews, validatorID, message, requiredRouting, b.subSystemToOverseer)

	// there should be empty interested peers so method call is returned without error
	// check the overseer chan message for the ProvisionableDataBitfield message
	go func() {
		for {
			request := <-subSystemToOverseer
			switch request := request.(type) {
			case provisionermessages.ProvisionableDataBitfield:
				assert.EqualValues(t, message.Hash, request.RelayParent)
				assert.EqualValues(t, message.CheckedSignedAvailabilityBitfield, request.Bitfield)
			}
		}
	}()

	// wait for the provisionable content checks
	time.Sleep(1 * time.Second)
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

	jobData := newPerRelayParentData(parachaintypes.SessionIndex(1), validatorSet)

	// control the peer protocol version here and the peer's Head
	peerViews := map[peer.ID]struct {
		view            parachaintypes.View
		protocolVersion uint32
	}{
		peerA: {
			view: parachaintypes.View{
				Heads: []common.Hash{relayParent}, // valid relayParent hash
			},
			protocolVersion: 2, // V2
		},
		peerB: {
			view:            parachaintypes.View{},
			protocolVersion: 3, //V3
		},
	}

	subSystemToOverseer := make(chan any)
	b := &BitfieldDistribution{
		subSystemToOverseer: subSystemToOverseer,
		peerViews:           peerViews,
		ourView:             parachaintypes.View{},
		topologies:          grid.SessionGridTopologyStorage{},
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

	// call the target method
	relayMessage(jobData, topologies, b.peerViews, validatorID, message, requiredRouting, b.subSystemToOverseer)

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
			}
		}
	}()

	// wait for the v2 content SendValidationMessage checks
	time.Sleep(1 * time.Second)
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

	jobData := newPerRelayParentData(parachaintypes.SessionIndex(1), validatorSet)

	// control the peer protocol version here and the peer's Head
	peerViews := map[peer.ID]struct {
		view            parachaintypes.View
		protocolVersion uint32
	}{
		peerA: {
			view:            parachaintypes.View{},
			protocolVersion: 2,
		},
		peerB: {
			view: parachaintypes.View{
				Heads: []common.Hash{relayParent},
			},
			protocolVersion: 3,
		},
	}

	subSystemToOverseer := make(chan any)
	b := &BitfieldDistribution{
		subSystemToOverseer: subSystemToOverseer,
		peerViews:           peerViews,
		ourView:             parachaintypes.View{},
		topologies:          grid.SessionGridTopologyStorage{},
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

	// call the target method
	relayMessage(jobData, topologies, b.peerViews, validatorID, message, requiredRouting, b.subSystemToOverseer)

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
			}
		}
	}()

	// wait for the v3 content SendValidationMessage checks
	time.Sleep(1 * time.Second)
}

func TestBitfieldDistribution_ProcessBitfieldDistributionMessageSignal_UnexpectedMessageType(t *testing.T) {
	overseerCh := make(chan any)
	b := NewBitfieldDistribution(overseerCh)

	message := validationprotocol.CheckedBitfield{}
	bdm := validationprotocol.BitfieldDistributionMessage{}
	err := bdm.SetValue(message)
	assert.Nil(t, err)

	err = b.ProcessBitfieldDistributionMessageSignal(bdm)
	assert.NotNil(t, err)
}

func TestBitfieldDistribution_ProcessBitfieldDistributionMessageSignal_NoRelayParentToWorkOn(t *testing.T) {
	overseerCh := make(chan any)
	b := NewBitfieldDistribution(overseerCh)

	message := validationprotocol.UncheckedBitfield{
		Hash: common.Hash{1, 2, 3},
	}
	bdm := validationprotocol.BitfieldDistributionMessage{}
	err := bdm.SetValue(message)
	assert.Nil(t, err)

	// not supposed to work on relay parent related data
	err = b.ProcessBitfieldDistributionMessageSignal(bdm)
	assert.Nil(t, err)
}

func TestBitfieldDistribution_ProcessBitfieldDistributionMessageSignal_ValidatorSetEmpty(t *testing.T) {
	jobData := newPerRelayParentData(parachaintypes.SessionIndex(1), nil)
	peerViews := map[peer.ID]struct {
		view            parachaintypes.View
		protocolVersion uint32
	}{}
	overseerCh := make(chan any)
	b := &BitfieldDistribution{
		subSystemToOverseer: overseerCh,
		peerViews:           peerViews,
		ourView:             parachaintypes.View{},
		topologies:          grid.SessionGridTopologyStorage{},
		perRelayParent: map[common.Hash]*perRelayParentData{
			common.Hash{1, 2, 3}: jobData,
		},
		reputation: util.NewReputationAggregator(func(rep util.UnifiedReputationChange) bool {
			return false
		}),
	}

	message := validationprotocol.UncheckedBitfield{
		Hash: common.Hash{1, 2, 3},
	}
	bdm := validationprotocol.BitfieldDistributionMessage{}
	err := bdm.SetValue(message)
	assert.Nil(t, err)

	// validator set is empty
	err = b.ProcessBitfieldDistributionMessageSignal(bdm)
	assert.Nil(t, err)
}

func TestBitfieldDistribution_ProcessBitfieldDistributionMessageSignal_ValidatorIdxInvalid(t *testing.T) {
	validatorIndex, validatorSet := prepareForValidators()
	jobData := newPerRelayParentData(parachaintypes.SessionIndex(1), validatorSet)
	peerViews := map[peer.ID]struct {
		view            parachaintypes.View
		protocolVersion uint32
	}{}
	overseerCh := make(chan any)
	b := &BitfieldDistribution{
		subSystemToOverseer: overseerCh,
		peerViews:           peerViews,
		ourView:             parachaintypes.View{},
		topologies:          grid.SessionGridTopologyStorage{},
		perRelayParent: map[common.Hash]*perRelayParentData{
			common.Hash{1, 2, 3}: jobData,
		},
		reputation: util.NewReputationAggregator(func(rep util.UnifiedReputationChange) bool {
			return false
		}),
	}

	message := validationprotocol.UncheckedBitfield{
		Hash: common.Hash{1, 2, 3},
		UncheckedSignedAvailabilityBitfield: parachaintypes.UncheckedSignedAvailabilityBitfield{
			ValidatorIndex: validatorIndex + 10, // make them different
		},
	}
	bdm := validationprotocol.BitfieldDistributionMessage{}
	err := bdm.SetValue(message)
	assert.Nil(t, err)

	// validator set is empty
	err = b.ProcessBitfieldDistributionMessageSignal(bdm)
	assert.Nil(t, err)
}

func TestBitfieldDistribution_ProcessBitfieldDistributionMessageSignal_CheckSignedAvailabilityBitfield(t *testing.T) {
	validatorIndex := 0

	keyring, err := keystore.NewSr25519Keyring()
	assert.Nil(t, err)
	aliceKeypair := keyring.Alice().(*sr25519.Keypair)

	validatorSet := []parachaintypes.ValidatorID{
		parachaintypes.ValidatorID(aliceKeypair.Public().Encode()),
		[sr25519.PublicKeyLength]byte{2},
	}
	jobData := newPerRelayParentData(parachaintypes.SessionIndex(1), validatorSet)
	peerViews := map[peer.ID]struct {
		view            parachaintypes.View
		protocolVersion uint32
	}{}

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
		topologies: grid.SessionGridTopologyStorage{
			CurrentTopology: sgte,
			PrevTopology:    sgte,
		},
		perRelayParent: map[common.Hash]*perRelayParentData{
			common.Hash{1, 2, 3}: jobData,
		},
		reputation: util.NewReputationAggregator(func(rep util.UnifiedReputationChange) bool {
			return false
		}),
	}

	bitfield, err := parachaintypes.NewBitVec([]bool{true, true, false})
	assert.Nil(t, err)
	data, err := bitfield.MarshalSCALE()
	assert.Nil(t, err)

	fmt.Println("data: ", data)
	signature, err := aliceKeypair.Sign(data)
	assert.Nil(t, err)

	message := validationprotocol.UncheckedBitfield{
		Hash: common.Hash{1, 2, 3},
		UncheckedSignedAvailabilityBitfield: parachaintypes.UncheckedSignedAvailabilityBitfield{
			Payload:        bitfield,
			ValidatorIndex: parachaintypes.ValidatorIndex(validatorIndex),
			Signature:      parachaintypes.ValidatorSignature(signature),
		},
	}
	bdm := validationprotocol.BitfieldDistributionMessage{}
	err = bdm.SetValue(message)
	assert.Nil(t, err)
	err = b.ProcessBitfieldDistributionMessageSignal(bdm)
	assert.Nil(t, err)
}
