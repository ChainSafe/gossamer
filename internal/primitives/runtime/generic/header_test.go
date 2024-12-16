// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package generic

import (
	"testing"

	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
)

func TestEncodeDecodeHeader(t *testing.T) {
	header := NewHeader[uint64, hash.H256, runtime.BlakeTwo256](
		100,
		hash.H256(""),
		hash.H256(""),
		hash.H256(""),
		runtime.Digest{
			Logs: []runtime.DigestItem{
				runtime.NewDigestItem(runtime.PreRuntime{
					ConsensusEngineID: runtime.ConsensusEngineID{'F', 'R', 'N', 'K'},
					Bytes:             []byte("test"),
				}),
			},
		},
	)

	encoded, err := scale.Marshal(header)
	require.NoError(t, err)

	h := &Header[uint64, hash.H256, runtime.BlakeTwo256]{}
	err = scale.Unmarshal(encoded, h)
	require.NoError(t, err)

	require.Equal(t, header, h)
}
