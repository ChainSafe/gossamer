package config

import (
	libp2p "github.com/libp2p/go-libp2p/core"
)

type MultiaddrPeerId struct {
	libp2p.Multiaddr
	libp2p.PeerID
}
