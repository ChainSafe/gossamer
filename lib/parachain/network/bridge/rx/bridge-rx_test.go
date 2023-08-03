package rx

import (
	"fmt"
	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/parachain"
	"github.com/libp2p/go-libp2p/core"
	"testing"
)

//type NetworkAction scale.VaryingDataType

//// NewNetworkActionVDT constructor for NetworkAction
//func NewNetworkActionVDT() NetworkAction {
//	vdt, err := scale.NewVaryingDataType(ReputationChange{}, DisconnectPeer{}, WriteNotification{})
//	if err != nil {
//		panic(err)
//	}
//	return NetworkAction(vdt)
//}

type ReputationChange struct {
	PeerId           core.PeerID
	ReputationChange peerset.ReputationChange
}

// Index returns varying data type index
func (rc ReputationChange) Index() uint {
	return 0
}

type DisconnectPeer struct {
	PeerId  core.PeerID
	PeerSet peerset.PeerSet
}

// Index returns varying data type index
func (dp DisconnectPeer) Index() uint {
	return 1
}

type WriteNotification struct {
	PeerId  core.PeerID
	PeerSet peerset.PeerSet
	Value   []byte
}

// Index returns varying data type index
func (wn WriteNotification) Index() uint {
	return 2
}

type TestNetworkHandle struct {
	actionRx      NetworkAction
	netTx         NetworkEvent
	protocolNames PeerSetProtocolNames
}

type NetworkEvent struct{}

type PeerSetProtocolNames struct {
	Protocols map[string]struct{}
	Names     map[struct{}]string
}

func TestSendOurViewUponConnection(t *testing.T) {
	view := parachain.View{
		Heads: []common.Hash{{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
			1, 1, 1}},
	}
	fmt.Printf("View %v\n", view)
}
