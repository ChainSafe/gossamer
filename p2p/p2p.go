package p2p

import (
	"bufio"
	"context"
	"fmt"
	"os"

	"github.com/libp2p/go-libp2p"
	libp2pnet "github.com/libp2p/go-libp2p-net"
	libp2phost "github.com/libp2p/go-libp2p-host"
	//multiaddr "github.com/multiformats/go-multiaddr"
	libp2pdht "github.com/libp2p/go-libp2p-kad-dht"
)

func Start() (libp2phost.Host, error) {
	host, err := libp2p.New(context.Background())
	host.SetStreamHandler("/chat/1.1.0", handleStream)

	if err != nil {
		return nil, err
	}

	fmt.Println("Host created. We are:", host.ID())
	fmt.Println(host.Addrs())

	// start dht; used for peer discovery
	// each node keeps a local copy of the dht
	_, err = startDHT(host)
	if err != nil {
		return nil, err
	}

	return host, nil
}

// start the kademlia DHT 
func startDHT(host libp2phost.Host) (*libp2pdht.IpfsDHT, error) {
	dht, err := libp2pdht.New(context.Background(), host)
	if err != nil {
		return nil, err
	}

	return dht, nil
}

func handleStream(stream libp2pnet.Stream) {
    // Create a buffer stream for non blocking read and write.
    rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))

    go readData(rw)
    go writeData(rw)

    // 'stream' will stay open until you close it (or the other side closes it).
}

func readData(rw *bufio.ReadWriter) {
	for {
		str, err := rw.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading from buffer")
			panic(err)
		}

		if str == "" {
			return
		}
		if str != "\n" {
			fmt.Printf("\x1b[32m%s\x1b[0m> ", str)
		}

	}
}

func writeData(rw *bufio.ReadWriter) {
	stdReader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("> ")
		sendData, err := stdReader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading from stdin")
			panic(err)
		}

		_, err = rw.WriteString(fmt.Sprintf("%s\n", sendData))
		if err != nil {
			fmt.Println("Error writing to buffer")
			panic(err)
		}
		err = rw.Flush()
		if err != nil {
			fmt.Println("Error flushing buffer")
			panic(err)
		}
	}
}