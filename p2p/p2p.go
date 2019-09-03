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
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package p2p

import (
	"bufio"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	mrand "math/rand"
	"time"

	log "github.com/ChainSafe/log15"
	ds "github.com/ipfs/go-datastore"
	dsync "github.com/ipfs/go-datastore/sync"
	libp2p "github.com/libp2p/go-libp2p"
	core "github.com/libp2p/go-libp2p-core"
	crypto "github.com/libp2p/go-libp2p-core/crypto"
	host "github.com/libp2p/go-libp2p-core/host"
	net "github.com/libp2p/go-libp2p-core/network"
	peer "github.com/libp2p/go-libp2p-core/peer"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	discovery "github.com/libp2p/go-libp2p/p2p/discovery"
	rhost "github.com/libp2p/go-libp2p/p2p/host/routed"
)

const protocolPrefix2 = "/substrate/dot/2"
const protocolPrefix3 = "/substrate/dot/3"
const mdnsPeriod = time.Minute

// Service describes a p2p service, including host and dht
type Service struct {
	ctx            context.Context
	host           core.Host
	dht            *kaddht.IpfsDHT
	dhtConfig      kaddht.BootstrapConfig
	bootstrapNodes []peer.AddrInfo
	mdns           discovery.Service
	noBootstrap    bool
}

// NewService creates a new p2p.Service using the service config. It initializes the host and dht
func NewService(conf *Config) (*Service, error) {
	ctx := context.Background()
	opts, err := conf.buildOpts()
	if err != nil {
		return nil, err
	}

	h, err := libp2p.New(ctx, opts...)
	if err != nil {
		return nil, err
	}

	h.SetStreamHandler(protocolPrefix2, handleStream)
	h.SetStreamHandler(protocolPrefix3, handleStream)

	dstore := dsync.MutexWrap(ds.NewMapDatastore())
	dht := kaddht.NewDHT(ctx, h, dstore)

	// wrap the host with routed host so we can look up peers in DHT
	h = rhost.Wrap(h, dht)


	var mdns discovery.Service
	if !conf.NoMdns {
		mdns, err = discovery.NewMdnsService(ctx, h, mdnsPeriod, protocolPrefix3)
		if err != nil {
			return nil, err
		}

		mdns.RegisterNotifee(Notifee{ctx: ctx, host: h})
	}

	dhtConfig := kaddht.BootstrapConfig{
		Queries: 1,
		Period:  time.Second,
	}

	bootstrapNodes, err := stringsToPeerAddrInfos(conf.BootstrapNodes)
	s := &Service{
		ctx:            ctx,
		host:           h,
		dht:            dht,
		dhtConfig:      dhtConfig,
		bootstrapNodes: bootstrapNodes,
		noBootstrap:    conf.NoBootstrap,
		mdns:           mdns,
	}
	return s, err
}

// Start begins the p2p Service, including discovery
func (s *Service) Start() <-chan error {
	log.Debug("starting p2p service", "addrs", s.host.Addrs(), "id", s.host.ID())
	e := make(chan error)
	go s.start(e)
	return e
}

// start begins the p2p Service, including discovery. start does not terminate once called.
func (s *Service) start(e chan error) {
	if len(s.bootstrapNodes) == 0 && !s.noBootstrap {
		e <- errors.New("no peers to bootstrap to")
	}

	// TODO: Remove (along with this function), seems to do nothing
	// this is in a go func that loops every minute due to the fact that we appear
	// to get kicked off the network after a few minutes
	// this will likely be resolved once we send messages back to the network
	//go func() {
	//	for {
	//		if !s.noBootstrap {
	//			// connect to the bootstrap nodes
	//			err := s.bootstrapConnect()
	//			if err != nil {
	//				e <- err
	//			}
	//		}
	//
	//		err := s.dht.Bootstrap(s.ctx)
	//		if err != nil {
	//			e <- err
	//		}
	//		time.Sleep(time.Minute)
	//	}
	//}()

	log.Info("listening for connections...")
	e <- nil
}

// Stop stops the p2p service
func (s *Service) Stop() <-chan error {
	e := make(chan error)

	//Stop the host & IpfsDHT
	err := s.host.Close()
	if err != nil {
		e <- err
	}

	err = s.dht.Close()
	if err != nil {
		e <- err
	}

	return e
}

// Send sends a message to a specific peer
func (s *Service) Send(peer core.PeerAddrInfo, msg []byte) error {
	err := s.host.Connect(s.ctx, peer)
	if err != nil {
		return err
	}

	stream, err := s.host.NewStream(s.ctx, peer.ID, protocolPrefix3)
	if err != nil {
		return err
	}

	_, err = stream.Write(msg)
	if err != nil {
		return err
	}

	return nil
}

// Ping pings a peer
func (s *Service) Ping(peer core.PeerID) error {
	ps, err := s.dht.FindPeer(s.ctx, peer)
	if err != nil {
		return fmt.Errorf("could not find peer: %s", err)
	}

	err = s.host.Connect(s.ctx, ps)
	if err != nil {
		return err
	}

	return s.dht.Ping(s.ctx, peer)
}

// Host returns the service's host
func (s *Service) Host() host.Host {
	return s.host
}

// DHT returns the service's dht
func (s *Service) DHT() *kaddht.IpfsDHT {
	return s.dht
}

// Ctx returns the service's ctx
func (s *Service) Ctx() context.Context {
	return s.ctx
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

// handles stream; reads message length, message type, and decodes message based on type
// TODO: implement all message types; send message back to peer when we get a message; gossip for certain message types
func handleStream(stream net.Stream) {
	defer func() {
		if err := stream.Close(); err != nil {
			log.Error("error closing stream", "err", err)
		}
	}()

	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))
	lengthByte, err := rw.Reader.ReadByte()
	if err != nil {
		log.Error("stream handler", "got stream from", stream.Conn().RemotePeer(), "err", err)
	}

	// decode message length using LEB128
	length := LEB128ToUint64([]byte{lengthByte})
	log.Info("stream handler", "got message with length", length)

	// read entire message
	rawMsg, err := rw.Reader.Peek(int(length))
	if err != nil {
		log.Error("stream handler", "err", err)
	}

	log.Info("stream handler", "got stream from", stream.Conn().RemotePeer(), "message", fmt.Sprintf("%x", rawMsg))

	_, err = DecodeMessage(rw, length)
	if err != nil {
		log.Error("stream handler", "err", err)
	}

	//log.Info("stream handler", "got message from", stream.Conn().RemotePeer(), "message", msg.String())
}

// PeerCount returns the number of connected peers
func (s *Service) PeerCount() int {
	peers := s.host.Network().Peers()
	return len(peers)
}
