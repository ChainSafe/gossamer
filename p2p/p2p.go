package p2p

import (
	"bufio"
	"context"
	"crypto/rand"
	"fmt"
	mrand "math/rand"
	"io"

	crypto "github.com/libp2p/go-libp2p-crypto"
	ds "github.com/ipfs/go-datastore"
	dsync "github.com/ipfs/go-datastore/sync"
	iaddr "github.com/ipfs/go-ipfs-addr"
	libp2p "github.com/libp2p/go-libp2p"
	host "github.com/libp2p/go-libp2p-host"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	net "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	ps "github.com/libp2p/go-libp2p-peerstore"
	rhost "github.com/libp2p/go-libp2p/p2p/host/routed"
	ma "github.com/multiformats/go-multiaddr"
)

const protocolPrefix = "/polkadot/0.0.0"

// Service defines a p2p service, including host and dht
type Service struct {
	ctx           context.Context
	host          host.Host
	hostAddr		ma.Multiaddr
	dht           *kaddht.IpfsDHT
	bootstrapNode string
}

// ServiceConfig is used to initialize a new p2p service
type ServiceConfig struct {
	BootstrapNode 	string
	Port          	int
	RandSeed 		int64
}

// NewService creates a new p2p.Service using the service config. It initializes the host and dht
func NewService(conf *ServiceConfig) (*Service, error) {
	ctx := context.Background()
	opts, err := conf.buildOpts()
	if err != nil {
		return nil, err
	}

	h, err := libp2p.New(ctx, opts...)
	if err != nil {
		return nil, err
	}

	h.SetStreamHandler(protocolPrefix, handleStream)

	dstore := dsync.MutexWrap(ds.NewMapDatastore())
	dht := kaddht.NewDHT(ctx, h, dstore)

	// wrap the host with routed host so we can look up peers in DHT
	h = rhost.Wrap(h, dht)

	// fmt.Println("Host created. We are:", s.host.ID().Pretty())
	// fmt.Println(s.host.Addrs())

	// build host multiaddress
	hostAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/ipfs/%s", h.ID().Pretty()))

	return &Service{
		ctx:           ctx,
		host:          h,
		hostAddr:		hostAddr,
		dht:           dht,
		bootstrapNode: conf.BootstrapNode,
	}, nil
}

// Start begins the p2p Service, including discovery
func (s *Service) Start() error {
	ipfsPeers, err := stringsToPeerInfos(IPFS_PEERS)
	if err != nil {
		return err
	}

	// connect to the chosen ipfs nodes
	err = bootstrapConnect(s.ctx, s.host, ipfsPeers)
	if err != nil {
		return err
	}

	// bootstrap the host
	err = s.dht.Bootstrap(s.ctx)
	if err != nil {
		return err
	}

	return nil
}

// Stop stops the p2p service
func (s *Service) Stop() {
	// TODO
}

// Broadcast sends a message to all peers
func (s *Service) Broadcast() {
	// TODO
}

// Send sends a message to a specific peer
func (s *Service) Send(peer peer.ID) {
	// TODO
}

// Ping pings a peer
func (s *Service) Ping(peer peer.ID) {
	// TODO
}

func (sc *ServiceConfig) buildOpts() ([]libp2p.Option, error) {
	// TODO: get external ip
	ip := "0.0.0.0"

	priv, err := generateKey(sc.RandSeed)
	if err != nil {
		return nil, err
	}

	addr, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", ip, sc.Port))
	if err != nil {
		return nil, err
	}

	return []libp2p.Option{
		libp2p.ListenAddrs(addr),
		libp2p.EnableRelay(),
		libp2p.Identity(priv),
	}, nil
}

// generateKey generates a libp2p private key which is used for secure messaging
func generateKey(seed int64) (crypto.PrivKey, error) {
	// If the seed is zero, use real cryptographic randomness. Otherwise, use a
	// deterministic randomness source to make generated keys stay the same
	// across multiple runs
	var r io.Reader
	if seed == 0 {
		r = rand.Reader
	} else {
		r = mrand.New(mrand.NewSource(seed))
	}

	// Generate a key pair for this host. We will use it at least
	// to obtain a valid host ID.
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		return nil, err
	}

	return priv, nil
}

// start DHT discovery
func (s *Service) startDHT() error {
	err := s.dht.Bootstrap(s.ctx)
	if err != nil {
		return err
	}

	addr, err := iaddr.ParseString(s.bootstrapNode)
	if err != nil {
		return err
	}

	peerinfo, err := ps.InfoFromP2pAddr(addr.Multiaddr())
	if err != nil {
		return err
	}

	err = s.host.Connect(s.ctx, *peerinfo)
	return err
}

// TODO: stream handling
func handleStream(stream net.Stream) {
	// Create a buffer stream for non blocking read and write.
	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))
	str, err := rw.ReadString('\n')
	if err != nil {
		return
	}

	fmt.Println("got stream: ", str)
}
