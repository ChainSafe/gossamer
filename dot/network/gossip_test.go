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
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/lib/utils"

	"github.com/stretchr/testify/require"
)

// test gossip messages to connected peers
func TestGossip(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping TestGossip; currently, nothing is gossiped")
	}

	basePathA := utils.NewTestBasePath(t, "nodeA")

	configA := &Config{
		BasePath:    basePathA,
		Port:        7001,
		RandSeed:    1,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeA := createTestService(t, configA)
	handlerA := newTestStreamHandler(testBlockAnnounceMessageDecoder)
	nodeA.host.registerStreamHandler("", handlerA.handleStream)

	basePathB := utils.NewTestBasePath(t, "nodeB")
	configB := &Config{
		BasePath:    basePathB,
		Port:        7002,
		RandSeed:    2,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeB := createTestService(t, configB)
	handlerB := newTestStreamHandler(testBlockAnnounceMessageDecoder)
	nodeB.host.registerStreamHandler("", handlerB.handleStream)

	addrInfosA, err := nodeA.host.addrInfos()
	require.NoError(t, err)

	err = nodeB.host.connect(*addrInfosA[0])
	// retry connect if "failed to dial" error
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = nodeB.host.connect(*addrInfosA[0])
	}
	require.NoError(t, err)

	basePathC := utils.NewTestBasePath(t, "nodeC")
	configC := &Config{
		BasePath:    basePathC,
		Port:        7003,
		RandSeed:    3,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeC := createTestService(t, configC)
	handlerC := newTestStreamHandler(testBlockAnnounceMessageDecoder)
	nodeC.host.registerStreamHandler("", handlerC.handleStream)

	err = nodeC.host.connect(*addrInfosA[0])
	// retry connect if "failed to dial" error
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = nodeC.host.connect(*addrInfosA[0])
	}
	require.NoError(t, err)

	addrInfosB, err := nodeB.host.addrInfos()
	require.NoError(t, err)

	err = nodeC.host.connect(*addrInfosB[0])
	// retry connect if "failed to dial" error
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = nodeC.host.connect(*addrInfosB[0])
	}
	require.NoError(t, err)

	err = nodeA.host.send(addrInfosB[0].ID, "", testBlockAnnounceMessage)
	require.NoError(t, err)

	time.Sleep(TestMessageTimeout)

	if hasSeenB, ok := nodeB.gossip.seen.Load(testBlockAnnounceMessage.Hash()); !ok || hasSeenB.(bool) == false {
		t.Error(
			"node B did not receive block request message from node A",
			"\nreceived:", hasSeenB,
			"\nexpected:", true,
		)
	}

	if hasSeenC, ok := nodeC.gossip.seen.Load(testBlockAnnounceMessage.Hash()); !ok || hasSeenC.(bool) == false {
		t.Error(
			"node C did not receive block request message from node B",
			"\nreceived:", hasSeenC,
			"\nexpected:", true,
		)
	}

	if hasSeenA, ok := nodeA.gossip.seen.Load(testBlockAnnounceMessage.Hash()); !ok || hasSeenA.(bool) == false {
		t.Error(
			"node A did not receive block request message from node C",
			"\nreceived:", hasSeenA,
			"\nexpected:", true,
		)
	}
}
