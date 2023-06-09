package parachain

import (
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
)

func TestStatement(t *testing.T) {
	hash1 := common.Hash{}
	for i := 0; i < 32; i++ {
		hash1[i] = 1
	}

	testCases := []struct {
		name          string
		enumValue     scale.VaryingDataTypeValue
		encodingValue []byte
	}{
		{
			name:          "Seconded",
			enumValue:     Seconded{},
			encodingValue: []byte{},
		},
		{
			name:          "Valid",
			enumValue:     Valid{Value: hash1},
			encodingValue: []byte{2, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
		},
	}

	for _, c := range testCases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			vtd := NewStatement()

			err := vtd.Set(c.enumValue)
			require.NoError(t, err)

			bytes, err := scale.Marshal(vtd)
			require.NoError(t, err)

			require.Equal(t, c.encodingValue, bytes)
		})
	}
}
