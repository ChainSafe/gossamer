// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"context"
	"fmt"
	"time"

	"github.com/ChainSafe/gossamer/dot/peerset"
	libp2pnetwork "github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

type RequestResponseProtocol struct {
	ctx             context.Context
	host            *host
	requestTimeout  time.Duration
	maxResponseSize uint64
	protocolID      protocol.ID
}

func (rrp *RequestResponseProtocol) DoRequest(to peer.ID, req Message, res ResponseMessage) error {
	rrp.host.p2pHost.ConnManager().Protect(to, "")
	defer rrp.host.p2pHost.ConnManager().Unprotect(to, "")

	ctx, cancel := context.WithTimeout(rrp.ctx, rrp.requestTimeout)
	defer cancel()

	stream, err := rrp.host.p2pHost.NewStream(ctx, to, rrp.protocolID)
	if err != nil {
		return err
	}

	defer func() {
		err := stream.Close()
		if err != nil {
			logger.Warnf("failed to close stream: %s", err)
		}
	}()

	if err = rrp.host.writeToStream(stream, req); err != nil {
		return err
	}

	return rrp.ReceiveResponse(stream, res)
}

func (rrp *RequestResponseProtocol) ReceiveResponse(stream libp2pnetwork.Stream, msg ResponseMessage) error {
	buf := make([]byte, rrp.maxResponseSize)
	n, err := readStream(stream, &buf, rrp.maxResponseSize)
	if err != nil {
		return fmt.Errorf("read stream error: %w", err)
	}

	if n == 0 {
		return fmt.Errorf("received empty message")
	}

	err = msg.Decode(buf[:n])
	if err != nil {
		rrp.host.cm.peerSetHandler.ReportPeer(peerset.ReputationChange{
			Value:  peerset.BadMessageValue,
			Reason: peerset.BadMessageReason,
		}, stream.Conn().RemotePeer())
		return fmt.Errorf("failed to decode block response: %w", err)
	}

	return nil
}

type ResponseMessage interface {
	String() string
	Encode() ([]byte, error)
	Decode(in []byte) (err error)
}
