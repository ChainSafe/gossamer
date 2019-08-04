package p2p

import (
	"context"

	log "github.com/ChainSafe/log15"

	"github.com/libp2p/go-libp2p-core/connmgr"
	net "github.com/libp2p/go-libp2p-core/network"
	peer "github.com/libp2p/go-libp2p-core/peer"
)

type ConnManager struct{}

// Notifee is used to monitor changes to a connection
// Currently, we only implemented notifications for OpenedStream and ClosedStream
func (cm ConnManager) Notifee() net.Notifiee {
	nb := new(net.NotifyBundle)
	nb.OpenedStreamF = OpenedStream
	nb.ClosedStreamF = ClosedStream
	return nb
}

func (_ ConnManager) TagPeer(peer.ID, string, int)             {}
func (_ ConnManager) UntagPeer(peer.ID, string)                {}
func (_ ConnManager) UpsertTag(peer.ID, string, func(int) int) {}
func (_ ConnManager) GetTagInfo(peer.ID) *connmgr.TagInfo      { return &connmgr.TagInfo{} }
func (_ ConnManager) TrimOpenConns(ctx context.Context)        {}
func (_ ConnManager) Protect(peer.ID, string)                  {}
func (_ ConnManager) Unprotect(peer.ID, string) bool           { return false }
func (_ ConnManager) Close() error                             { return nil }

func OpenedStream(n net.Network, s net.Stream) {
	if string(s.Protocol()) == protocolPrefix2 || s.Protocol() == protocolPrefix3 {
		log.Info("opened stream", "peer", s.Conn().RemotePeer(), "protocol", s.Protocol())
	}
}

func ClosedStream(n net.Network, s net.Stream) {
	if string(s.Protocol()) == protocolPrefix2 || s.Protocol() == protocolPrefix3 {
		log.Info("closed stream", "peer", s.Conn().RemotePeer(), "protocol", s.Protocol())
	}
}
