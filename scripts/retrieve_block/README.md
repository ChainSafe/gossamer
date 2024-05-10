## retrieve_block

The script is used to retrieve blocks from the network

### Usage

#### To retrieve a single block

```sh
go run retrieve_block.go [number or hash] [network chain spec] [output file]

# requesting a single block
go run retrieve_block.go 0x9b0211aadcef4bb65e69346cfd256ddd2abcb674271326b08f0975dac7c17bc7 ./westend.json file.out
```

#### To retrieve a chain of blocks (ascending or descending)

> _note that the arguments to request a chain of blocks are separated by comma_

> _the max numbers of blocks you can retrieve is 128. Given network limitations in the implementation the peer might not respond the full chain you've requested_

```sh
go run retrieve_block.go [number or hash],[direction],[number of blocks] [network chain spec] [output file]

# requesting from a chain of blocks from 10 to 13
go run retrieve_block.go 10,asc,3 ./westend.json file.out

# requesting from a chain of blocks from 50 to 30
go run retrieve_block.go 50,desc,21 ./westend.json file.out
```

The block response is written to the output file in the raw form, to decode it use

```go
rawBlockResponse, err := os.ReadFile(output_file)
if err != nil {
    panic(err)
}

blockResponse := &network.BlockResponseMessage{}
err := blockResponse.Decode(rawBlockResponse)
if err != nil {
    panic(err)
}

// iterate over the blocks
for _, b := range blockResponse.BlockData {

}
```
