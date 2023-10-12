package gossip

import (
	"time"

	"github.com/ChainSafe/gossamer/client/network"
)

/// Wraps around an implementation of the [`Network`] trait and provides gossiping capabilities on
/// top of it.
// pub struct GossipEngine<B: BlockT> {
// 	state_machine: ConsensusGossip<B>,
// 	network: Box<dyn Network<B> + Send>,
// 	sync: Box<dyn Syncing<B>>,
// 	periodic_maintenance_interval: futures_timer::Delay,
// 	protocol: ProtocolName,

// 	/// Incoming events from the network.
// 	network_event_stream: Pin<Box<dyn Stream<Item = Event> + Send>>,
// 	/// Incoming events from the syncing service.
// 	sync_event_stream: Pin<Box<dyn Stream<Item = SyncEvent> + Send>>,
// 	/// Outgoing events to the consumer.
// 	message_sinks: HashMap<B::Hash, Vec<Sender<TopicNotification>>>,
// 	/// Buffered messages (see [`ForwardingState`]).
// 	forwarding_state: ForwardingState<B>,

//		is_terminated: bool,
//	}
type GossipEngine struct {
	stateMachine                ConsensusGossip
	network                     Network
	sync                        Syncing
	periodicMaintenanceInterval time.Timer
	protocol                    network.ProtocolName
}
