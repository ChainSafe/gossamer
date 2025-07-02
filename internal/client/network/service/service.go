// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package service

import (
	"github.com/ChainSafe/gossamer/internal/client/network"
	"github.com/ChainSafe/gossamer/internal/client/network/config"
	"github.com/ChainSafe/gossamer/internal/client/network/event"
	"github.com/ChainSafe/gossamer/internal/client/network/role"
	"github.com/ChainSafe/gossamer/internal/client/network/types/multiaddr"
	peerid "github.com/ChainSafe/gossamer/internal/client/network/types/peer-id"
)

// NetworkSyncForkRequest provides an ability to set a fork sync request for a particular block.
type NetworkSyncForkRequest[BlockHash, BlockNumber any] interface {
	// Notifies the sync service to try and sync the given block from the given peers.
	//
	// If the given slice of peers is empty then the underlying implementation
	// should make a best effort to fetch the block from any peers it is
	// connected to.
	SetSyncForkRequest(peers []peerid.PeerID, hash BlockHash, number BlockNumber)
}

// NetworkPeers provides low-level API for manipulating network peers.
type NetworkPeers interface {
	// Set authorized peers.
	//
	// Need a better solution to manage authorized peers, but now just use reserved peers for
	// prototyping.
	SetAuthorizedPeers(peers map[peerid.PeerID]struct{})
	// Set authorized_only flag.
	//
	// Need a better solution to decide authorized_only, but now just use reservedOnly flag for prototyping.
	SetAuthorizedOnly(reservedOnly bool)
	// Adds an address known to a node.
	AddKnownAddress(peerID peerid.PeerID, addr multiaddr.Multiaddr)
	// Report a given peer as either beneficial (+) or costly (-) according to the given scalar.
	ReportPeer(peerID peerid.PeerID, costBenefit network.ReputationChange)
	// Get peer reputation.
	PeerReputation(peerID peerid.PeerID) int32
	// Disconnect from a node as soon as possible.
	//
	// This triggers the same effects as if the connection had closed itself spontaneously.
	DisconnectPeer(who peerid.PeerID, protocol network.ProtocolName)
	// Connect to unreserved peers and allow unreserved peers to connect for syncing purposes.
	AcceptUnreservedPeers()
	// Disconnect from unreserved peers and deny new unreserved peers to connect for syncing purposes.
	DenyUnreservedPeers()
	// Adds a PeerID and its MultiAddr as reserved for a sync protocol (default peer set).
	//
	// Returns an error if the given string is not a valid multiaddress or contains an invalid peer ID (which includes
	// the local peer ID).
	AddReservedPeer(peer config.MultiaddrPeerId) error
	// Removes a [peerid.PeerID] from the list of reserved peers for a sync protocol (default peer set).
	RemoveReservedPeer(peerID peerid.PeerID)
	// Sets the reserved set of a protocol to the given set of peers.
	//
	// Each Multiaddr must end with a "/p2p/" component containing the peer id. It can also
	// consist of only "/p2p/<peerid>".
	//
	// The node will start establishing/accepting connections and substreams to/from peers in this set, if it doesn't
	// have any substream open with them yet.
	//
	// Note however, if a call to this function results in less peers on the reserved set, they will not necessarily
	// get disconnected (depending on available free slots in the peer set). If you want to also disconnect those
	// removed peers, you will have to call RemoveFromPeersSet on those in addition to updating the reserved set. You
	// can omit this step if the peer set is in reserved only mode.
	//
	// Returns an error if one of the given addresses is invalid or contains an invalid peer ID (which includes the
	// local peer id).
	SetReservedPeers(protocol network.ProtocolName, peers map[multiaddr.Multiaddr]struct{}) error
	// Add peers to a peer set.
	//
	// Each Multiaddr must end with a "/p2p/" component containing the peer id. It can also consist of only
	// "/p2p/<peerid>".
	//
	// Returns an error if one of the given addresses is invalid or contains an invalid peer id (which includes the
	// local peer id).
	AddPeersToReservedSet(protocol network.ProtocolName, peers map[multiaddr.Multiaddr]struct{}) error
	// Remove peers from a peer set.
	RemovePeersFromReservedSet(protocol network.ProtocolName, peers []peerid.PeerID)
	// Returns the number of peers in the sync peer set we're connected to.
	SyncNumConnected() uint

	// Attempt to get peer role.
	//
	// Right now the peer role is decoded from the received handshake for all protocols ("/block-announces/1" has other
	// information as well). If the handshake cannot be decoded into a role, the role queried from peer storeand if the
	// role is not stored there either, nil is returned and the peer should be discarded.
	PeerRole(peerID peerid.PeerID, handshake []byte) *role.ObservedRole

	// Get the list of reserved peers.
	//
	// Returns an error if the network worker is no longer running.
	ReservedPeers() <-chan struct {
		Peers []peerid.PeerID
		Error error
	}
}

// NetworkEventStream provides access to network-level event stream.
type NetworkEventStream interface {
	// Returns a stream containing the events that happen on the network. If this method is called multiple times, the
	// events are duplicated. The stream never ends (unless the network worker gets shut down). The name passed is
	// used to identify the channel in the Prometheus metrics.
	EventStream(name string) chan event.Event
}

// NetworkBlock provides ability to announce blocks to the network.
type NetworkBlock[BlockHash any, BlockNumber any] interface {
	// Make sure an important block is propagated to peers.
	//
	// In chain based consensus, we often need to make sure non-best forks are at least temporarily synced. This
	// function forces such an announcement.
	AnnounceBlock(hash BlockHash, data []byte)

	// Inform the network service about new best imported block.
	NewBestBlockImported(hash BlockHash, number BlockNumber)
}

