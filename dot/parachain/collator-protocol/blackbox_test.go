package collatorprotocol

import (
	"context"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	mock_network "github.com/ChainSafe/gossamer/dot/parachain/collator-protocol/mock-network"
	overseer "github.com/ChainSafe/gossamer/dot/parachain/overseer"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	libp2phost "github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	protocol "github.com/libp2p/go-libp2p/core/protocol"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// func TestBackedMessage(t *testing.T) {
// 	t.Parallel()
// 	ctrl := gomock.NewController(t)
// 	defer ctrl.Finish()

// 	o := NewMockOverseerI(ctrl)
// 	overseerToSubsystem := make(chan any)

// 	// o.EXPECT().Send(backed).Do(func() {
// 	// 	overseerToSubsystem <- backed
// 	// })

// 	cpvs := CollatorProtocolValidatorSide{
// 		// net: c.net,
// 		// SubSystemToOverseer: make(chan<- any),
// 		overseer:            o,
// 		OverseerToSubSystem: overseerToSubsystem,
// 		// perRelayParent: c.perRelayParent,
// 		// fetchedCandidates:     c.fetchedCandidates,
// 		// peerData:              c.peerData,
// 		// BlockedAdvertisements: c.blockedAdvertisements,
// 		// activeLeaves:   c.activeLeaves,
// 	}

// 	// Test 1: What happens when collator protocol receives a Backed message

// 	// Overseer sends a Backed message
// 	backed := collatorprotocolmessages.Backed{}
// 	overseerToSubsystem <- backed

// 	// A backed message tells collator protocol that given candidate was backed.
// 	// we should check for blocked advertisements associated with this backed message
// 	// if so,, we should remove the blocked advertisement and request unblocked collations
// 	// lenFetchedCandidatesBefore := len(cpvs.fetchedCandidates)

// 	// err := cpvs.processMessage(c.msg)
// 	// if c.errString == "" {
// 	// 	require.NoError(t, err)
// 	// } else {
// 	// 	require.ErrorContains(t, err, c.errString)
// 	// }

// 	// if c.deletesFetchCandidate {
// 	// 	require.Equal(t, lenFetchedCandidatesBefore-1, len(cpvs.fetchedCandidates))
// 	// } else {
// 	// 	require.Equal(t, lenFetchedCandidatesBefore, len(cpvs.fetchedCandidates))
// 	// }

// }

func testSetup(t *testing.T) *MockOverseerI {
	t.Helper()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	o := NewMockOverseerI(ctrl)

	return o
}

// func TestBackedCandidateUnblocksAdvertisements(t *testing.T) {
// 	overseer := testSetup(t)

// 	testConnectAndDeclareCollator(t)

// 	testAdvertiseCollation(t)

// }

const portsAmount = 200

// portQueue is a blocking port queue
type portQueue chan uint16

func (pq portQueue) put(p uint16) {
	pq <- p
}

func (pq portQueue) get() (port uint16) {
	port = <-pq
	return port
}

var availablePorts portQueue

func init() {
	availablePorts = make(chan uint16, portsAmount)
	const startAt = uint16(7500)
	for port := startAt; port < portsAmount+startAt; port++ {
		availablePorts.put(port)
	}
}

// availablePort is test helper function that gets an available port and release the same port after test ends
func availablePort(t *testing.T) uint16 {
	t.Helper()
	port := availablePorts.get()

	t.Cleanup(func() {
		availablePorts.put(port)
	})

	return port
}

// addrInfo returns the libp2p peer.AddrInfo of the host
func addrInfo(p2pHost libp2phost.Host) peer.AddrInfo {
	return peer.AddrInfo{
		ID:    p2pHost.ID(),
		Addrs: p2pHost.Addrs(),
	}
}

func testCreateCollatorValidatorPair(t *testing.T) (*network.Service, *network.Service, *CollatorProtocolValidatorSide) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	// create networks for collator and validator
	config := &network.Config{
		BasePath:    t.TempDir(),
		Port:        availablePort(t),
		NoBootstrap: true,
		NoMDNS:      true,
		MinPeers:    1,
		MaxPeers:    2,
	}
	validatorNode := createTestService(t, config)
	addrInfoA := addrInfo(validatorNode.GetP2PHost())

	configB := &network.Config{
		BasePath:    t.TempDir(),
		Port:        availablePort(t),
		NoBootstrap: true,
		NoMDNS:      true,
		MinPeers:    1,
		MaxPeers:    2,
	}
	collatorNode := createTestService(t, configB)

	addrInfoB := addrInfo(collatorNode.GetP2PHost())
	validatorNode.GetP2PHost().Peerstore().AddAddrs(addrInfoB.ID, addrInfoB.Addrs, peerstore.PermanentAddrTTL)
	validatorNode.Connect(addrInfoB)

	collatorNode.GetP2PHost().Peerstore().AddAddrs(addrInfoA.ID, addrInfoA.Addrs, peerstore.PermanentAddrTTL)
	collatorNode.Connect(addrInfoA)

	time.Sleep(100 * time.Millisecond)

	collationProtocolID := "/6761727661676500000000000000000000000000000000000000000000000000/1/collations/1"

	// create overseer
	// ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockBlockState := NewMockBlockState(ctrl)
	finalizedNotifierChan := make(chan *types.FinalisationInfo)
	importedBlockNotiferChan := make(chan *types.Block)

	mockBlockState.EXPECT().GetFinalisedNotifierChannel().Return(finalizedNotifierChan)
	mockBlockState.EXPECT().GetImportedBlockNotifierChannel().Return(importedBlockNotiferChan)
	mockBlockState.EXPECT().FreeFinalisedNotifierChannel(finalizedNotifierChan)
	mockBlockState.EXPECT().FreeImportedBlockNotifierChannel(importedBlockNotiferChan)

	overseer := overseer.NewOverseer(mockBlockState)

	cpvs, err := Register(validatorNode, protocol.ID(collationProtocolID), overseer)
	require.NoError(t, err)
	overseer.RegisterSubsystem(cpvs)

	err = overseer.Start()
	require.NoError(t, err)

	defer overseer.Stop()

	_, err = RegisterCollatorSide(collatorNode, protocol.ID(collationProtocolID), nil)
	require.NoError(t, err)

	time.Sleep(1 * time.Second)
	ctx := context.Background()
	/*stream*/
	_, err = validatorNode.GetP2PHost().NewStream(
		ctx, collatorNode.GetP2PHost().ID(), protocol.ID(collationProtocolID))
	require.NoError(t, err)

	collatorProtocolMessage := NewCollatorProtocolMessage()
	err = collatorProtocolMessage.Set(Declare{
		// CollatorId: ,
		// ParaID:,
		// CollatorSignature,
	})
	require.NoError(t, err)
	collationMessage := NewCollationProtocol()

	err = collationMessage.Set(collatorProtocolMessage)
	require.NoError(t, err)

	err = collatorNode.SendMessage(validatorNode.GetP2PHost().ID(), &collationMessage)
	require.NoError(t, err)

	return collatorNode, validatorNode, cpvs
	// NOTE TO SELF : visit TestHandleLightMessage_Response
}

