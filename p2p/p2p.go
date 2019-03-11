package p2p

import (
	"bufio"
	"context"
	"fmt"
	//"os"

	ma "github.com/multiformats/go-multiaddr"
	libp2p "github.com/libp2p/go-libp2p"
	host "github.com/libp2p/go-libp2p-host"
	net "github.com/libp2p/go-libp2p-net"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	rhost "github.com/libp2p/go-libp2p/p2p/host/routed"
)

const protocolPrefix = "/polkadot/0.0.0"

type Service struct {
	ctx 			context.Context
	host 			host.Host
	dht 			*kaddht.IpfsDHT
	bootstrapNode 	string
}

type ServiceConfig struct {
	bootstrapNode 	string
	port 			int
}

// NewService creates a new p2p.Service using the service config. It initializes the host and dht
func NewService(conf *ServiceConfig) (*Service, error) {
	ctx := context.Background()
	opts, err := conf.buildOpts()
	if err != nil {
		return nil, err
	}

	h, err := libp2p.New(ctx, opts...)
	if err != nil {
		return nil, err
	}

	h.SetStreamHandler(protocolPrefix, handleStream)

	dht, err := kaddht.New(ctx, h)
	if err != nil {
		return nil, err
	}

	// wrap the host with routed host so we can look up peers in DHT
	h = rhost.Wrap(h, dht)

	return &Service {
		ctx: ctx,
		host: h,
		dht: dht,
		bootstrapNode: conf.bootstrapNode,
	}, nil
}

// Start begins the p2p Service, including discovery
func (s *Service) Start() error {
	fmt.Println("Host created. We are:", s.host.ID())
	fmt.Println(s.host.Addrs())

	return nil
}

// Stop stops the p2p service
func (s *Service) Stop() {

}

func (sc *ServiceConfig) buildOpts() ([]libp2p.Option, error) {
	// TODO: get external ip
	ip := "0.0.0.0"

	addr, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", ip, sc.port))
	if err != nil {
		return nil, err
	}

	return []libp2p.Option{
		libp2p.ListenAddrs(addr),
		libp2p.EnableRelay(),
	}, nil
}

// start DHT discovery
func (s *Service) startDHT() (error) {
	return nil
}

// TODO: stream handling
func handleStream(stream net.Stream) {
	// Create a buffer stream for non blocking read and write.
	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))

	go readData(rw)
	go writeData(rw)

	// 'stream' will stay open until you close it (or the other side closes it).
}

func readData(rw *bufio.ReadWriter) {
	// for {
	// 	str, err := rw.ReadString('\n')
	// 	if err != nil {
	// 		fmt.Println("Error reading from buffer")
	// 		panic(err)
	// 	}

	// 	if str == "" {
	// 		return
	// 	}
	// 	if str != "\n" {
	// 		fmt.Printf("\x1b[32m%s\x1b[0m> ", str)
	// 	}

	// }
}

func writeData(rw *bufio.ReadWriter) {
	// stdReader := bufio.NewReader(os.Stdin)

	// for {
	// 	fmt.Print("> ")
	// 	sendData, err := stdReader.ReadString('\n')
	// 	if err != nil {
	// 		fmt.Println("Error reading from stdin")
	// 		panic(err)
	// 	}

	// 	_, err = rw.WriteString(fmt.Sprintf("%s\n", sendData))
	// 	if err != nil {
	// 		fmt.Println("Error writing to buffer")
	// 		panic(err)
	// 	}
	// 	err = rw.Flush()
	// 	if err != nil {
	// 		fmt.Println("Error flushing buffer")
	// 		panic(err)
	// 	}
	// }
}
