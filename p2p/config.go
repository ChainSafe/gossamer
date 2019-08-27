package p2p

import (
	"fmt"

	log "github.com/ChainSafe/log15"
	"github.com/libp2p/go-libp2p"
)

// Config is used to configure a p2p service
type Config struct {
	BootstrapNodes []string
	Hostname       string // TODO: This MUST be an IP, probably need to convert "localhost" to IP
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

	addr := fmt.Sprintf("/ip4/%s/tcp/%d", cfg.Hostname, cfg.Port)
	log.Debug("configuring node address", "addr", addr)

	return []libp2p.Option{
		libp2p.ListenAddrStrings(addr),
		libp2p.DisableRelay(),
		libp2p.Identity(priv),
		libp2p.NATPortMap(),
		libp2p.Ping(true),
	}, nil
}
