package grandpa

import (
	"time"

	peerid "github.com/ChainSafe/gossamer/internal/client/network/types/peer-id"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
)

type peerIDsNeighborPacket[N runtime.Number] struct {
	PeerIDs        []peerid.PeerID
	NeighborPacket neighborPacket[N]
}

// / A sender used to send neighbor packets to a background job.
type neighbourPacketSender[N runtime.Number] chan peerIDsNeighborPacket[N]

// neighborPacketWorker is listening on a channel for new neighbor packets being produced by components within
// `finality-grandpa` and forwards those packets to the underlying `NetworkEngine` through the `NetworkBridge` that it
// is being polled. Periodically it sends out the last packet in cases where no new ones arrive.
type neighborPacketWorker[N runtime.Number] struct {
	last              *peerIDsNeighborPacket[N]
	rebroadcastPeriod time.Duration
	delay             *time.Timer // not meant to be optional
	rx                chan peerIDsNeighborPacket[N]
}

func newNeighborPacketWorker[N runtime.Number](rebroadcastPeriod time.Duration) (neighborPacketWorker[N], neighbourPacketSender[N]) {
	sender := make(neighbourPacketSender[N], 100_000) // supposed to be unbounded, and queue size warning of 100_000
	delay := time.NewTimer(rebroadcastPeriod)
	return neighborPacketWorker[N]{
		last:              nil,
		rebroadcastPeriod: rebroadcastPeriod,
		delay:             delay,
		rx:                sender,
	}, sender
}

type peerIDsGossipMessage struct {
	PeerIDs       []peerid.PeerID
	GossipMessage gossipMessage
}

func (w *neighborPacketWorker[N]) Stream() chan peerIDsGossipMessage {
	ch := make(chan peerIDsGossipMessage)
	go func() {
		defer close(ch)
		for {
			select {
			case msg, ok := <-w.rx:
				if !ok {
					return
				}
				to := msg.PeerIDs
				packet := msg.NeighborPacket
				w.delay.Reset(w.rebroadcastPeriod)
				w.last = &peerIDsNeighborPacket[N]{
					PeerIDs:        to,
					NeighborPacket: packet,
				}
				ch <- peerIDsGossipMessage{
					PeerIDs:       to,
					GossipMessage: gossipMessageNeighbor[N](versionedNeighborPacket[N]{packet}),
				}
			case <-w.delay.C:
				// Getting this far here implies that the timer fired.
				w.delay.Reset(w.rebroadcastPeriod)

				if w.last != nil {
					ch <- peerIDsGossipMessage{
						PeerIDs:       w.last.PeerIDs,
						GossipMessage: gossipMessageNeighbor[N](versionedNeighborPacket[N]{w.last.NeighborPacket}),
					}
				}
			}
		}
	}()
	return ch
}
