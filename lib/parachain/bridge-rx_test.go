package parachain

import (
	"fmt"
	"github.com/ChainSafe/gossamer/lib/parachain/network/bridge"
	"github.com/golang/mock/gomock"
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
type TestNetworkHandle struct {
	//actionRx chan NetworkAction
	networkStream  *EventStream
	networkActions []*NetworkAction
}

func (nh *TestNetworkHandle) connectPeer(peer core.PeerID, peerSet, observedRole string) {
	item := &Item{
		Value: len(peerSet),
	}
	nh.networkStream.AddItem(item)
	fmt.Printf("Added ITem %v\n", item)
	action := &NetworkAction{
		Peer: peer,
		//PeerSet: peerSet,
		WireMsg: observedRole,
	}
	nh.networkActions = append(nh.networkActions, action)
}

func (nh *TestNetworkHandle) nextNetworkActions(count int) []NetworkAction {
	actions := []NetworkAction{}
	for i := 0; i < count; i++ {
		actions = append(actions, *nh.networkActions[len(nh.networkActions)-i-1])
	}
	return actions
}

// VirtualOverseer is a simple type representing a virtual overseer
type VirtualOverseer struct{}

func (vo *VirtualOverseer) send(signal OverseerSignal) {
	fmt.Printf("Got overseer signal: %v\n", signal)
}

// TestHarness represents a test harness
type TestHarness struct {
	networkHandle   TestNetworkHandle
	virtualOverseer VirtualOverseer
	shared          bridge.Shared
}

type TestHarnessFn func(*TestHarness) VirtualOverseer

func newTestNetwork(t *testing.T) Network {
	t.Helper()
	ctrl := gomock.NewController(t)
	mockNetwork := NewMockNetwork(ctrl)

	return mockNetwork
}

func testHarness(t *testing.T, oracle *Oracle, testFunc TestHarnessFn) {
	var wg sync.WaitGroup
	wg.Add(1)
	network := newTestNetwork(t)
	shared := &bridge.Shared{
		SharedInner: bridge.SharedInner{
			LocalView:       bridge.View{},
			ValidationPeers: make(map[core.PeerID]bridge.PeerData),
			CollationPeers:  make(map[core.PeerID]bridge.PeerData),
		},
	}
	networkStream := NewEventStream(shared)

	bridge := NetworkBridgeRx{
		networkService: network,
		SyncOracle:     oracle,
		Shared:         shared,
	}

	go func() {
		defer wg.Done()
		runNetworkIn(bridge, networkStream)
	}()
	time.Sleep(time.Second)
	networkHandle := TestNetworkHandle{networkStream: networkStream}
	virtualOverseer := VirtualOverseer{}
	testHarness := &TestHarness{
		networkHandle:   networkHandle,
		virtualOverseer: virtualOverseer,
		shared:          *shared,
	}

	virtualOverseer = testFunc(testHarness)
	wg.Wait()
}

func TestSendOurViewUponConnection(t *testing.T) {
	// - sync oracle is started (as false syncing) once started syncing the handle will be done waiting
	// - overseer sends ActivatedLeaf signal to all subsystems to start working on active leaf
	// - peer is connected (determine where to capture this event)
	//   - send peer current view
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
		time.Sleep(time.Millisecond)
		networkHandle.connectPeer(peer, "Collation", "Full")
		fmt.Printf("awaitng peer connect\n")
		// TODO:  implementation of awaitPeerConnections
		//awaitPeerConnections(&testHarness.shared, 1, 1)
		fmt.Printf("past peer connect \n")
		view := []common.Hash{head}
		time.Sleep(time.Second)
		actions := networkHandle.nextNetworkActions(2)

		assertNetworkActionsContains(
			actions,
			&NetworkAction{
				Peer:    peer,
				PeerSet: peerset.PeerSet{},
				WireMsg: EncodeViewUpdate(view),
			},
		)

		assertNetworkActionsContains(
			actions,
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
func assertNetworkActionsContains(actions []NetworkAction, action *NetworkAction) {
	// TODO: implement this
	for i, networkAction := range actions {
		fmt.Printf("Action %v, action: %v\n\tcompaire action %v\n", i, networkAction, action)
	}
}
