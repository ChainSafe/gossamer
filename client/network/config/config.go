package config

import "github.com/libp2p/go-libp2p/core"

type MultiaddrPeerId struct {
	core.Multiaddr
	core.PeerID
}
