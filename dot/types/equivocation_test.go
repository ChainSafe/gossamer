package types

import (
	"testing"

	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
)

func TestEquivocationProof(t *testing.T) {
	exp := []byte{
		0, 91, 50, 25, 214, 94, 119, 36, 71, 216, 33, 152,
		85, 184, 34, 120, 61, 161, 164, 223, 76, 53, 40,
		246, 76, 38, 235, 204, 43, 31, 179, 28, 1, 0, 0,
		0, 0, 0, 0, 0}

	dec := EquivocationProof{}
	err := scale.Unmarshal(exp, &dec)
	require.NoError(t, err)

	enc, err := scale.Marshal(dec)
	require.NoError(t, err)
	require.Equal(t, exp, enc)
}
