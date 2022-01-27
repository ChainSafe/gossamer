// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p"
	libp2phost "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/stretchr/testify/require"
)

func setupStreamManagerTest(t *testing.T, cleanupStreamInterval time.Duration) (context.Context, []libp2phost.Host, []*streamManager) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())

	t.Cleanup(func() {
		cancel()
	})

	smA := newStreamManager(ctx, cleanupStreamInterval)
	smB := newStreamManager(ctx, cleanupStreamInterval)

	addrA, err := ma.NewMultiaddr("/ip4/0.0.0.0/tcp/0")
	require.NoError(t, err)
	addrB, err := ma.NewMultiaddr("/ip4/0.0.0.0/tcp/0")
	require.NoError(t, err)

	ha, err := libp2p.New(
		ctx, libp2p.ListenAddrs(addrA),
	)
	require.NoError(t, err)

	hb, err := libp2p.New(
		ctx, libp2p.ListenAddrs(addrB),
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

	const interval = time.Millisecond * 50
	ctx, hosts, sms := setupStreamManagerTest(t, interval)

	ha, hb := hosts[0], hosts[1]
	smA, smB := sms[0], sms[1]

	stream, err := ha.NewStream(ctx, hb.ID(), "")
	require.NoError(t, err)

	smA.logNewStream(stream)
	smA.start()
	smB.start()

	time.Sleep(interval * 2)
	connsAToB := ha.Network().ConnsToPeer(hb.ID())
	require.GreaterOrEqual(t, len(connsAToB), 1)
	require.Equal(t, 0, len(connsAToB[0].GetStreams()))

	connsBToA := hb.Network().ConnsToPeer(ha.ID())
	require.GreaterOrEqual(t, len(connsBToA), 1)
	require.Equal(t, 0, len(connsBToA[0].GetStreams()))
}

func TestStreamManager_KeepStream(t *testing.T) {
	t.Skip() // TODO: test is flaky (#1026)
	const interval = time.Millisecond * 50

	ctx, hosts, sms := setupStreamManagerTest(t, interval)
	ha, hb := hosts[0], hosts[1]
	smA, smB := sms[0], sms[1]

	stream, err := ha.NewStream(ctx, hb.ID(), "")
	require.NoError(t, err)

	smA.logNewStream(stream)
	smA.start()
	smB.start()

	time.Sleep(interval / 3)
	connsAToB := ha.Network().ConnsToPeer(hb.ID())
	require.GreaterOrEqual(t, len(connsAToB), 1)
	require.Equal(t, 1, len(connsAToB[0].GetStreams()))

	connsBToA := hb.Network().ConnsToPeer(ha.ID())
	require.GreaterOrEqual(t, len(connsBToA), 1)
	require.Equal(t, 1, len(connsBToA[0].GetStreams()))
}
