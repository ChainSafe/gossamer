package parachaininteraction

import (
	"fmt"

	"github.com/ChainSafe/gossamer/lib/common"
)

const (
	LEGACY_VALIDATION_PROTOCOL_V1 = "/polkadot/validation/1"
	LEGACY_COLLATION_PROTOCOL_V1  = "/polkadot/collation/1"

	// TODO: Find the actual current protocol names and fill these constants
	// could be something like 		"/7ac8741de8b7146d8a5617fd462914557fe63c265a7f1c10e7dae32858eebb80/collation/1";
	COLLATION_PROTOCOL_NAME  = ""
	VALIDATION_PROTOCOL_NAME = ""
	// all protocol related things here /home/kishan/code/polkadot/node/network/protocol/src/request_response/mod.rs

// impl ReqProtocolNames {
// 	/// Construct [`ReqProtocolNames`] from `genesis_hash` and `fork_id`.
// 	pub fn new<Hash: AsRef<[u8]>>(genesis_hash: Hash, fork_id: Option<&str>) -> Self {
// 		let mut names = HashMap::new();
// 		for protocol in Protocol::iter() {
// 			names.insert(protocol, Self::generate_name(protocol, &genesis_hash, fork_id));
// 		}
// 		Self { names }
// 	}

// 	/// Get on the wire [`Protocol`] name.
// 	pub fn get_name(&self, protocol: Protocol) -> ProtocolName {
// 		self.names
// 			.get(&protocol)
// 			.expect("All `Protocol` enum variants are added above via `strum`; qed")
// 			.clone()
// 	}

// 	/// Protocol name of this protocol based on `genesis_hash` and `fork_id`.
// 	fn generate_name<Hash: AsRef<[u8]>>(
// 		protocol: Protocol,
// 		genesis_hash: &Hash,
// 		fork_id: Option<&str>,
// 	) -> ProtocolName {
// 		let prefix = if let Some(fork_id) = fork_id {
// 			format!("/{}/{}", hex::encode(genesis_hash), fork_id)
// 		} else {
// 			format!("/{}", hex::encode(genesis_hash))
// 		};

// 		let short_name = match protocol {
// 			Protocol::ChunkFetchingV1 => "/req_chunk/1",
// 			Protocol::CollationFetchingV1 => "/req_collation/1",
// 			Protocol::PoVFetchingV1 => "/req_pov/1",
// 			Protocol::AvailableDataFetchingV1 => "/req_available_data/1",
// 			Protocol::StatementFetchingV1 => "/req_statement/1",
// 			Protocol::DisputeSendingV1 => "/send_dispute/1",
// 		};

// 		format!("{}{}", prefix, short_name).into()
// 	}
// }

)

type ReqProtocolName uint

const (
	ChunkFetchingV1 ReqProtocolName = iota
	CollationFetchingV1
	PoVFetchingV1
	AvailableDataFetchingV1
	StatementFetchingV1
	DisputeSendingV1
)

type PeerSetProtocolName uint

const (
	ValidationProtocol PeerSetProtocolName = iota
	CollationProtocol
)

// NOTE: All protocol related things here /polkadot/node/network/protocol/src/request_response/mod.rs

func GenerateReqProtocolName(protocol ReqProtocolName, forkID string, GenesisHash common.Hash) string {
	prefix := fmt.Sprintf("/%s", GenesisHash.String())

	if forkID != "" {
		prefix = fmt.Sprintf("%s/%s", prefix, forkID)
	}

	switch protocol {
	case ChunkFetchingV1:
		return fmt.Sprintf("%s/req_chunk/1", prefix)
	case CollationFetchingV1:
		return fmt.Sprintf("%s/req_collation/1", prefix)
	case PoVFetchingV1:
		return fmt.Sprintf("%s/req_pov/1", prefix)
	case AvailableDataFetchingV1:
		return fmt.Sprintf("%s/req_available_data/1", prefix)
	case StatementFetchingV1:
		return fmt.Sprintf("%s/req_statement/1", prefix)
	case DisputeSendingV1:
		return fmt.Sprintf("%s/send_dispute/1", prefix)
	default:
		panic("unknown protocol")
	}
}

func GeneratePeersetProtocolName(protocol PeerSetProtocolName, forkID string, GenesisHash common.Hash, version uint32) string {
	prefix := fmt.Sprintf("/%s", GenesisHash.String())

	if forkID != "" {
		prefix = fmt.Sprintf("%s/%s", prefix, forkID)
	}

	switch protocol {
	case ValidationProtocol:
		return fmt.Sprintf("%s/validation/%d", prefix, version)
		// message over this protocol is BitfieldDistributionMessage, StatementDistributionMessage, ApprovalDistributionMessage
	case CollationProtocol:
		return fmt.Sprintf("%s/collation/%d", prefix, version)

		// message over this protocol is CollatorProtocolMessage
	default:
		panic("unknown protocol")
	}
}

/*

A node can let us know that it is a collator using `Declare` message
```
	#[derive(Debug, Clone, Encode, Decode, PartialEq, Eq)]
	pub enum CollatorProtocolMessage {
		/// Declare the intent to advertise collations under a collator ID, attaching a
		/// signature of the `PeerId` of the node using the given collator ID key.
		#[codec(index = 0)]
		Declare(CollatorId, ParaId, CollatorSignature),
		/// Advertise a collation to a validator. Can only be sent once the peer has
		/// declared that they are a collator with given ID.
		#[codec(index = 1)]
		AdvertiseCollation(Hash),
		/// A collation sent to a validator was seconded.
		#[codec(index = 4)]
		CollationSeconded(Hash, UncheckedSignedFullStatement),
	}
```

Register two protocols:
- ValidationProtocolV1
- CollationProtocolV1


- Use service.SendMessage to send messages
*/

