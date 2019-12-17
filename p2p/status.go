// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more detailg.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package p2p

import (
	"context"
	"time"

	log "github.com/ChainSafe/log15"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
)

// ExpireStatusInterval is the time between checking if status expired
const ExpireStatusInterval = 20 * time.Minute

// SendStatusInterval is the time between sending status messages
const SendStatusInterval = 5 * time.Minute

// status submodule
type status struct {
	host          *host
	hostMessage   *StatusMessage
	peerConfirmed map[peer.ID]time.Time
	peerMessage   map[peer.ID]*StatusMessage
}

// newStatus creates a new status instance from host
func newStatus(host *host) (s *status, err error) {
	s = &status{
		host:          host,
		peerConfirmed: make(map[peer.ID]time.Time),
		peerMessage:   make(map[peer.ID]*StatusMessage),
	}
	return s, err
}

// confirmed returns true if peer is confirmed
func (status *status) confirmed(peer peer.ID) bool {
	return !status.peerConfirmed[peer].IsZero()
}

// setHostMessage sets the host status message
func (status *status) setHostMessage(msg Message) {
	status.hostMessage = msg.(*StatusMessage)
}

// handleConn starts status processes upon connection
func (status *status) handleConn(conn network.Conn) {
	ctx := context.Background()
	peer := conn.RemotePeer()

	// check if host message set
	if status.hostMessage != nil {

		// start sending status messages to connected peer
		go status.sendMessages(ctx, peer)

	} else {
		log.Error(
			"Failed to start sending status messages to peer",
			"peer", peer,
			"err", "host status message not set",
		)
	}
}

// sendMessages sends status messages to the connected peer
func (status *status) sendMessages(ctx context.Context, peer peer.ID) {
	for {
		// check if peer is still connected
		if !status.host.peerConnected(peer) {
			ctx.Done() // cancel running processes
			return     // exit
		}

		// send host status message to peer
		err := status.host.send(peer, status.hostMessage)
		if err != nil {
			log.Error(
				"Failed to send status message to peer",
				"peer", peer,
				"err", err,
			)
		}

		// wait between sending status messages
		time.Sleep(SendStatusInterval)
	}
}

// handleMessage checks if the peer status message is compatibale with the host
// status message, then either manages peer status or closes peer connection
func (status *status) handleMessage(stream network.Stream, msg *StatusMessage) {
	ctx := context.Background()
	peer := stream.Conn().RemotePeer()

	// check if valid status message
	if status.validMessage(msg) {

		// update peer confirmed status message time
		status.peerConfirmed[peer] = time.Now()

		// update peer status message (StatusMessage stored to generate PeerInfo)
		status.peerMessage[peer] = msg

		// manage status message expiration
		go status.manageExpiration(ctx, peer)

	} else {

		// close connection with peer if status message is not valid
		err := status.closePeer(ctx, peer)
		if err != nil {
			log.Error("Failed to close peer with invalid status message", "err", err)
		}
	}
}

// validMessage confirms the status message is valid
func (status *status) validMessage(msg *StatusMessage) bool {
	switch {
	case msg.GenesisHash != status.hostMessage.GenesisHash:
		log.Debug("Failed to validate status message", "err", "genesis hash")
		return false
	case msg.ProtocolVersion < status.hostMessage.MinSupportedVersion:
		log.Debug("Failed to validate status message", "err", "protocol version")
		return false
	case msg.MinSupportedVersion > status.hostMessage.ProtocolVersion:
		log.Debug("Failed to validate status message", "err", "protocol version")
		return false
	}
	return true
}

// manageExpiration closes peer connection if status message has exipred
func (status *status) manageExpiration(ctx context.Context, peer peer.ID) {

	// wait to check status message
	time.Sleep(ExpireStatusInterval)

	// get time of last confirmed status message
	lastConfirmed := status.peerConfirmed[peer]

	// check if status message has expired
	if time.Since(lastConfirmed) > ExpireStatusInterval {

		// update peer information and close connection
		err := status.closePeer(ctx, peer)
		if err != nil {
			log.Error("Failed to close peer with expired status message", "err", err)
		}
	}
}

// closePeer updates status state and closes the connection
func (status *status) closePeer(ctx context.Context, peer peer.ID) error {

	// cancel running processes
	ctx.Done()

	// update peer status
	status.peerConfirmed[peer] = time.Time{}
	status.peerMessage[peer] = nil

	// close connection with peer
	err := status.host.closePeer(peer)

	return err
}
