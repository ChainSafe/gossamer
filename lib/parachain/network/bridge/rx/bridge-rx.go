package rx

import (
	"fmt"
	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/parachain/network/bridge"
	"github.com/libp2p/go-libp2p/core"
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
	SyncOracle *Oracle
	Shared     *bridge.Shared
}

func runNetworkIn(bridge NetworkBridgeRx) {
	for i := 0; i < 5; i++ {
		fmt.Printf("In Run Network i: %v\n", i)
		fmt.Printf("bridge syncOrcale: %v\nbridge Shared: %v\n", bridge.SyncOracle, bridge.Shared)
		time.Sleep(time.Second)
	}
}
