// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p" //nolint
	libp2phost "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/stretchr/testify/require"
)

func setupStreamManagerTest(t *testing.T) (context.Context, []libp2phost.Host, []*streamManager) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())

	cleanupStreamInterval = time.Millisecond * 500
	t.Cleanup(func() {
		cleanupStreamInterval = time.Minute
		cancel()
	})

	smA := newStreamManager(ctx)
	smB := newStreamManager(ctx)

	portA := availablePort(t)
	portB := availablePort(t)

	addrA, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", portA))
	require.NoError(t, err)
	addrB, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", portB))
	require.NoError(t, err)

	ha, err := libp2p.New( //nolint
		ctx, libp2p.ListenAddrs(addrA), //nolint
	)
	require.NoError(t, err)

	hb, err := libp2p.New( //nolint
		ctx, libp2p.ListenAddrs(addrB), //nolint
	)
	require.NoError(t, err)

	err = ha.Connect(ctx, peer.AddrInfo{
		ID:    hb.ID(),
		Addrs: hb.Addrs(),
	})
	require.NoError(t, err)

	hb.SetStreamHandler("", func(stream network.Stream) {
		smB.logNewStream(stream)
	})

	return ctx, []libp2phost.Host{ha, hb}, []*streamManager{smA, smB}
}

func TestStreamManager(t *testing.T) {
	t.Parallel()

	ctx, hosts, sms := setupStreamManagerTest(t)
	ha, hb := hosts[0], hosts[1]
	smA, smB := sms[0], sms[1]

	stream, err := ha.NewStream(ctx, hb.ID(), "")
	require.NoError(t, err)

	smA.logNewStream(stream)
	smA.start()
	smB.start()

	time.Sleep(cleanupStreamInterval * 2)
	connsAToB := ha.Network().ConnsToPeer(hb.ID())
	require.GreaterOrEqual(t, len(connsAToB), 1)
	require.Equal(t, 0, len(connsAToB[0].GetStreams()))

	connsBToA := hb.Network().ConnsToPeer(ha.ID())
	require.GreaterOrEqual(t, len(connsBToA), 1)
	require.Equal(t, 0, len(connsBToA[0].GetStreams()))
}

func TestStreamManager_KeepStream(t *testing.T) {
	t.Skip() // TODO: test is flaky (#1026)
	ctx, hosts, sms := setupStreamManagerTest(t)
	ha, hb := hosts[0], hosts[1]
	smA, smB := sms[0], sms[1]

	stream, err := ha.NewStream(ctx, hb.ID(), "")
	require.NoError(t, err)

	smA.logNewStream(stream)
	smA.start()
	smB.start()

	time.Sleep(cleanupStreamInterval / 3)
	connsAToB := ha.Network().ConnsToPeer(hb.ID())
	require.GreaterOrEqual(t, len(connsAToB), 1)
	require.Equal(t, 1, len(connsAToB[0].GetStreams()))

	connsBToA := hb.Network().ConnsToPeer(ha.ID())
	require.GreaterOrEqual(t, len(connsBToA), 1)
	require.Equal(t, 1, len(connsBToA[0].GetStreams()))
}
