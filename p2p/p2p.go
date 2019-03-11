package p2p

import (
	"bufio"
	"context"
	"fmt"
	//"os"

	libp2p "github.com/libp2p/go-libp2p"
	host "github.com/libp2p/go-libp2p-host"
	net "github.com/libp2p/go-libp2p-net"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
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
}

// NewService creates a new p2p.Service using the config. It initializes the host and dht
func NewService(conf *ServiceConfig) (*Service, error) {
	h, err := libp2p.New(context.Background())
	if err != nil {
		return nil, err
	}

	h.SetStreamHandler(protocolPrefix, handleStream)

	dht, err := startDHT(h)
	if err != nil {
		return nil, err
	}

	return &Service {
		ctx: context.Background(),
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

// start dht; dht used for peer discovery. it keeps a list of peers in the network
// each node keeps a local copy of the dht
func startDHT(host host.Host) (*kaddht.IpfsDHT, error) {
	dht, err := kaddht.New(context.Background(), host)
	if err != nil {
		return nil, err
	}

	return dht, nil
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
