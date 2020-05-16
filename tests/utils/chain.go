package utils

import (
	"errors"
	"fmt"
	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	log "github.com/ChainSafe/log15"
	"github.com/stretchr/testify/require"
	"math/big"
	"testing"
)

// GetHeader calls the endpoint chain_getHeader
func GetHeader(t *testing.T, node *Node, hash common.Hash) *types.Header { //nolint
	respBody, err := PostRPC(t, ChainGetHeader, NewEndpoint(HOSTNAME,node.RPCPort), "[\""+hash.String()+"\"]")
	require.NoError(t, err)

	header := new(modules.ChainBlockHeaderResponse)
	err = DecodeRPC(t, respBody, header)
	require.Nil(t, err)

	return HeaderResponseToHeader(t, header)
}

// GetChainHead calls the endpoint chain_getHeader to get the latest chain head
func GetChainHead(t *testing.T, node *Node) *types.Header {
	respBody, err := PostRPC(t, ChainGetHeader, NewEndpoint(HOSTNAME,node.RPCPort), "[]")
	require.NoError(t, err)

	header := new(modules.ChainBlockHeaderResponse)
	err = DecodeRPC(t, respBody, header)
	require.Nil(t, err)

	return HeaderResponseToHeader(t, header)
}


// GetBlock calls the endpoint chain_getBlock
func GetBlock(t *testing.T, node *Node, hash common.Hash) *types.Block {
	respBody, err := PostRPC(t, ChainGetBlock, NewEndpoint(HOSTNAME,node.RPCPort), "[\""+hash.String()+"\"]")
	require.NoError(t, err)

	block := new(modules.ChainBlockResponse)
	err = DecodeRPC(t, respBody, block)
	if err != nil {
		return nil
	}

	header := block.Block.Header

	parentHash, err := common.HexToHash(header.ParentHash)
	require.NoError(t, err)

	nb, err := common.HexToBytes(header.Number)
	require.NoError(t, err)
	number := big.NewInt(0).SetBytes(nb)

	stateRoot, err := common.HexToHash(header.StateRoot)
	require.NoError(t, err)

	extrinsicsRoot, err := common.HexToHash(header.ExtrinsicsRoot)
	require.NoError(t, err)

	digest := [][]byte{}

	for _, l := range header.Digest.Logs {
		var d []byte
		d, err = common.HexToBytes(l)
		require.NoError(t, err)
		digest = append(digest, d)
	}

	h, err := types.NewHeader(parentHash, number, stateRoot, extrinsicsRoot, digest)
	require.NoError(t, err)

	b, err := types.NewBodyFromExtrinsicStrings(block.Block.Body)
	require.NoError(t, err, fmt.Sprintf("%v", block.Block.Body))

	return &types.Block{
		Header: h,
		Body:   b,
	}
}


// CompareChainHeads calls getChainHead for each node in the array
// it returns a map of chainHead hashes to node key names, and an error if the hashes don't all match
func CompareChainHeads(t *testing.T, nodes []*Node) (map[common.Hash][]string, error) {
	hashes := make(map[common.Hash][]string)
	for _, node := range nodes {
		header := GetChainHead(t, node)
		log.Info("getting header from node", "header", header, "hash", header.Hash(), "node", node.Key)
		hashes[header.Hash()] = append(hashes[header.Hash()], node.Key)
	}

	var err error
	if len(hashes) != 1 {
		err = errors.New("node chain head hashes don't match")
	}

	return hashes, err
}
