package types

import (
	"testing"

	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
)

func TestCompactStatement_Codec(t *testing.T) {
	t.Parallel()

	// with
	compactStatement, err := NewCompactStatement()
	require.NoError(t, err)
	compactStatementList := scale.NewVaryingDataTypeSlice(scale.VaryingDataType(compactStatement))
	err = compactStatementList.Add(ValidCompactStatement{CandidateHash: getRandomHash()},
		SecondedCompactStatement{CandidateHash: getRandomHash()},
	)
	require.NoError(t, err)

	// when
	encoded, err := scale.Marshal(compactStatementList)
	require.NoError(t, err)

	// then
	decoded := scale.NewVaryingDataTypeSlice(scale.VaryingDataType(compactStatement))
	err = scale.Unmarshal(encoded, &decoded)
	require.NoError(t, err)
	require.Equal(t, compactStatementList, decoded)
}
