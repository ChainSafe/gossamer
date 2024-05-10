package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/internal/pubip"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/variadic"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	lip2pnetwork "github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"

	ma "github.com/multiformats/go-multiaddr"
)

type chainSpec struct {
	Bootnodes  []string `json:"bootnodes"`
	ProtocolID string   `json:"protocolId"`
}

func setupP2PClient() host.Host {
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

	// Set your own keypair
	priv, _, err := crypto.GenerateKeyPair(
		crypto.Ed25519, // Select your key type. Ed25519 are nice short
		-1,             // Select key length when possible (i.e. RSA).
	)
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

func buildRequestMessage(arg string) *network.BlockRequestMessage {
	params := strings.Split(arg, ",")
	targetBlock := parseTargetBlock(params[0])

	if len(params) == 1 {
		return network.NewBlockRequest(targetBlock, 1,
			network.BootstrapRequestData, network.Ascending)
	}

	amount, err := strconv.Atoi(params[2])
	if err != nil || amount < 0 {
		log.Fatalf("could not parse the amount of blocks, expected positive number got: %s", params[2])
	}

	switch strings.ToLower(params[1]) {
	case "asc":
		return network.NewBlockRequest(targetBlock, uint32(amount),
			network.BootstrapRequestData, network.Ascending)
	case "desc":
		return network.NewBlockRequest(targetBlock, uint32(amount),
			network.BootstrapRequestData, network.Descending)
	}

	log.Fatalf("not supported direction: %s, use 'asc' for ascending or 'desc' for descending", params[1])
	return nil
}

func parseTargetBlock(arg string) variadic.Uint32OrHash {
	var value any
	value, err := strconv.Atoi(arg)
	if err != nil {
		value = common.MustHexToHash(arg)
	}

	v, err := variadic.NewUint32OrHash(value)
	if err != nil {
		log.Fatalf("\ncannot parse variadic type: %s", err.Error())
	}

	return *v
}

func parseChainSpec(arg string) *chainSpec {
	rawChainSpec, err := os.ReadFile(arg)
	if err != nil {
		log.Fatalf("reading %s file: %s", arg, err.Error())
	}

	cs := &chainSpec{}
	err = json.Unmarshal(rawChainSpec, cs)
	if err != nil {
		log.Fatalf("unmarshaling json: %s", err.Error())
	}

	return cs
}

func parseBootnodes(bootnodes []string) []peer.AddrInfo {
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

func writeMessageOnStream(msg *network.BlockRequestMessage, stream lip2pnetwork.Stream) {
	encMsg, err := msg.Encode()
	if err != nil {
		log.Fatalf("encoding message: %s", err.Error())
	}

	msgLen := uint64(len(encMsg))
	lenBytes := network.Uint64ToLEB128(msgLen)
	encMsg = append(lenBytes, encMsg...)

	_, err = stream.Write(encMsg)
	if err != nil {
		log.Fatalf("writting message: %s", err.Error())
	}
}

func readStream(stream lip2pnetwork.Stream) []byte {
	responseBuf := make([]byte, network.MaxBlockResponseSize)

	length, _, err := network.ReadLEB128ToUint64(stream)
	if err != nil {
		log.Fatalf("reading response length: %s", err.Error())
	}

	if length == 0 {
		return nil
	}

	if length > network.MaxBlockResponseSize {
		log.Fatalf("%s: max %d, got %d", network.ErrGreaterThanMaxSize, network.MaxBlockResponseSize, length)
	}

	if length > uint64(len(responseBuf)) {
		extraBytes := int(length) - len(responseBuf)
		responseBuf = append(responseBuf, make([]byte, extraBytes)...)
	}

	var tot int
	for tot < int(length) {
		n, err := stream.Read(responseBuf[tot:])
		if err != nil {
			log.Fatalf("reading stream: %s", err.Error())
		}
		tot += n
	}

	if tot != int(length) {
		log.Fatalf("%s: expected %d bytes, received %d bytes", network.ErrFailedToReadEntireMessage, length, tot)
	}

	return responseBuf[:tot]
}

func waitAndStoreResponse(stream lip2pnetwork.Stream, outputFile string) bool {
	output := readStream(stream)
	if len(output) == 0 {
		return false
	}

	blockResponse := &network.BlockResponseMessage{}
	err := blockResponse.Decode(output)
	if err != nil {
		log.Fatalf("could not decode block response message: %s", err.Error())
	}

	resultOutput := strings.Builder{}
	resultOutput.WriteString(fmt.Sprintf("retrieved %d blocks", len(blockResponse.BlockData)))
	if len(blockResponse.BlockData) > 1 {
		resultOutput.WriteString(fmt.Sprintf(" from block #%d to block #%d", blockResponse.BlockData[0].Header.Number,
			blockResponse.BlockData[len(blockResponse.BlockData)-1].Header.Number))
	}

	log.Println(resultOutput.String())
	os.WriteFile(outputFile, []byte(common.BytesToHex(output)), os.ModePerm)
	return true
}

/*
This is a script to query the block data from a running peer node.

Once the node has started and processed the block whose state you need, can execute the script like so:
 1. go run retrieve_block.go <block hash | block number> <genesis file with bootnodes> <destionation file>
*/
func main() {
	if len(os.Args) != 4 {
		log.Fatalf(`
		script usage:
			go run retrieve_block.go [number or hash] [network chain spec] [output file]
			go run retrieve_block.go [number or hash],[direction],[number of blocks] [network chain spec] [output file]`)
	}

	requestMessage := buildRequestMessage(os.Args[1])
	p2pHost := setupP2PClient()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	chain := parseChainSpec(os.Args[2])
	bootnodes := parseBootnodes(chain.Bootnodes)
	protocolID := protocol.ID(fmt.Sprintf("/%s/sync/2", chain.ProtocolID))

	for _, bootnodesAddr := range bootnodes {
		err := p2pHost.Connect(ctx, bootnodesAddr)
		if err != nil {
			continue
		}

		log.Printf("requesting from peer %s\n", bootnodesAddr.String())
		stream, err := p2pHost.NewStream(ctx, bootnodesAddr.ID, protocolID)
		if err != nil {
			log.Printf("WARN: failed to create stream using protocol %s: %s", protocolID, err.Error())
		}

		defer stream.Close()
		writeMessageOnStream(requestMessage, stream)
		if !waitAndStoreResponse(stream, os.Args[3]) {
			continue
		}

		break
	}
}
