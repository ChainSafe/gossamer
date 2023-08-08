package parachain

import (
	"fmt"
	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/parachain/network/bridge"
	"github.com/libp2p/go-libp2p/core"
	"sync"
	"time"
)

// LeafStatus is a simple type representing a leaf status
type LeafStatus string

// ActivatedLeaf represents an activated leaf
type ActivatedLeaf struct {
	hash   common.Hash
	number int
	status LeafStatus
}

// ActiveLeavesUpdate represents an active leaves update
type ActiveLeavesUpdate struct {
	ActivatedLeaf
}

// OverseerSignal represents an overseer signal
type OverseerSignal struct {
	ActiveLeaves ActiveLeavesUpdate
}

type Item struct {
	Value int
}

type EventStream struct {
	mutex   sync.Mutex
	items   []*Item
	recvCh  chan *Item
	closeCh chan struct{}
	shared  *bridge.Shared
}

func NewEventStream(shared *bridge.Shared) *EventStream {
	return &EventStream{
		recvCh:  make(chan *Item),
		closeCh: make(chan struct{}),
		shared:  shared,
	}
}

func (tes *EventStream) AddItem(item *Item) {
	tes.mutex.Lock()
	defer tes.mutex.Unlock()

	tes.items = append(tes.items, item)
	select {
	case tes.recvCh <- item:
	default:
	}
}

func (tes *EventStream) Close() {
	close(tes.closeCh)
}

func (tes *EventStream) StartConsuming() {
	go func() {
		defer close(tes.recvCh)
		for {
			select {
			case <-tes.closeCh:
				return
			case item, ok := <-tes.recvCh:
				if !ok {
					return
				}
				tes.shared.ValidationPeers[core.PeerID(fmt.Sprintf("%v", item.Value))] = bridge.PeerData{}
				fmt.Printf("Received: %v\n", item.Value)
			}
		}
	}()
}

// NetworkAction represents a network action
//
//	TODO: this is a place holder, replace with variable data type
type NetworkAction struct {
	Peer    core.PeerID
	PeerSet peerset.PeerSet
	WireMsg string
}

// Oracle is a simple type representing an oracle
type Oracle struct{}

type NetworkBridgeRx struct {
	networkService Network
	SyncOracle     *Oracle
	Shared         *bridge.Shared
}

func runNetworkIn(bridge NetworkBridgeRx, networkStream *EventStream) {
	networkStream.StartConsuming()

	for i := 0; i < 5; i++ {
		fmt.Printf("In Run Network i: %v\n", i)
		fmt.Printf("bridge syncOrcale: %v\nbridge Shared: %v\n", bridge.SyncOracle, len(bridge.Shared.ValidationPeers))
		time.Sleep(time.Second)
	}
}
