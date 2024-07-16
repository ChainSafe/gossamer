// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package retrieve_state

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/ChainSafe/gossamer/dot/network/messages"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/scripts/p2p"
	"github.com/libp2p/go-libp2p-core/protocol"
)

func main() {
	targetBlock := common.MustHexToHash("0x70e1b2c35aad9fd5af576dd00d05789a9e37ecbc486be075fec1b3b5592f5c80")

	p2pHost := p2p.SetupP2PClient()
	chain := p2p.ParseChainSpec(os.Args[1])
	bootnodes := p2p.ParseBootnodes(chain.Bootnodes)

	requestMessage := &messages.StateRequest{
		Block:   targetBlock,
		Start:   [][]byte{},
		NoProof: true,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	protocolID := protocol.ID(fmt.Sprintf("/%s/state/2", chain.ProtocolID))

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
		p2p.WriteStream(requestMessage, stream)

		// ============================
		output := p2p.ReadStream(stream)
		if len(output) == 0 {
			continue
		}

		stateResponse := &messages.StateRequest{}
		// ============================

		break
	}
}
