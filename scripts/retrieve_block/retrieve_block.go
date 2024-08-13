// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/ChainSafe/gossamer/dot/network/messages"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/variadic"
	"github.com/ChainSafe/gossamer/scripts/p2p"
	lip2pnetwork "github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/protocol"
)

func buildRequestMessage(arg string) *messages.BlockRequestMessage {
	params := strings.Split(arg, ",")
	targetBlock := parseTargetBlock(params[0])

	if len(params) == 1 {
		return messages.NewBlockRequest(targetBlock, 1,
			messages.BootstrapRequestData, messages.Ascending)
	}

	amount, err := strconv.Atoi(params[2])
	if err != nil || amount < 0 {
		log.Fatalf("could not parse the amount of blocks, expected positive number got: %s", params[2])
	}

	switch strings.ToLower(params[1]) {
	case "asc":
		return messages.NewBlockRequest(targetBlock, uint32(amount),
			messages.BootstrapRequestData, messages.Ascending)
	case "desc":
		return messages.NewBlockRequest(targetBlock, uint32(amount),
			messages.BootstrapRequestData, messages.Descending)
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

func waitAndStoreResponse(stream lip2pnetwork.Stream, outputFile string) bool {
	output, err := p2p.ReadStream(stream)
	if len(output) == 0 {
		return false
	}

	if err != nil {
		log.Println(err.Error())
		return false
	}

	blockResponse := &messages.BlockResponseMessage{}
	err = blockResponse.Decode(output)
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
	err = os.WriteFile(outputFile, []byte(common.BytesToHex(output)), os.ModePerm)
	if err != nil {
		log.Fatalf("failed to write response to file %s: %s", outputFile, err.Error())
	}
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

	p2pHost := p2p.SetupP2PClient()
	chain := p2p.ParseChainSpec(os.Args[2])
	bootnodes := p2p.ParseBootnodes(chain.Bootnodes)

	requestMessage := buildRequestMessage(os.Args[1])

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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

		defer stream.Close() //nolint:errcheck
		err = p2p.WriteStream(requestMessage, stream)
		if err != nil {
			log.Println(err.Error())
			continue
		}

		if !waitAndStoreResponse(stream, os.Args[3]) {
			continue
		}

		break
	}
}
