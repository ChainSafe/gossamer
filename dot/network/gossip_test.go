// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"

	"github.com/stretchr/testify/require"
)

// test gossip messages to connected peers
func TestGossip(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping TestGossip; currently, nothing is gossiped")
	}

	t.Parallel()

	configA := &Config{
		BasePath:    t.TempDir(),
		Port:        availablePort(t),
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeA := createTestService(t, configA)
	handlerA := newTestStreamHandler(testBlockAnnounceMessageDecoder)
	nodeA.host.registerStreamHandler(nodeA.host.protocolID, handlerA.handleStream)

	configB := &Config{
		BasePath:    t.TempDir(),
		Port:        availablePort(t),
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeB := createTestService(t, configB)
	handlerB := newTestStreamHandler(testBlockAnnounceMessageDecoder)
	nodeB.host.registerStreamHandler(nodeB.host.protocolID, handlerB.handleStream)

	addrInfoA := nodeA.host.addrInfo()
	err := nodeB.host.connect(addrInfoA)
	// retry connect if "failed to dial" error
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = nodeB.host.connect(addrInfoA)
	}
	require.NoError(t, err)

	configC := &Config{
		BasePath:    t.TempDir(),
		Port:        availablePort(t),
		NoBootstrap: true,
		NoMDNS:      true,
	}

	nodeC := createTestService(t, configC)
	handlerC := newTestStreamHandler(testBlockAnnounceMessageDecoder)
	nodeC.host.registerStreamHandler(nodeC.host.protocolID, handlerC.handleStream)

	err = nodeC.host.connect(addrInfoA)
	// retry connect if "failed to dial" error
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = nodeC.host.connect(addrInfoA)
	}
	require.NoError(t, err)

	addrInfoB := nodeB.host.addrInfo()
	err = nodeC.host.connect(addrInfoB)
	// retry connect if "failed to dial" error
	if failedToDial(err) {
		time.Sleep(TestBackoffTimeout)
		err = nodeC.host.connect(addrInfoB)
	}
	require.NoError(t, err)

	announceMessage := &BlockAnnounceMessage{
		Number: 128 * 7,
		Digest: types.NewDigest(),
	}

	_, err = nodeA.host.send(addrInfoB.ID, "", announceMessage)
	require.NoError(t, err)

	time.Sleep(TestMessageTimeout)

	hash, err := announceMessage.Hash()
	require.NoError(t, err)

	_, ok := nodeB.gossip.seenMap[hash]
	require.True(t, ok, "node B did not receive block request message from node A")

	_, ok = nodeC.gossip.seenMap[hash]
	require.True(t, ok, "node C did not receive block request message from node B")

	_, ok = nodeA.gossip.seenMap[hash]
	require.True(t, ok, "node A did not receive block request message from node C")
}
