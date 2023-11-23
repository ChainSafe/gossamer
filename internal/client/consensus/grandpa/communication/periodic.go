package communication

import (
	"time"

	libp2p "github.com/libp2p/go-libp2p/core"
)

type peerIDsNeighbourPacket[N any] struct {
	PeerIDs        []libp2p.PeerID
	NeighborPacket neighborPacket[N]
}

// / A sender used to send neighbor packets to a background job.
type neighbourPacketSender[N any] chan peerIDsNeighbourPacket[N]

// / NeighborPacketWorker is listening on a channel for new neighbor packets being produced by
// / components within `finality-grandpa` and forwards those packets to the underlying
// / `NetworkEngine` through the `NetworkBridge` that it is being polled by (see `Stream`
// / implementation). Periodically it sends out the last packet in cases where no new ones arrive.
//
//	pub(super) struct NeighborPacketWorker<B: BlockT> {
//		last: Option<(Vec<PeerId>, NeighborPacket<NumberFor<B>>)>,
//		rebroadcast_period: Duration,
//		delay: Delay,
//		rx: TracingUnboundedReceiver<(Vec<PeerId>, NeighborPacket<NumberFor<B>>)>,
//	}
type neighborPacketWorker[N any] struct {
	last              *peerIDsNeighbourPacket[N]
	rebroadcastPeriod time.Duration
	delay             time.Timer
	rx                chan peerIDsNeighbourPacket[N]
}
