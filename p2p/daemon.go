package p2p

import (
	"context"
	"fmt"

	libp2p "github.com/libp2p/go-libp2p"
	p2pd "github.com/libp2p/go-libp2p-daemon"
	c "github.com/libp2p/go-libp2p-daemon/p2pclient"
	ma "github.com/multiformats/go-multiaddr"
)

var (
	bindIP string = "0.0.0.0"
	portStartPoint int = 30300
)

func createDaemonClientPair(opts []libp2p.Option) (*p2pd.Daemon, *c.Client, func(), context.Context, error) {
	ctx, _:= context.WithCancel(context.Background())

	dAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d",bindIP,portStartPoint))
	cmaddr, _ := ma.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d",bindIP,portStartPoint+1))

	daemon, err := p2pd.NewDaemon(ctx, dAddr, "", opts...)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	client, err := c.NewClient(daemon.Listener().Multiaddr(), cmaddr)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	closer := func() {
		_ = client.Close()
		_ = daemon.Close()
	}
	
	return daemon, client, closer, ctx, nil
}