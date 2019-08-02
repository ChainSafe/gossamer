package p2p

import (
	"context"

	"github.com/ChainSafe/gossamer/common"
	log "github.com/ChainSafe/log15"

	"github.com/libp2p/go-libp2p-core/connmgr"
	net "github.com/libp2p/go-libp2p-core/network"
	peer "github.com/libp2p/go-libp2p-core/peer"
)

type ConnManager struct{}

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
		msg, err := common.HexToBytes("0x000200000002000000043ac2250000000000e72b2144e0339cbf48ca6c076f021c84684745b9ddc2e40df5a562083b05d3cbdcd1346701ca8396496e52aa2785b1748deb6db09551b72159dcb3e08991025b0400")
		_, err = s.Write(msg)
		if err != nil {
			log.Error("sending stream", "error", err)
		}
	}
}

func ClosedStream(n net.Network, s net.Stream) {
	//log.Info("closed stream", "peer", s.Conn().RemotePeer(), "protocol", s.Protocol())
}
