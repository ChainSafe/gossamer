// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachain

import (
	"fmt"
	"testing"

	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
)

func TestDecodeWireMessage(t *testing.T) {
	decoded := NewWireMessageVDT()
	// sample messages from plokadot host
	//enc := []byte{2, 0, 0, 0, 0, 0}
	enc := []byte{2, 4, 1, 15, 190, 85, 58, 60, 125, 210, 153, 173, 240, 97, 225, 33, 196, 131, 95, 237, 230, 93, 245,
		57, 10, 182, 30, 150, 57, 162, 184, 190, 118, 178, 0, 0, 0}
	err := scale.Unmarshal(enc, &decoded)
	require.NoError(t, err)
	fmt.Printf("decode %v\n", decoded)
}
