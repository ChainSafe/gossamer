// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package main

import (
	"testing"

	"github.com/ChainSafe/gossamer/dot/network/messages"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/variadic"
	"github.com/stretchr/testify/require"
)

func TestBuildRequestMessage(t *testing.T) {
	testcases := []struct {
		arg      string
		expected *messages.BlockRequestMessage
	}{
		{
			arg: "10",
			expected: messages.NewBlockRequest(
				*variadic.MustNewUint32OrHash(uint(10)), 1,
				messages.BootstrapRequestData, messages.Ascending),
		},
		{
			arg: "0x9b0211aadcef4bb65e69346cfd256ddd2abcb674271326b08f0975dac7c17bc7",
			expected: messages.NewBlockRequest(*variadic.MustNewUint32OrHash(
				common.MustHexToHash("0x9b0211aadcef4bb65e69346cfd256ddd2abcb674271326b08f0975dac7c17bc7")),
				1, messages.BootstrapRequestData, messages.Ascending),
		},
		{
			arg: "0x9b0211aadcef4bb65e69346cfd256ddd2abcb674271326b08f0975dac7c17bc7,asc,20",
			expected: messages.NewBlockRequest(*variadic.MustNewUint32OrHash(
				common.MustHexToHash("0x9b0211aadcef4bb65e69346cfd256ddd2abcb674271326b08f0975dac7c17bc7")),
				20, messages.BootstrapRequestData, messages.Ascending),
		},
		{
			arg: "1,asc,20",
			expected: messages.NewBlockRequest(*variadic.MustNewUint32OrHash(uint(1)),
				20, messages.BootstrapRequestData, messages.Ascending),
		},
		{
			arg: "0x9b0211aadcef4bb65e69346cfd256ddd2abcb674271326b08f0975dac7c17bc7,desc,20",
			expected: messages.NewBlockRequest(*variadic.MustNewUint32OrHash(
				common.MustHexToHash("0x9b0211aadcef4bb65e69346cfd256ddd2abcb674271326b08f0975dac7c17bc7")),
				20, messages.BootstrapRequestData, messages.Descending),
		},
		{
			arg: "1,desc,20",
			expected: messages.NewBlockRequest(*variadic.MustNewUint32OrHash(uint(1)),
				20, messages.BootstrapRequestData, messages.Descending),
		},
	}

	for _, tt := range testcases {
		message := buildRequestMessage(tt.arg)
		require.Equal(t, tt.expected, message)
	}
}
