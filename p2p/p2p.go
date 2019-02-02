package p2p

import (
	"bufio"
	"context"
	"fmt"
	"os"

	"github.com/libp2p/go-libp2p"
	net "gx/ipfs/QmNgLg1NTw37iWbYPKcyK85YJ9Whs1MkPtJwhfqbNYAyKg/go-libp2p-net"
	//multiaddr "github.com/multiformats/go-multiaddr"
	//dht "github.com/libp2p/go-libp2p-kad-dht"
)

func Start() {
	host, err := libp2p.New(context.Background())
	host.SetStreamHandler("/chat/1.1.0", handleStream)

	if err != nil {
		panic(err)
	}
	
	fmt.Println("Host created. We are:", host.ID())
	fmt.Println(host.Addrs())

	// new dht client
	//dht, err := dht.New(context.Background(), host)
}

func handleStream(stream net.Stream) {
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
			// Green console colour: 	\x1b[32m
			// Reset console colour: 	\x1b[0m
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