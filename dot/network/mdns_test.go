// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/stretchr/testify/require"
)

// wait time to discover and connect using mdns discovery
var TestMDNSTimeout = time.Second

// test mdns discovery service (discovers and connects)
func TestMDNS(t *testing.T) {
	t.Parallel()

	basePathA := utils.NewTestBasePath(t, "nodeA")

	configA := &Config{
		BasePath:    basePathA,
		Port:        uint16(availablePorts.get()),
		NoBootstrap: true,
	}

	nodeA := createTestService(t, configA)
	nodeA.noGossip = true

	basePathB := utils.NewTestBasePath(t, "nodeB")

	configB := &Config{
		BasePath:    basePathB,
		Port:        uint16(availablePorts.get()),
		NoBootstrap: true,
	}

	nodeB := createTestService(t, configB)
	nodeB.noGossip = true

	time.Sleep(TestMDNSTimeout)

	peerCountA := nodeA.host.peerCount()
	peerCountB := nodeB.host.peerCount()

	if peerCountA == 0 {
		// check peerstore for disconnected peers
		peerCountA := len(nodeA.host.h.Peerstore().Peers())
		require.NotZero(t, peerCountA)
	}

	if peerCountB == 0 {
		// check peerstore for disconnected peers
		peerCountB := len(nodeB.host.h.Peerstore().Peers())
		require.NotZero(t, peerCountB)
	}
}
