// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// wait time to discover and connect using mdns discovery
var TestMDNSTimeout = time.Second

// test mdns discovery service (discovers and connects)
func TestMDNS(t *testing.T) {
	t.Parallel()

	configA := &Config{
		BasePath:    t.TempDir(),
		Port:        availablePort(t),
		NoBootstrap: true,
	}

	nodeA := createTestService(t, configA)
	nodeA.noGossip = true

	configB := &Config{
		BasePath:    t.TempDir(),
		Port:        availablePort(t),
		NoBootstrap: true,
	}

	nodeB := createTestService(t, configB)
	nodeB.noGossip = true

	time.Sleep(TestMDNSTimeout)

	peerCountA := nodeA.host.peerCount()
	peerCountB := nodeB.host.peerCount()

	if peerCountA == 0 {
		// check peerstore for disconnected peers
		peerCountA := len(nodeA.host.p2pHost.Peerstore().Peers())
		require.NotZero(t, peerCountA)
	}

	if peerCountB == 0 {
		// check peerstore for disconnected peers
		peerCountB := len(nodeB.host.p2pHost.Peerstore().Peers())
		require.NotZero(t, peerCountB)
	}
}
