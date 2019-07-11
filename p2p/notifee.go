package p2p

import (
	"context"
	
	log "github.com/inconshreveable/log15"
	host "github.com/libp2p/go-libp2p-host"
	peer "github.com/libp2p/go-libp2p-core/peer"
)

type Notifee struct {
	ctx  context.Context
	host host.Host
}

func (n Notifee) HandlePeerFound(p peer.AddrInfo) {
	log.Info("mdns", "peer found", p)
	err := n.host.Connect(n.ctx, p)
	if err != nil {
		log.Error("mdns", "connect error", err)
	}
}