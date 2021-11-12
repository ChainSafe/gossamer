// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package utils

import (
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/require"
)

// GetStorage calls the endpoint state_getStorage
func GetStorage(t *testing.T, node *Node, key []byte) []byte {
	respBody, err := PostRPC(StateGetStorage, NewEndpoint(node.RPCPort), "[\""+common.BytesToHex(key)+"\"]")
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
