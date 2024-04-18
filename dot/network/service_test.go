// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
)

var (
	testPeerID         = peer.ID("kishan")
	testMessageTimeout = time.Second * 3
)

func TestFreeNetworkEventsChannel(t *testing.T) {
	t.Parallel()

	config := &Config{
		BasePath:    t.TempDir(),
		Port:        availablePort(t),
		NoBootstrap: true,
		NoMDNS:      true,
	}

	node := createTestService(t, config)
	ch := node.GetNetworkEventsChannel()

	require.Equal(t, 1, len(node.networkEventInfoChannels))

	node.FreeNetworkEventsChannel(ch)
	require.Equal(t, 1, len(node.networkEventInfoChannels))

}

func TestGetNetworkEventsChannel(t *testing.T) {
	t.Parallel()

	config := &Config{
		BasePath:    t.TempDir(),
		Port:        availablePort(t),
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeA := createTestService(t, config)

	ch := nodeA.GetNetworkEventsChannel()
	defer nodeA.FreeNetworkEventsChannel(ch)

	configB := &Config{
		BasePath:    t.TempDir(),
		Port:        availablePort(t),
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeB := createTestService(t, configB)

	addrInfoB := addrInfo(nodeB.host)
	err := nodeA.host.connect(addrInfoB)
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = nodeA.host.connect(addrInfoB)
	}
	require.NoError(t, err)

	// let's disconnect peer B
	nodeA.processMessage(peerset.Message{
		Status: peerset.Drop,
		PeerID: addrInfoB.ID,
	})

	// now, let's connect peer B again
	nodeA.processMessage(peerset.Message{
		Status: peerset.Connect,
		PeerID: addrInfoB.ID,
	})
	for i := 0; i < 2; i++ {
		select {
		case <-ch:
		case <-time.After(testMessageTimeout):
			t.Fatal("did not any network event")
		}
	}
}