var TestProtocolID = "/gossamer/test/0"

// helper method to create and start a new network service
func createTestService(t *testing.T, cfg *network.Config) (srvc *network.Service) {
	t.Helper()
	ctrl := gomock.NewController(t)

	if cfg == nil {
		cfg = &network.Config{
			BasePath:     t.TempDir(),
			Port:         availablePort(t),
			NoBootstrap:  true,
			NoMDNS:       true,
			LogLvl:       log.Warn,
			SlotDuration: time.Second,
		}
	}
	if cfg.BlockState == nil {
		header := &types.Header{
			ParentHash:     common.Hash{},
			Number:         1,
			StateRoot:      common.Hash{},
			ExtrinsicsRoot: common.Hash{},
			Digest:         types.NewDigest(),
		}

		blockstate := mock_network.NewMockBlockState(ctrl)

		blockstate.EXPECT().BestBlockHeader().Return(header, nil).AnyTimes()
		blockstate.EXPECT().GetHighestFinalisedHeader().Return(header, nil).AnyTimes()
		blockstate.EXPECT().GenesisHash().Return(common.NewHash([]byte{})).AnyTimes()

		cfg.BlockState = blockstate
	}

	if cfg.TransactionHandler == nil {
		th := mock_network.NewMockTransactionHandler(ctrl)
		th.EXPECT().
			HandleTransactionMessage(
				gomock.AssignableToTypeOf(peer.ID("")),
				gomock.Any()).
			Return(true, nil).AnyTimes()

		th.EXPECT().TransactionsCount().Return(0).AnyTimes()
		cfg.TransactionHandler = th
	}

	if cfg.Syncer == nil {
		syncer := mock_network.NewMockSyncer(ctrl)
		syncer.EXPECT().
			HandleBlockAnnounceHandshake(
				gomock.AssignableToTypeOf(peer.ID("")), gomock.Any()).
			Return(nil).AnyTimes()

		syncer.EXPECT().
			HandleBlockAnnounce(
				gomock.AssignableToTypeOf(peer.ID("")), gomock.Any()).
			Return(nil).AnyTimes()

		// syncer.EXPECT().
		// 	CreateBlockResponse(gomock.Any()).
		// 	Return(newTestBlockResponseMessage(t), nil).AnyTimes()

		syncer.EXPECT().IsSynced().Return(false).AnyTimes()
		cfg.Syncer = syncer
	}

	if cfg.Telemetry == nil {
		telemetryMock := mock_network.NewMockTelemetry(ctrl)
		telemetryMock.EXPECT().SendMessage(gomock.Any()).AnyTimes()
		cfg.Telemetry = telemetryMock
	}

	cfg.SlotDuration = time.Second
	cfg.ProtocolID = TestProtocolID // default "/gossamer/gssmr/0"

	if cfg.LogLvl == 0 {
		cfg.LogLvl = 4
	}

	srvc, err := network.NewService(cfg)
	require.NoError(t, err)

	// srvc.noDiscover = true

	err = srvc.Start()
	require.NoError(t, err)

	t.Cleanup(func() {
		err := srvc.Stop()
		require.NoError(t, err)
	})
	return srvc
}