// ValidationResult is the substream acceptance result.
type ValidationResult uint

const (
	// Accept inbound substream.
	ValidationResultAccept = iota + 1
	// Reject inbound substream.
	ValidationResultReject
)

// Direction is the substream direction.
type Direction uint

const (
	// Substream opened by the remote node.
	DirectionInbound = iota + 1
	// Substream opened by the local node.
	DirectionOutbound
)

// NotificationEvent are events received by the protocol from [NotificationService].
type NotificationEvent interface {
	isNotificationEvent()
}

// NotificationEventValidateInboundSubstream is the validate inbound substream event.
type NotificationEventValidateInboundSubstream struct {
	// Peer ID.
	Peer peerid.PeerID
	// Received handshake.
	Handshake []byte
	// Channel for sending validation result back to [NotificationService]
	ResultChan chan ValidationResult
}

func (NotificationEventValidateInboundSubstream) isNotificationEvent() {}

// NotificationEventNotificationStreamOpened means a remote identified by peer id opened a substream and sent
// handshake. Validate handshake and report status (accept/reject) to [NotificationService].
type NotificationEventNotificationStreamOpened struct {
	// Peer ID.
	Peer peerid.PeerID
	// Is the substream inbound or outbound.
	Direction Direction
	// Received handshake.
	Handshake []byte
	// Negotiated fallback.
	NegotiatedFallback *network.ProtocolName
}

func (NotificationEventNotificationStreamOpened) isNotificationEvent() {}

// NotificationEventNotificationStreamClosed means a substream was closed.
type NotificationEventNotificationStreamClosed struct {
	// Peer ID.
	Peer peerid.PeerID
}

func (NotificationEventNotificationStreamClosed) isNotificationEvent() {}

// NotificationEventNotificationReceived means a notification was received from the substream.
type NotificationEventNotificationReceived struct {
	// Peer ID.
	Peer peerid.PeerID
	// Received notification.
	Notification []byte
}

func (NotificationEventNotificationReceived) isNotificationEvent() {}

// NotificationService is the notification service. It defines behaviours that both the protocol implementations and
// NotificationService can expect from each other.
//
// NotificationService can send two different kinds of information to protocol:
//   - substream-related information
//   - notification-related information
//
// When an unvalidated, inbound substream is received by NotificationService, it sends the inbound stream information
// (peer ID, handshake) to protocol for validation. Protocol must then verify that the handshake is valid (and in the
// future that it has a slot it can allocate for the peer) and then report back the [ValidationResult] which is either
// [ValidationResultAccept] or [ValidationResultReject].
//
// After the validation result has been received by NotificationService, it prepares the substream for communication by
// initialising the necessary sinks and emits [NotificationEventNotificationStreamOpened] which informs the protocol
// that the remote peer is ready to receive notifications.
//
// Both local and remote peer can close the substream at any time. Local peer can do so by calling CloseSubstream which
// instructs NotificationService to close the substream. Remote closing the substream is indicated to the local peer by
// receiving [NotificationEventNotificationStreamClosed] event.
//
// In case the protocol must update its handshake while it's operating (such as updating the best block information),
// it can do so by calling SetHandshake which instructs NotificationService to update the handshake it stored during
// protocol initialization.
//
// All peer events are multiplexed on the same incoming event stream from NotificationService and thus each event
// carries a peer id so the protocol knows whose information to update when receiving an event.
type NotificationService interface {
	// Instruct NotificationService to open a new substream for peer.
	OpenSubstream(peer peerid.PeerID) <-chan error

	// Instruct NotificationService to close substream for "peer".
	CloseSubstream(peer peerid.PeerID) <-chan error

	// Send synchronous notification to peer.
	SendSyncNotification(peer peerid.PeerID, notification []byte)

	// Send asynchronous notification to peer, allowing sender to exercise backpressure.
	// Returns an error if the peer doesn't exist.
	SendAsyncNotification(peer peerid.PeerID, notification []byte) <-chan error

	// Set handshake for the notification protocol replacing the old handshake.
	SetHandshake(handshake []byte) <-chan error

	// Non-blocking variant of SetHandshake that attempts to update the handshake and returns an error if the channel
	// is blocked.
	//
	// Technically the function can return an error if the channel to NotificationService is closed but that doesn't
	// happen under normal operation.
	TrySetHandshake(handshake []byte) error

	// Get next event from the NotificationService event stream.
	NextEvent() <-chan NotificationEvent

	// Get protocol name of the NotificationService.
	Protocol() network.ProtocolName

	// Get message sink of the peer.
	MessageSink(peer peerid.PeerID) MessageSink
}

// MessageSink for peers.
//
// If protocol cannot use [NotificationService] to send notifications to peers and requires, e.g., notifications to be
// sent in another task, the protocol may acquire a MessageSink object for each peer by calling
// [NotificationService.MessageSink]. Calling this function returns an object which allows the protocol to send
// notifications to the remote peer.
//
// Use of this API is discouraged as it's not as performant as sending notifications through [NotificationService] due
// to synchronisation required to keep the underlying notification sink up to date with possible sink replacement
// events.
type MessageSink interface {
	// Send synchronous notification to the peer associated with this MessageSink.
	SendSyncNotification(notification []byte)

	// Send an asynchronous notification to to the peer associated with this MessageSink,
	// allowing sender to exercise backpressure.
	//
	// Returns an error if the peer does not exist.
	SendAsyncNotification(notification []byte) <-chan error
}
