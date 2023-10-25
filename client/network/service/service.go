package service

import (
	"github.com/ChainSafe/gossamer/client/network"
	"github.com/ChainSafe/gossamer/client/network/config"
	"github.com/ChainSafe/gossamer/client/network/event"
	"github.com/ChainSafe/gossamer/client/peerset"
	libp2p "github.com/libp2p/go-libp2p/core"
)

// / Provides an ability to set a fork sync request for a particular block.
type NetworkSyncForkRequest[BlockHash, BlockNumber any] interface {
	/// Notifies the sync service to try and sync the given block from the given
	/// peers.
	///
	/// If the given vector of peers is empty then the underlying implementation
	/// should make a best effort to fetch the block from any peers it is
	/// connected to (NOTE: this assumption will change in the future #3629).
	// fn set_sync_fork_request(&self, peers: Vec<PeerId>, hash: BlockHash, number: BlockNumber);
	SetSyncForkRequest(peers []libp2p.PeerID, hash BlockHash, number BlockNumber)
}

// / Provides low-level API for manipulating network peers.
type NetworkPeers interface {
	/// Set authorized peers.
	///
	/// Need a better solution to manage authorized peers, but now just use reserved peers for
	/// prototyping.
	// 	fn set_authorized_peers(&self, peers: HashSet<PeerId>);
	SetAuthorizedPeers(peers map[libp2p.PeerID]any)
	/// Set authorized_only flag.
	///
	/// Need a better solution to decide authorized_only, but now just use reserved_only flag for
	/// prototyping.
	// 	fn set_authorized_only(&self, reserved_only: bool);
	SetAuthorizedOnly(reservedOnly bool)
	//  Adds an address known to a node.
	// 	fn add_known_address(&self, peer_id: PeerId, addr: Multiaddr);
	AddKnownAddress(peerID libp2p.PeerID, addr libp2p.Multiaddr)
	//  Report a given peer as either beneficial (+) or costly (-) according to the
	//  given scalar.
	// 	fn report_peer(&self, who: PeerId, cost_benefit: ReputationChange);
	ReportPeer(peerID libp2p.PeerID, costBenefit peerset.ReputationChange)
	//  Disconnect from a node as soon as possible.
	//
	//  This triggers the same effects as if the connection had closed itself spontaneously.
	//
	//  See also [`NetworkPeers::remove_from_peers_set`], which has the same effect but also
	//  prevents the local node from re-establishing an outgoing substream to this peer until it
	//  is added again.
	// 	fn disconnect_peer(&self, who: PeerId, protocol: ProtocolName);
	DisconnectPeer(who libp2p.PeerID, protocol network.ProtocolName)
	// /// Connect to unreserved peers and allow unreserved peers to connect for syncing purposes.
	// fn accept_unreserved_peers(&self);
	AcceptUnreservedPeers()
	//  Disconnect from unreserved peers and deny new unreserved peers to connect for syncing
	//  purposes.
	// 	fn deny_unreserved_peers(&self);
	DenyUnreservedPeers()
	//  Adds a `PeerId` and its `Multiaddr` as reserved for a sync protocol (default peer set).
	//
	//  Returns an `Err` if the given string is not a valid multiaddress
	//  or contains an invalid peer ID (which includes the local peer ID).
	// 	fn add_reserved_peer(&self, peer: MultiaddrWithPeerId) -> Result<(), String>;
	AddReservedPeer(peer config.MultiaddrPeerId) error
	//  Removes a `PeerId` from the list of reserved peers for a sync protocol (default peer set).
	// 	fn remove_reserved_peer(&self, peer_id: PeerId);
	RemoveReservedPeer(peerID libp2p.PeerID)
	//  Sets the reserved set of a protocol to the given set of peers.
	//
	//  Each `Multiaddr` must end with a `/p2p/` component containing the `PeerId`. It can also
	//  consist of only `/p2p/<peerid>`.
	//
	//  The node will start establishing/accepting connections and substreams to/from peers in this
	//  set, if it doesn't have any substream open with them yet.
	//
	//  Note however, if a call to this function results in less peers on the reserved set, they
	//  will not necessarily get disconnected (depending on available free slots in the peer set).
	//  If you want to also disconnect those removed peers, you will have to call
	//  `remove_from_peers_set` on those in addition to updating the reserved set. You can omit
	//  this step if the peer set is in reserved only mode.
	//
	//  Returns an `Err` if one of the given addresses is invalid or contains an
	//  invalid peer ID (which includes the local peer ID).
	// 	fn set_reserved_peers(
	// 		&self,
	// 		protocol: ProtocolName,
	// 		peers: HashSet<Multiaddr>,
	// 	) -> Result<(), String>;
	SetReservedPeers(protocol network.ProtocolName, peers map[libp2p.Multiaddr]any) error
	//  Add peers to a peer set.
	//
	//  Each `Multiaddr` must end with a `/p2p/` component containing the `PeerId`. It can also
	//  consist of only `/p2p/<peerid>`.
	//
	//  Returns an `Err` if one of the given addresses is invalid or contains an
	//  invalid peer ID (which includes the local peer ID).
	// 	fn add_peers_to_reserved_set(
	// 		&self,
	// 		protocol: ProtocolName,
	// 		peers: HashSet<Multiaddr>,
	// 	) -> Result<(), String>;
	AddPeersToReservedSet(protocol network.ProtocolName, peers map[libp2p.Multiaddr]any) error
	//  Remove peers from a peer set.
	// 	fn remove_peers_from_reserved_set(&self, protocol: ProtocolName, peers: Vec<PeerId>);
	RemovePeersFromReservedSet(protocol network.ProtocolName, peers []libp2p.PeerID)
	//  Add a peer to a set of peers.
	//
	//  If the set has slots available, it will try to open a substream with this peer.
	//
	//  Each `Multiaddr` must end with a `/p2p/` component containing the `PeerId`. It can also
	//  consist of only `/p2p/<peerid>`.
	//
	//  Returns an `Err` if one of the given addresses is invalid or contains an
	//  invalid peer ID (which includes the local peer ID).
	// 	fn add_to_peers_set(
	// 		&self,
	// 		protocol: ProtocolName,
	// 		peers: HashSet<Multiaddr>,
	// 	) -> Result<(), String>;
	AddToPeersSet(protocol network.ProtocolName, peers map[libp2p.Multiaddr]any) error
	//  Remove peers from a peer set.
	//
	//  If we currently have an open substream with this peer, it will soon be closed.
	// 	fn remove_from_peers_set(&self, protocol: ProtocolName, peers: Vec<PeerId>);
	RemoveFromPeersSet(protocol network.ProtocolName, peers []libp2p.PeerID)
	//  Returns the number of peers in the sync peer set we're connected to.
	// 	fn sync_num_connected(&self) -> usize;
	SyncNumConnected() uint
}

