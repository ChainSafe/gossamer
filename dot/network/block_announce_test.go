// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/require"
)

func Test_BlockAnnounceMessage_String(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		messageBuilder func() BlockAnnounceMessage
		s              string
	}{
		"empty": {
			messageBuilder: func() BlockAnnounceMessage {
				return BlockAnnounceMessage{}
			},
			s: "BlockAnnounceMessage " +
				"ParentHash=0x0000000000000000000000000000000000000000000000000000000000000000 " +
				"Number=0 " +
				"StateRoot=0x0000000000000000000000000000000000000000000000000000000000000000 " +
				"ExtrinsicsRoot=0x0000000000000000000000000000000000000000000000000000000000000000 " +
				"Digest=[]",
		},
		"filled": {
			messageBuilder: func() BlockAnnounceMessage {
				digest := types.NewDigest()
				digest.Add(types.PreRuntimeDigest{
					ConsensusEngineID: types.ConsensusEngineID{'a', 'b', 'c', 'd'},
					Data:              []byte{1, 2, 3, 4},
				})
				return BlockAnnounceMessage{
					ParentHash:     common.Hash{1},
					Number:         2,
					StateRoot:      common.Hash{3},
					ExtrinsicsRoot: common.Hash{4},
					Digest:         digest,
					BestBlock:      true,
				}
			},

			s: "BlockAnnounceMessage " +
				"ParentHash=0x0100000000000000000000000000000000000000000000000000000000000000 " +
				"Number=2 " +
				"StateRoot=0x0300000000000000000000000000000000000000000000000000000000000000 " +
				"ExtrinsicsRoot=0x0400000000000000000000000000000000000000000000000000000000000000 " +
				"Digest=[PreRuntimeDigest ConsensusEngineID=abcd Data=0x01020304]",
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			message := testCase.messageBuilder()
			s := message.String()
			require.Equal(t, testCase.s, s)
		})
	}
}
