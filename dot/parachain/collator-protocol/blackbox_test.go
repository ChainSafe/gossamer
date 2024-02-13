package collatorprotocol

import (
	"context"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	overseer "github.com/ChainSafe/gossamer/dot/parachain/overseer"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/log"
	libp2phost "github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
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

func testConnectAndDeclareCollator(t *testing.T) {

}

func testAdvertiseCollation(t *testing.T) {

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

func testCreateCollatorValidatorPair(t *testing.T) {
	t.Parallel()

	// create networks for collator and validator
	config := &network.Config{
		BasePath:    t.TempDir(),
		Port:        availablePort(t),
		NoBootstrap: true,
		NoMDNS:      true,
	}
	validatorNode := createTestService(t, config)

	configB := &network.Config{
		BasePath:    t.TempDir(),
		Port:        availablePort(t),
		NoBootstrap: true,
		NoMDNS:      true,
	}
	collatorNode := createTestService(t, configB)

	addrInfoB := addrInfo(collatorNode.GetP2PHost())
	err := validatorNode.Connect(addrInfoB)
	require.NoError(t, err)

	ctx := context.Background()
	collationProtocolID := "/6761727661676500000000000000000000000000000000000000000000000000/1/collations/1"
	stream, err := validatorNode.GetP2PHost().NewStream(ctx, collatorNode.GetP2PHost().ID(), validatorNode.ProtocolID()+protocol.ID(collationProtocolID))
	require.NoError(t, err)

	// create overseer
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockBlockState := NewMockBlockState(ctrl)
	finalizedNotifierChan := make(chan *types.FinalisationInfo)
	importedBlockNotiferChan := make(chan *types.Block)

	mockBlockState.EXPECT().GetFinalisedNotifierChannel().Return(finalizedNotifierChan)
	mockBlockState.EXPECT().GetImportedBlockNotifierChannel().Return(importedBlockNotiferChan)
	mockBlockState.EXPECT().FreeFinalisedNotifierChannel(finalizedNotifierChan)
	mockBlockState.EXPECT().FreeImportedBlockNotifierChannel(importedBlockNotiferChan)

	overseer := overseer.NewOverseer(mockBlockState)
	err = overseer.Start()
	require.NoError(t, err)

	defer overseer.Stop()

	cpvs, err := Register(validatorNode, protocol.ID(collationProtocolID), overseer)
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

	collatorNode.SendMessage(validatorNode.GetP2PHost().ID(), &collationMessage)
	require.NoError(t, err)

	// NOTE TO SELF : visit TestHandleLightMessage_Response
}

var TestProtocolID = "/gossamer/test/0"

// helper method to create and start a new network service
func createTestService(t *testing.T, cfg *network.Config) (srvc *network.Service) {
	t.Helper()
	// ctrl := gomock.NewController(t)

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
