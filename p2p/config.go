package p2p

import (
	"fmt"

	"github.com/libp2p/go-libp2p"
	ma "github.com/multiformats/go-multiaddr"
)

// Config is used to configure a p2p service
type Config struct {
	BootstrapNodes []string
	Hostname           string // TODO: This MUST be an IP, probably need to convert "localhost" to IP
	Port           int
	RandSeed       int64
	NoBootstrap    bool
	NoMdns         bool
}

func (cfg *Config) buildOpts() ([]libp2p.Option, error) {
	priv, err := generateKey(cfg.RandSeed)
	if err != nil {
		return nil, err
	}

	addr, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", cfg.Hostname, cfg.Port))
	if err != nil {
		return nil, err
	}

	return []libp2p.Option{
		libp2p.ListenAddrs(addr),
		libp2p.DisableRelay(),
		libp2p.Identity(priv),
		libp2p.NATPortMap(),
		libp2p.Ping(true),
	}, nil
}