// /home/kishan/code/polkadot/node/network/bridge/src/tx/mod.rs

// send message is in node/network/bridge/src/network.rs

// code for send_message
// pub(crate) fn send_message<M>(
// 	net: &mut impl Network,
// 	mut peers: Vec<PeerId>,
// 	peer_set: PeerSet,
// 	version: ProtocolVersion,
// 	protocol_names: &PeerSetProtocolNames,
// 	message: M,
// 	metrics: &super::Metrics,
// ) where
// 	M: Encode + Clone,
// {

// Implement network pub trait Network: Clone + Send + 'static {

// code to write notifications is in substrate, its following in substrate /client/network/src/service.rs

// impl<B, H> NetworkEventStream for NetworkService<B, H>
// where
// 	B: BlockT + 'static,
// 	H: ExHashT,
// {
// 	fn event_stream(&self, name: &'static str) -> Pin<Box<dyn Stream<Item = Event> + Send>> {
// 		let (tx, rx) = out_events::channel(name, 100_000);
// 		let _ = self.to_worker.unbounded_send(ServiceToWorkerMsg::EventStream(tx));
// 		Box::pin(rx)
// 	}
// }

// impl<B, H> NetworkNotification for NetworkService<B, H>
// where
// 	B: BlockT + 'static,
// 	H: ExHashT,
// {
// 	fn write_notification(&self, target: PeerId, protocol: ProtocolName, message: Vec<u8>) {
// 		// We clone the `NotificationsSink` in order to be able to unlock the network-wide
// 		// `peers_notifications_sinks` mutex as soon as possible.
// 		let sink = {
// 			let peers_notifications_sinks = self.peers_notifications_sinks.lock();
// 			if let Some(sink) = peers_notifications_sinks.get(&(target, protocol.clone())) {
// 				sink.clone()
// 			} else {
// 				// Notification silently discarded, as documented.
// 				debug!(
// 					target: "sub-libp2p",
// 					"Attempted to send notification on missing or closed substream: {}, {:?}",
// 					target, protocol,
// 				);
// 				return
// 			}
// 		};

// 		if let Some(notifications_sizes_metric) = self.notifications_sizes_metric.as_ref() {
// 			notifications_sizes_metric
// 				.with_label_values(&["out", &protocol])
// 				.observe(message.len() as f64);
// 		}

// 		// Sending is communicated to the `NotificationsSink`.
// 		trace!(
// 			target: "sub-libp2p",
// 			"External API => Notification({:?}, {:?}, {} bytes)",
// 			target, protocol, message.len()
// 		);
// 		trace!(target: "sub-libp2p", "Handler({:?}) <= Sync notification", target);
// 		sink.send_sync_notification(message);
// 	}

// 	fn notification_sender(
// 		&self,
// 		target: PeerId,
// 		protocol: ProtocolName,
// 	) -> Result<Box<dyn NotificationSenderT>, NotificationSenderError> {
// 		// We clone the `NotificationsSink` in order to be able to unlock the network-wide
// 		// `peers_notifications_sinks` mutex as soon as possible.
// 		let sink = {
// 			let peers_notifications_sinks = self.peers_notifications_sinks.lock();
// 			if let Some(sink) = peers_notifications_sinks.get(&(target, protocol.clone())) {
// 				sink.clone()
// 			} else {
// 				return Err(NotificationSenderError::Closed)
// 			}
// 		};

// 		let notification_size_metric = self
// 			.notifications_sizes_metric
// 			.as_ref()
// 			.map(|histogram| histogram.with_label_values(&["out", &protocol]));

// 		Ok(Box::new(NotificationSender { sink, protocol_name: protocol, notification_size_metric }))
// 	}
// }

// #[async_trait::async_trait]
// impl<B, H> NetworkRequest for NetworkService<B, H>
// where
// 	B: BlockT + 'static,
// 	H: ExHashT,
// {
// 	async fn request(
// 		&self,
// 		target: PeerId,
// 		protocol: ProtocolName,
// 		request: Vec<u8>,
// 		connect: IfDisconnected,
// 	) -> Result<Vec<u8>, RequestFailure> {
// 		let (tx, rx) = oneshot::channel();

// 		self.start_request(target, protocol, request, tx, connect);

// 		match rx.await {
// 			Ok(v) => v,
// 			// The channel can only be closed if the network worker no longer exists. If the
// 			// network worker no longer exists, then all connections to `target` are necessarily
// 			// closed, and we legitimately report this situation as a "ConnectionClosed".
// 			Err(_) => Err(RequestFailure::Network(OutboundFailure::ConnectionClosed)),
// 		}
// 	}

// 	fn start_request(
// 		&self,
// 		target: PeerId,
// 		protocol: ProtocolName,
// 		request: Vec<u8>,
// 		tx: oneshot::Sender<Result<Vec<u8>, RequestFailure>>,
// 		connect: IfDisconnected,
// 	) {
// 		let _ = self.to_worker.unbounded_send(ServiceToWorkerMsg::Request {
// 			target,
// 			protocol: protocol.into(),
// 			request,
// 			pending_response: tx,
// 			connect,
// 		});
// 	}
// }

// How to register a protocol
// look at get_info in /home/kishan/code/polkadot/node/network/protocol/src/peer_set.rs
