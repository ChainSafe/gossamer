// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package variadic

import (
	"encoding/binary"
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/require"
)

func TestNewUint32OrHash(t *testing.T) {
	hash, err := common.HexToHash("0xdcd1346701ca8396496e52aa2785b1748deb6db09551b72159dcb3e08991025b")
	require.NoError(t, err)

	res, err := NewUint32OrHash(hash)
	require.NoError(t, err)
	require.Equal(t, res.Value(), hash)

	num := 77

	res, err = NewUint32OrHash(num)
	require.NoError(t, err)
	require.Equal(t, uint32(num), res.Value())

	res, err = NewUint32OrHash(uint32(num))
	require.NoError(t, err)
	require.Equal(t, uint32(num), res.Value())
}

func TestNewUint32OrHashFromBytes(t *testing.T) {
	genesisHash, err := common.HexToBytes("0xdcd1346701ca8396496e52aa2785b1748deb6db09551b72159dcb3e08991025b")
	require.NoError(t, err)

	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, uint32(1))

	for _, x := range []struct {
		description     string
		targetHash      []byte
		targetFirstByte uint8
		expectedType    interface{}
	}{
		{
			description:     "block request with genesis hash type 0",
			targetHash:      genesisHash,
			targetFirstByte: 0,
			expectedType:    common.Hash{},
		},
		{
			description:     "block request with Block Number int type 1",
			targetHash:      buf,
			targetFirstByte: 1,
			expectedType:    (uint32)(0),
		},
	} {
		t.Run(x.description, func(t *testing.T) {
			data := append([]byte{x.targetFirstByte}, x.targetHash...)

			val := NewUint32OrHashFromBytes(data)
			require.NoError(t, err)
			require.IsType(t, x.expectedType, val.Value())

			if x.expectedType == (uint32)(0) {
				startingBlockByteArray := make([]byte, 4)
				binary.LittleEndian.PutUint32(startingBlockByteArray, val.Value().(uint32))
				require.Equal(t, x.targetHash, startingBlockByteArray)
			} else {
				require.Equal(t, common.NewHash(x.targetHash), val.Value())
			}
		})
	}
}
