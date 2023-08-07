package rx

import (
	"fmt"
	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/lib/parachain"
	"github.com/ChainSafe/gossamer/lib/parachain/network/bridge"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/libp2p/go-libp2p/core"
)

func makeSyncOracle() (*Oracle, *sync.WaitGroup) {
	return &Oracle{}, &sync.WaitGroup{}
}

func (o *Oracle) awaitModeSwitch() {}

// NetworkHandle is a simple type representing a network handle
type NetworkHandle struct{}

func (nh *NetworkHandle) connectPeer(peer core.PeerID, peerSet, observedRole string) {}

func (nh *NetworkHandle) nextNetworkActions(count int) []string {
	return nil
}

// VirtualOverseer is a simple type representing a virtual overseer
type VirtualOverseer struct{}

func (vo *VirtualOverseer) send(signal OverseerSignal) {}

// TestHarness represents a test harness
type TestHarness struct {
	networkHandle   NetworkHandle
	virtualOverseer VirtualOverseer
	shared          bridge.Shared
}

type TestHarnessFn func(*TestHarness) VirtualOverseer

func newTestNetwork(t *testing.T) parachain.PNetwork {
	t.Helper()
	//ctrl := gomock.NewController(t)
	mockNetwork, err := parachain.NewService(nil, "", common.Hash{})
	require.NoError(t, err)
	//mockNetwork := parachain.NewMockPNetwork(ctrl)
	//if cfg == nil {
	cfg := &network.Config{
		BasePath:    t.TempDir(),
		Port:        7500,
		NoBootstrap: true,
		NoMDNS:      true,
		//LogLvl:       log.Warn,
		SlotDuration: time.Second,
	}
	//}

	if cfg.BlockState == nil {
		//header := &types.Header{
		//	ParentHash:     common.Hash{},
		//	Number:         1,
		//	StateRoot:      common.Hash{},
		//	ExtrinsicsRoot: common.Hash{},
		//	Digest:         types.NewDigest(),
		//}
		//
		//blockstate := NewMockBlockState(ctrl)
		//
		//blockstate.EXPECT().BestBlockHeader().Return(header, nil).AnyTimes()
		//blockstate.EXPECT().GetHighestFinalisedHeader().Return(header, nil).AnyTimes()
		//blockstate.EXPECT().GenesisHash().Return(common.NewHash([]byte{})).AnyTimes()
		//
		//cfg.BlockState = blockstate
	}

	if cfg.TransactionHandler == nil {
		//th := NewMockTransactionHandler(ctrl)
		//th.EXPECT().
		//	HandleTransactionMessage(
		//		gomock.AssignableToTypeOf(peer.ID("")),
		//		gomock.Any()).
		//	Return(true, nil).AnyTimes()
		//
		//th.EXPECT().TransactionsCount().Return(0).AnyTimes()
		//cfg.TransactionHandler = th
	}

	cfg.SlotDuration = time.Second
	//cfg.ProtocolID = TestProtocolID // default "/gossamer/gssmr/0"

	if cfg.LogLvl == 0 {
		cfg.LogLvl = 4
	}

	if cfg.Syncer == nil {
		//syncer := NewMockSyncer(ctrl)
		//syncer.EXPECT().
		//	HandleBlockAnnounceHandshake(
		//		gomock.AssignableToTypeOf(peer.ID("")), gomock.Any()).
		//	Return(nil).AnyTimes()
		//
		//syncer.EXPECT().
		//	HandleBlockAnnounce(
		//		gomock.AssignableToTypeOf(peer.ID("")), gomock.Any()).
		//	Return(nil).AnyTimes()
		//
		//syncer.EXPECT().
		//	CreateBlockResponse(gomock.Any()).
		//	Return(newTestBlockResponseMessage(t), nil).AnyTimes()
		//
		//syncer.EXPECT().IsSynced().Return(false).AnyTimes()
		//cfg.Syncer = syncer
	}

	if cfg.Telemetry == nil {
		//telemetryMock := NewMockTelemetry(ctrl)
		//telemetryMock.EXPECT().SendMessage(gomock.Any()).AnyTimes()
		//cfg.Telemetry = telemetryMock
	}

	srvc, err := network.NewService(cfg)
	require.NoError(t, err)

	//srvc.noDiscover = true

	err = srvc.Start()
	require.NoError(t, err)

	t.Cleanup(func() {
		err := srvc.Stop()
		require.NoError(t, err)
	})

	return mockNetwork
}

func testHarness(t *testing.T, oracle *Oracle, testFunc TestHarnessFn) {
	var wg sync.WaitGroup
	wg.Add(1)
	network := newTestNetwork(t)
	shared := bridge.Shared{
		SharedInner: bridge.SharedInner{},
	}

	testFunc(&TestHarness{
		networkHandle:   NetworkHandle{},
		virtualOverseer: VirtualOverseer{},
		shared:          shared,
	})
	bridge := NetworkBridgeRx{
		networkService: network,
		SyncOracle:     oracle,
		Shared:         &shared,
	}

	go func() {
		defer wg.Done()
		runNetworkIn(bridge)
	}()

	wg.Wait()
}

func TestSendOurViewUponConnection(t *testing.T) {
	oracle, handle := makeSyncOracle()

	testHarness(t, oracle, func(testHarness *TestHarness) VirtualOverseer {
		networkHandle := testHarness.networkHandle
		virtualOverseer := testHarness.virtualOverseer

		peer := core.PeerID("random-peer-id")
		head := common.Hash{1, 2, 3}
		virtualOverseer.send(OverseerSignal{
			ActiveLeaves: ActiveLeavesUpdate{
				ActivatedLeaf: ActivatedLeaf{
					hash:   head,
					number: 1,
					status: LeafStatus("Fresh"),
				},
			},
		})

		handle.Wait()

		networkHandle.connectPeer(peer, "Validation", "Full")
		networkHandle.connectPeer(peer, "Collation", "Full")
		fmt.Printf("awaitng peer connect\n")
		// TODO:  implementation of awaitPeerConnections
		//awaitPeerConnections(&testHarness.shared, 1, 1)
		fmt.Printf("past peer connect \n")
		view := []common.Hash{head}
		actions := networkHandle.nextNetworkActions(2)

		assertNetworkActionsContains(
			&actions,
			&NetworkAction{
				Peer:    peer,
				PeerSet: peerset.PeerSet{},
				WireMsg: EncodeViewUpdate(view),
			},
		)

		assertNetworkActionsContains(
			&actions,
			&NetworkAction{
				Peer:    peer,
				PeerSet: peerset.PeerSet{},
				WireMsg: EncodeViewUpdate(view),
			},
		)

		virtualOverseer.send(OverseerSignal{})
		return virtualOverseer
	})
}

// EncodeViewUpdate encodes the view update
func EncodeViewUpdate(view []common.Hash) string {
	// TODO: add actual encoding
	return fmt.Sprintf("%v", view)
}

func awaitPeerConnections(shared *bridge.Shared, numValidationPeers int, numCollationPeers int) {
	for {
		if len(shared.SharedInner.ValidationPeers) == numValidationPeers && len(shared.SharedInner.
			CollationPeers) == numCollationPeers {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// assertNetworkActionsContains is a helper function to assert that a network action is present
func assertNetworkActionsContains(actions *[]string, action *NetworkAction) {
	// TODO: implement this
	fmt.Printf("Actions %v, action: %v\n", actions, action)
}
