package main

import (
	"fmt"
	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v3"
)

func main() {
	api, err := gsrpc.NewSubstrateAPI("ws://127.0.0.1:8546")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Connected to gossamer API")

	hash, err := api.RPC.Chain.GetBlockHashLatest()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Hash: ", hash)
}
