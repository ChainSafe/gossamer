// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package messages

import (
	"log"
	"testing"

	_ "embed"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

//go:embed testdata/warp_sync_proofs.yaml
var rawWarpSyncProofs []byte

type WarpSyncProofs struct {
	SubstrateWarpSyncProof1 string `yaml:"substrate_warp_sync_proof_1"`
}

func TestDecodeWarpSyncProof(t *testing.T) {
	warpSyncProofs := &WarpSyncProofs{}
	err := yaml.Unmarshal(rawWarpSyncProofs, warpSyncProofs)
	require.NoError(t, err)

	// Generated using substrate
	expected := common.MustHexToBytes(warpSyncProofs.SubstrateWarpSyncProof1)
	if err != nil {
		log.Fatal(err)
	}

	var proof WarpSyncProof

	err = proof.Decode(expected)
	require.NoError(t, err)

	encoded, err := proof.Encode()
	require.NoError(t, err)
	require.Equal(t, expected, encoded)
}
