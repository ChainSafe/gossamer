// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package main

import (
	"testing"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/variadic"
	"github.com/stretchr/testify/require"
)

func TestBuildRequestMessage(t *testing.T) {
	testcases := []struct {
		arg      string
		expected *network.BlockRequestMessage
	}{
		{
			arg: "10",
			expected: network.NewBlockRequest(
				*variadic.MustNewUint32OrHash(uint(10)), 1,
				network.BootstrapRequestData, network.Ascending),
		},
		{
			arg: "0x9b0211aadcef4bb65e69346cfd256ddd2abcb674271326b08f0975dac7c17bc7",
			expected: network.NewBlockRequest(*variadic.MustNewUint32OrHash(
				common.MustHexToHash("0x9b0211aadcef4bb65e69346cfd256ddd2abcb674271326b08f0975dac7c17bc7")),
				1, network.BootstrapRequestData, network.Ascending),
		},
		{
			arg: "0x9b0211aadcef4bb65e69346cfd256ddd2abcb674271326b08f0975dac7c17bc7,asc,20",
			expected: network.NewBlockRequest(*variadic.MustNewUint32OrHash(
				common.MustHexToHash("0x9b0211aadcef4bb65e69346cfd256ddd2abcb674271326b08f0975dac7c17bc7")),
				20, network.BootstrapRequestData, network.Ascending),
		},
		{
			arg: "1,asc,20",
			expected: network.NewBlockRequest(*variadic.MustNewUint32OrHash(uint(1)),
				20, network.BootstrapRequestData, network.Ascending),
		},
		{
			arg: "0x9b0211aadcef4bb65e69346cfd256ddd2abcb674271326b08f0975dac7c17bc7,desc,20",
			expected: network.NewBlockRequest(*variadic.MustNewUint32OrHash(
				common.MustHexToHash("0x9b0211aadcef4bb65e69346cfd256ddd2abcb674271326b08f0975dac7c17bc7")),
				20, network.BootstrapRequestData, network.Descending),
		},
		{
			arg: "1,desc,20",
			expected: network.NewBlockRequest(*variadic.MustNewUint32OrHash(uint(1)),
				20, network.BootstrapRequestData, network.Descending),
		},
	}

	for _, tt := range testcases {
		message := buildRequestMessage(tt.arg)
		require.Equal(t, tt.expected, message)
	}
}
