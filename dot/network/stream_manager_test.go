package network

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/stretchr/testify/require"
)

func TestStreamManager(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	cleanupStreamInterval = time.Millisecond * 500
	defer func() {
		cleanupStreamInterval = time.Minute
		cancel()
	}()

	smA := newStreamManager(ctx)
	smB := newStreamManager(ctx)

	portA := 7001
	portB := 7002
	addrA, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", portA))
	require.NoError(t, err)
	addrB, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", portB))
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
		smB.start()
	})

	stream, err := ha.NewStream(ctx, hb.ID(), "")
	require.NoError(t, err)

	smA.logNewStream(stream)
	smA.start()

	time.Sleep(cleanupStreamInterval * 2)
	connsAToB := ha.Network().ConnsToPeer(hb.ID())
	require.Equal(t, 1, len(connsAToB))
	require.Equal(t, 0, len(connsAToB[0].GetStreams()))

	connsBToA := hb.Network().ConnsToPeer(ha.ID())
	require.Equal(t, 1, len(connsBToA))
	require.Equal(t, 0, len(connsBToA[0].GetStreams()))
}
