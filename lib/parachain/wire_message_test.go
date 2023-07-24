// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachain

import (
	"fmt"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestDecodeWireMessage(t *testing.T) {
	decoded := NewWireMessageVDT()
	enc := []byte{2, 0, 0, 0, 0, 0}
	err := scale.Unmarshal(enc, &decoded)
	require.NoError(t, err)
	fmt.Printf("decode %v\n", decoded)
}
