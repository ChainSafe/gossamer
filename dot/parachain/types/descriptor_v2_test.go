package parachaintypes

import (
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
)

func candidateDescriptor(t *testing.T) CandidateDescriptor {
	t.Helper()

	return CandidateDescriptor{
		ParaID:                      1000,
		RelayParent:                 common.MustHexToHash("0xded542bacb3ca6c033a57676f94ae7c8f36834511deb44e3164256fd3b1c0de0"), //nolint:lll
		Collator:                    collatorID(t),
		PersistedValidationDataHash: common.MustHexToHash("0x690d8f252ef66ab0f969c3f518f90012b849aa5ac94e1752c5e5ae5a8996de37"), //nolint:lll
		PovHash:                     common.MustHexToHash("0xe7df1126ac4b4f0fb1bc00367a12ec26ca7c51256735a5e11beecdc1e3eca274"), //nolint:lll
		ErasureRoot:                 common.MustHexToHash("0xc07f658163e93c45a6f0288d229698f09c1252e41076f4caa71c8cbc12f118a1"), //nolint:lll
		ParaHead:                    common.MustHexToHash("0x9a8a7107426ef873ab89fc8af390ec36bdb2f744a9ff71ad7f18a12d55a7f4f5"), //nolint:lll
	}
}

func collatorID(t *testing.T) CollatorID {
	t.Helper()

	return CollatorID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16,
		17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31}
}

func TestCandidateDescriptorV2_rebuildCollatorField(t *testing.T) {
	t.Parallel()

	descriptorV1 := CandidateDescriptor{
		Collator: collatorID(t),
	}

	descriptorV2 := descriptorV1.V2()
	collatorField := descriptorV2.RebuildCollatorField()

	require.Equal(t, descriptorV1.Collator[:], collatorField[:])

}

func TestCandidateDescriptorV2_BackwardCompatibility(t *testing.T) {
	t.Parallel()

	descriptorV1 := candidateDescriptor(t)
	descriptorV2 := descriptorV1.V2()

	encodedV1, err := scale.Marshal(descriptorV1)
	require.NoError(t, err)

	encodedV2, err := scale.Marshal(descriptorV2)
	require.NoError(t, err)

	require.Equal(t, encodedV1, encodedV2)
}
