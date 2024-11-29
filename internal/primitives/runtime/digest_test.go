// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package runtime

import (
	"testing"

	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
)

func TestEncodeDecodeDigest(t *testing.T) {
	digest := Digest{
		Logs: []DigestItem{
			NewDigestItem(PreRuntime{
				ConsensusEngineID: ConsensusEngineID{'F', 'R', 'N', 'K'},
				Bytes:             []byte("test"),
			}),
		},
	}

	encoded, err := scale.Marshal(digest)
	require.NoError(t, err)

	d := Digest{}
	err = scale.Unmarshal(encoded, &d)
	require.NoError(t, err)

	require.Equal(t, digest, d)
}