func TestSomething(t *testing.T) {
	testCreateCollatorValidatorPair(t)
}

// TestCollatorDeclare tests the happy path on receiving a valid Declare message. The happy path
// means that
// 1) Declare message has a valid signature and
// 2) it's para id is a value that we are assigned to.
// In this case, not down the that collator's peer ID, collator ID and set it as collating.
func TestCollatorDeclare(t *testing.T) {
	collatorNode, validatorNode, _ := testCreateCollatorValidatorPair(t)

	testDeclare(t, collatorNode, validatorNode)
	testAdvertiseCollation(t, collatorNode, validatorNode)
	// declare done

}

func testAdvertiseCollation(t *testing.T, collatorNode *network.Service, validatorNode *network.Service) {
	testRelayParent := getDummyHash(5)

	collatorProtocolMessage := NewCollatorProtocolMessage()
	err := collatorProtocolMessage.Set(AdvertiseCollation(testRelayParent))
	require.NoError(t, err)
	collationMessage := NewCollationProtocol()

	err = collationMessage.Set(collatorProtocolMessage)
	require.NoError(t, err)

	err = collatorNode.SendMessage(validatorNode.GetP2PHost().ID(), &collationMessage)
	require.NoError(t, err)

}

func testDeclare(t *testing.T, collatorNode *network.Service, validatorNode *network.Service) {
	collatorKeypair, err := sr25519.GenerateKeypair()
	require.NoError(t, err)
	collatorID, err := sr25519.NewPublicKey(collatorKeypair.Public().Encode())
	require.NoError(t, err)

	payload := getDeclareSignaturePayload(collatorNode.GetP2PHost().ID())
	signatureBytes, err := collatorKeypair.Sign(payload)
	require.NoError(t, err)

	signature := [sr25519.SignatureLength]byte{}
	copy(signature[:], signatureBytes)

	collatorProtocolMessage := NewCollatorProtocolMessage()
	err = collatorProtocolMessage.Set(Declare{
		CollatorId:        parachaintypes.CollatorID(collatorID.AsBytes()),
		ParaID:            1000,
		CollatorSignature: parachaintypes.CollatorSignature(signature),
	})
	require.NoError(t, err)
	collationMessage := NewCollationProtocol()

	err = collationMessage.Set(collatorProtocolMessage)
	require.NoError(t, err)

	err = collatorNode.SendMessage(validatorNode.GetP2PHost().ID(), &collationMessage)
	require.NoError(t, err)
}
