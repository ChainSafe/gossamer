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

package network

import (
	"math/big"
	"os"
	"path"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/optional"
	"github.com/stretchr/testify/require"
)

// have a peer send a message status with a block ahead
// test exchanged messages after peer connected are correct
func TestSendBlockRequestMessage(t *testing.T) {
	dataDirA := path.Join(os.TempDir(), "gossamer-test", "nodeA")
	defer os.RemoveAll(dataDirA)

	configA := &Config{
		DataDir:     dataDirA,
		Port:        7001,
		RandSeed:    1,
		NoBootstrap: true,
		NoMdns:      true,
	}

	blockStateA := newMockBlockState(big.NewInt(3))
	nodeA, msgSendA, msgRecA := createTestServiceWithBlockState(t, configA, blockStateA)
	defer nodeA.Stop()

	nodeA.noGossip = true

	genesisHash, err := common.HexToHash("0xdcd1346701ca8396496e52aa2785b1748deb6db09551b72159dcb3e08991025b")
	if err != nil {
		t.Fatal(err)
	}

	bestBlockHash, err := common.HexToHash("0x829de6be9a35b55c794c609c060698b549b3064c183504c18ab7517e41255569")
	if err != nil {
		t.Fatal(err)
	}

	testStatusMessage := &StatusMessage{
		ProtocolVersion:     uint32(2),
		MinSupportedVersion: uint32(2),
		Roles:               byte(4),
		BestBlockNumber:     uint64(2434417),
		BestBlockHash:       bestBlockHash,
		GenesisHash:         genesisHash,
		ChainStatus:         []byte{0},
	}

	// simulate host status message sent from core service on startup
	msgRecA <- testStatusMessage

	dataDirB := path.Join(os.TempDir(), "gossamer-test", "nodeB")
	defer os.RemoveAll(dataDirB)

	configB := &Config{
		DataDir:     dataDirB,
		Port:        7002,
		RandSeed:    2,
		NoBootstrap: true,
		NoMdns:      true,
	}

	blockStateB := newMockBlockState(big.NewInt(1))
	nodeB, _, msgRecB := createTestServiceWithBlockState(t, configB, blockStateB)
	defer nodeB.Stop()

	nodeB.noGossip = true

	// simulate host status message sent from core service on startup
	msgRecB <- testStatusMessage

	addrInfosB, err := nodeB.host.addrInfos()
	if err != nil {
		t.Fatal(err)
	}

	err = nodeA.host.connect(*addrInfosB[0])
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(TestStatusTimeout)

	if !nodeA.status.confirmed(nodeB.host.h.ID()) {
		t.Error("node A did not confirm status of node B")
	}

	if !nodeB.status.confirmed(nodeA.host.h.ID()) {
		t.Error("node B did not confirm status of node A")
	}

	// get latest block header from block state
	latestHeader := blockStateB.LatestHeader()
	currentHash := blockStateB.LatestHeader().Hash()

	// expected block request message
	var expectedMessage = &BlockRequestMessage{
		RequestedData: 3,
		StartingBlock: append([]byte{0}, currentHash[:]...),
		EndBlockHash:  optional.NewHash(true, latestHeader.Hash()),
		Direction:     1,
		Max:           optional.NewUint32(false, 0),
	}

	select {
	case msg := <-msgSendA:
		require.NotNil(t, msg)

		// assert correct cast
		actualBlockRequest, ok := msg.(*BlockRequestMessage)
		require.True(t, ok)
		require.NotNil(t, actualBlockRequest)

		// assign ID since its random
		actualBlockRequest.ID = expectedMessage.ID

		// assert everything else
		require.Equal(t, expectedMessage, actualBlockRequest)

	case <-time.After(TestMessageTimeout):
		t.Error("node B timeout waiting for message from node A")
	}
}