// / Provides access to network-level event stream.
type NetworkEventStream interface {
	/// Returns a stream containing the events that happen on the network.
	///
	/// If this method is called multiple times, the events are duplicated.
	///
	/// The stream never ends (unless the `NetworkWorker` gets shut down).
	///
	/// The name passed is used to identify the channel in the Prometheus metrics. Note that the
	/// parameter is a `&'static str`, and not a `String`, in order to avoid accidentally having
	/// an unbounded set of Prometheus metrics, which would be quite bad in terms of memory
	// fn event_stream(&self, name: &'static str) -> Pin<Box<dyn Stream<Item = Event> + Send>>;
	EventStream(name string) chan event.Event
}

// / Provides ability to send network notifications.
type NetworkNotification interface {
	/// Appends a notification to the buffer of pending outgoing notifications with the given peer.
	/// Has no effect if the notifications channel with this protocol name is not open.
	///
	/// If the buffer of pending outgoing notifications with that peer is full, the notification
	/// is silently dropped and the connection to the remote will start being shut down. This
	/// happens if you call this method at a higher rate than the rate at which the peer processes
	/// these notifications, or if the available network bandwidth is too low.
	///
	/// For this reason, this method is considered soft-deprecated. You are encouraged to use
	/// [`NetworkNotification::notification_sender`] instead.
	///
	/// > **Note**: The reason why this is a no-op in the situation where we have no channel is
	/// >			that we don't guarantee message delivery anyway. Networking issues can cause
	/// >			connections to drop at any time, and higher-level logic shouldn't differentiate
	/// >			between the remote voluntarily closing a substream or a network error
	/// >			preventing the message from being delivered.
	///
	/// The protocol must have been registered with
	/// `crate::config::NetworkConfiguration::notifications_protocols`.
	// fn write_notification(&self, target: PeerId, protocol: ProtocolName, message: Vec<u8>);

	/// Obtains a [`NotificationSender`] for a connected peer, if it exists.
	///
	/// A `NotificationSender` is scoped to a particular connection to the peer that holds
	/// a receiver. With a `NotificationSender` at hand, sending a notification is done in two
	/// steps:
	///
	/// 1.  [`NotificationSender::ready`] is used to wait for the sender to become ready
	/// for another notification, yielding a [`NotificationSenderReady`] token.
	/// 2.  [`NotificationSenderReady::send`] enqueues the notification for sending. This operation
	/// can only fail if the underlying notification substream or connection has suddenly closed.
	///
	/// An error is returned by [`NotificationSenderReady::send`] if there exists no open
	/// notifications substream with that combination of peer and protocol, or if the remote
	/// has asked to close the notifications substream. If that happens, it is guaranteed that an
	/// [`Event::NotificationStreamClosed`] has been generated on the stream returned by
	/// [`NetworkEventStream::event_stream`].
	///
	/// If the remote requests to close the notifications substream, all notifications successfully
	/// enqueued using [`NotificationSenderReady::send`] will finish being sent out before the
	/// substream actually gets closed, but attempting to enqueue more notifications will now
	/// return an error. It is however possible for the entire connection to be abruptly closed,
	/// in which case enqueued notifications will be lost.
	///
	/// The protocol must have been registered with
	/// `crate::config::NetworkConfiguration::notifications_protocols`.
	///
	/// # Usage
	///
	/// This method returns a struct that allows waiting until there is space available in the
	/// buffer of messages towards the given peer. If the peer processes notifications at a slower
	/// rate than we send them, this buffer will quickly fill up.
	///
	/// As such, you should never do something like this:
	///
	/// ```ignore
	/// // Do NOT do this
	/// for peer in peers {
	/// 	if let Ok(n) = network.notification_sender(peer, ...) {
	/// 			if let Ok(s) = n.ready().await {
	/// 				let _ = s.send(...);
	/// 			}
	/// 	}
	/// }
	/// ```
	///
	/// Doing so would slow down all peers to the rate of the slowest one. A malicious or
	/// malfunctioning peer could intentionally process notifications at a very slow rate.
	///
	/// Instead, you are encouraged to maintain your own buffer of notifications on top of the one
	/// maintained by `sc-network`, and use `notification_sender` to progressively send out
	/// elements from your buffer. If this additional buffer is full (which will happen at some
	/// point if the peer is too slow to process notifications), appropriate measures can be taken,
	/// such as removing non-critical notifications from the buffer or disconnecting the peer
	/// using [`NetworkPeers::disconnect_peer`].
	///
	///
	/// Notifications              Per-peer buffer
	///   broadcast    +------->   of notifications   +-->  `notification_sender`  +-->  Internet
	///                    ^       (not covered by
	///                    |         sc-network)
	///                    +
	///      Notifications should be dropped
	///             if buffer is full
	///
	///
	/// See also the `sc-network-gossip` crate for a higher-level way to send notifications.
	// fn notification_sender(
	// 	&self,
	// 	target: PeerId,
	// 	protocol: ProtocolName,
	// ) -> Result<Box<dyn NotificationSender>, NotificationSenderError>;

	/// Set handshake for the notification protocol.
	// fn set_notification_handshake(&self, protocol: ProtocolName, handshake: Vec<u8>);
}

// / Provides ability to announce blocks to the network.
type NetworkBlock[BlockHash any, BlockNumber any] interface {
	/// Make sure an important block is propagated to peers.
	///
	/// In chain-based consensus, we often need to make sure non-best forks are
	/// at least temporarily synced. This function forces such an announcement.
	// fn announce_block(&self, hash: BlockHash, data: Option<Vec<u8>>);
	AnnounceBlock(hash BlockHash, data []byte)

	/// Inform the network service about new best imported block.
	// fn new_best_block_imported(&self, hash: BlockHash, number: BlockNumber);
	NewBestBlockImported(hash BlockHash, number BlockNumber)
}
