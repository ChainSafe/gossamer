package rx

import (
	"fmt"
	"sync"
	"testing"

	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/libp2p/go-libp2p/core"
)

// Oracle is a simple type representing an oracle
type Oracle struct{}

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
	shared          interface{} // TODO: determine actual shared data type
}

func testHarness(oracle *Oracle, testFunc func(*TestHarness)) {
	var wg sync.WaitGroup
	testFunc(&TestHarness{
		networkHandle:   NetworkHandle{},
		virtualOverseer: VirtualOverseer{},
		shared:          nil, // TODO: Replace with the actual shared data
	})
	wg.Wait()
}

func TestSendOurViewUponConnection(t *testing.T) {
	oracle, handle := makeSyncOracle()

	testHarness(oracle, func(testHarness *TestHarness) {
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

		// TODO:  implementation of awaitPeerConnections
		//awaitPeerConnections(&testHarness.shared, 1, 1)

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
	})
}

// EncodeViewUpdate encodes the view update
func EncodeViewUpdate(view []common.Hash) string {
	// TODO: add actual encoding
	return fmt.Sprintf("%v", view)
}

// assertNetworkActionsContains is a helper function to assert that a network action is present
func assertNetworkActionsContains(actions *[]string, action *NetworkAction) {
	// TODO: implement this
	fmt.Printf("Actions %v, action: %v\n", actions, action)
}
