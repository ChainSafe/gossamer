package bridge

import (
	"github.com/ChainSafe/gossamer/lib/common"
	parachaintypes "github.com/ChainSafe/gossamer/lib/parachain/types"
	"github.com/libp2p/go-libp2p/core"
	"sync"
)

// ProtocolVersion a generic version the protocol
type ProtocolVersion uint32

type PeerData struct {
	view    View
	version ProtocolVersion
}

type SharedInner struct {
	LocalView       View
	ValidationPeers map[core.PeerID]PeerData
	CollationPeers  map[core.PeerID]PeerData
}

type Shared struct {
	sync.Mutex
	SharedInner
}
type View struct {
	Heads           []common.Hash
	FinalizedNumber parachaintypes.BlockNumber
}
