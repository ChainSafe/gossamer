package p2p

import (
	"fmt"
	"testing"

	peer "github.com/libp2p/go-libp2p-core/peer"
	libp2p "github.com/libp2p/go-libp2p"
	ma "github.com/multiformats/go-multiaddr"
)

func TestDaemon(t *testing.T) {
	ip := "0.0.0.0"

	addr, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", ip, 7070))
	if err != nil {
		t.Fatal(err)
	}

	opts := []libp2p.Option{
		libp2p.ListenAddrs(addr),
		//libp2p.DisableRelay(),
		//libp2p.NATPortMap(),
		//libp2p.Ping(true),
	}

	d, c, closer, _, err := createDaemonClientPair(opts)
	if err != nil {
		t.Fatal(err)
	}

	defer closer()
	
	fmt.Printf("ID: %s\n",d.ID().Pretty())
	fmt.Printf("DAEMON PEERLIST: %v\n", d.Addrs())

    bootstrapNodes := []string{
        "/ip4/104.211.54.233/tcp/30363/p2p/QmUghPWmHR8pQbZyBMeYzvPcH7VRcTiBibcyBG7wMKHaSZ",
        "/ip4/104.211.48.51/tcp/30363/p2p/QmYWrEtg4iQYwV9PG37PhfLHLATQJUTYiZRyoUvSYny9ba",
        "/ip4/104.211.48.247/tcp/30363/p2p/QmYT3p4qGj1jwb7hDx1A6cDzAPtaHp3VR34vmw5BsXXB8D",
        "/ip4/40.117.153.33/tcp/30363/p2p/QmPiGU1jwL9UDw2FMyMQFr9FdpF9hURKxkfy6PWw6aLsur",
    }

    maddrs := make([]ma.Multiaddr, len(bootstrapNodes))

    for i, addr := range bootstrapNodes {
    	maddr, err := ma.NewMultiaddr(addr)
		if err != nil {
			t.Fatal(err)
		}
		maddrs[i] = maddr
    }

    peerid, err := peer.IDFromString("QmUghPWmHR8pQbZyBMeYzvPcH7VRcTiBibcyBG7wMKHaSZ")
    if err != nil {
    	t.Fatal(err)
    }
    // bootstrapNodesMaddrs, err := stringsToPeerInfos(bootstrapNodes)
    // if err != nil {
    // 	t.Fatal(err)
    // }

    err = c.Connect(peerid, maddrs)
    if err != nil {
    	t.Fatal(err)
    }


}