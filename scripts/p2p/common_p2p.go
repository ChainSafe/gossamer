// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package p2p

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/network/messages"
	"github.com/ChainSafe/gossamer/internal/pubip"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	lip2pnetwork "github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

type ChainSpec struct {
	Bootnodes  []string `json:"bootnodes"`
	ProtocolID string   `json:"protocolId"`
}

func SetupP2PClient() host.Host {
	const port = 30333

	// create multiaddress (without p2p identity)
	listenAddress := fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port)
	addr, err := ma.NewMultiaddr(listenAddress)
	if err != nil {
		log.Fatalf("creating multiaddress: %s", err.Error())
	}

	var externalAddr ma.Multiaddr
	ip, err := pubip.Get()
	if err != nil {
		log.Fatalf("getting public ip: %s", err.Error())
	}

	externalAddr, err = ma.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", ip, port))
	if err != nil {
		log.Fatalf("creating external address: %s", err.Error())
	}

	priv, _, err := crypto.GenerateKeyPair(crypto.Ed25519, -1)
	if err != nil {
		log.Fatalf("generating keypair: %s", err.Error())
	}

	opts := []libp2p.Option{
		libp2p.ListenAddrs(addr),
		libp2p.DisableRelay(),
		libp2p.Identity(priv),
		libp2p.NATPortMap(),
		libp2p.AddrsFactory(func(as []ma.Multiaddr) []ma.Multiaddr {
			return append(as, externalAddr)
		}),
	}

	// create libp2p host instance
	p2pHost, err := libp2p.New(opts...)
	if err != nil {
		log.Fatalf("instantiating libp2p host: %s", err.Error())
	}

	return p2pHost
}

func ParseChainSpec(arg string) *ChainSpec {
	chainSpecFile, err := os.Open(filepath.Clean(arg))
	if err != nil {
		log.Fatalf("openning %s file: %s", arg, err.Error())
	}

	defer func() {
		_ = chainSpecFile.Close()
	}()

	rawChainSpec, err := io.ReadAll(chainSpecFile)
	if err != nil {
		log.Fatalf("reading %s file: %s", arg, err.Error())
	}

	cs := &ChainSpec{}
	err = json.Unmarshal(rawChainSpec, cs)
	if err != nil {
		log.Fatalf("unmarshaling json: %s", err.Error())
	}

	return cs
}

func ParseBootnodes(bootnodes []string) []peer.AddrInfo {
	addrs := make([]peer.AddrInfo, 0, len(bootnodes))
	for _, bn := range bootnodes {
		addrs = append(addrs, parsePeerAddress(bn))
	}

	return addrs
}

func parsePeerAddress(arg string) peer.AddrInfo {
	maddr, err := ma.NewMultiaddr(arg)
	if err != nil {
		log.Fatalf("parsing peer multiaddress: %s", err.Error())
	}
	p, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		log.Fatalf("getting address info: %s", err.Error())
	}
	return *p
}

var (
	errZeroLength = errors.New("zero length")
)

func ReadStream(stream lip2pnetwork.Stream) ([]byte, error) {
	responseBuf := make([]byte, network.MaxBlockResponseSize)

	length, _, err := network.ReadLEB128ToUint64(stream)
	if err != nil {
		return nil, fmt.Errorf("reading leb128: %w", err)
	}

	if length == 0 {
		return nil, errZeroLength
	}

	if length > network.MaxBlockResponseSize {
		return nil, fmt.Errorf("%w: max %d, got %d", network.ErrGreaterThanMaxSize, network.MaxBlockResponseSize, length)
	}

	if length > uint64(len(responseBuf)) {
		extraBytes := int(length) - len(responseBuf)
		responseBuf = append(responseBuf, make([]byte, extraBytes)...)
	}

	var tot int
	for tot < int(length) {
		n, err := stream.Read(responseBuf[tot:])
		if err != nil {
			return nil, fmt.Errorf("reading stream: %w", err)
		}
		tot += n
	}

	if tot != int(length) {
		return nil, fmt.Errorf("%w: expected %d bytes, received %d bytes", network.ErrFailedToReadEntireMessage, length, tot)
	}

	return responseBuf[:tot], nil
}

func WriteStream(msg messages.P2PMessage, stream lip2pnetwork.Stream) error {
	encMsg, err := msg.Encode()
	if err != nil {
		return fmt.Errorf("encoding message: %w", err)
	}

	msgLen := uint64(len(encMsg))
	lenBytes := network.Uint64ToLEB128(msgLen)
	encMsg = append(lenBytes, encMsg...)

	_, err = stream.Write(encMsg)
	if err != nil {
		return fmt.Errorf("writing message: %w", err)
	}

	return nil
}
