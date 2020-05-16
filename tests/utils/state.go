package utils

import (
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/require"
	"testing"
)

// GetStorage calls the endpoint state_getStorage
func GetStorage(t *testing.T, node *Node, key []byte) []byte {
	respBody, err := PostRPC(t, StateGetStorage, NewEndpoint(HOSTNAME,node.RPCPort), "[\""+common.BytesToHex(key)+"\"]")
	require.NoError(t, err)

	v := new(string)
	err = DecodeRPC(t, respBody, v)
	require.NoError(t, err)
	if *v == "" {
		return []byte{}
	}

	value, err := common.HexToBytes(*v)
	require.NoError(t, err)

	return value
}

