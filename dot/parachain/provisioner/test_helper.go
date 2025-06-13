package provisioner

import (
	"testing"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
)

func dummyCandidateDescriptorV2(relayParent common.Hash) *parachaintypes.CandidateDescriptorV2 {
	invalid := common.Hash{}

	return &parachaintypes.CandidateDescriptorV2{
		ParaID:                      parachaintypes.ParaID(1),
		RelayParent:                 relayParent,
		CurrentVersion:              0,
		CoreIndex:                   1,
		SessionIndex:                1,
		Reserved1:                   [25]uint8{},
		PersistedValidationDataHash: invalid,
		PovHash:                     invalid,
		ErasureRoot:                 invalid,
		Reserved2:                   [64]uint8{},
		ParaHead:                    invalid,
		ValidationCodeHash:          parachaintypes.ValidationCodeHash(invalid),
	}
}

func coreState(t *testing.T, coreStateValue any) parachaintypes.CoreState {
	t.Helper()

	cs := parachaintypes.CoreState{}
	err := cs.SetValue(coreStateValue)
	require.NoError(t, err)

	return cs
}

func occupiedCore(t *testing.T, paraID parachaintypes.ParaID) parachaintypes.OccupiedCore {
	t.Helper()

	descriptor := dummyCandidateDescriptorV2(common.Hash{0})
	descriptor.ParaID = paraID

	availability, err := parachaintypes.NewBitVec(make([]bool, 32))
	require.NoError(t, err)

	return parachaintypes.OccupiedCore{
		OccupiedSince:       100,
		TimeoutAt:           200,
		Availability:        availability,
		GroupResponsible:    parachaintypes.GroupIndex(paraID),
		CandidateDescriptor: *descriptor,
	}
}

func signedBitfield(
	t *testing.T,
	ks keystore.Keystore,
	field parachaintypes.BitVec,
	validatorIndex parachaintypes.ValidatorIndex,
) parachaintypes.CheckedSignedAvailabilityBitfield {
	t.Helper()

	keyring, err := keystore.NewSr25519Keyring()
	require.NoError(t, err)

	keyPair := keyring.Alice()
	err = ks.Insert(keyPair)
	require.NoError(t, err)

	publicKeyBytes := keyPair.Public().Encode()
	validatorID := parachaintypes.ValidatorID(publicKeyBytes)

	validator := parachaintypes.Validator{
		SigningContext: parachaintypes.SigningContext{},
		Key:            validatorID,
	}

	encoded, err := scale.Marshal(field)
	require.NoError(t, err)

	sign, err := validator.Sign(ks, encoded)
	require.NoError(t, err)

	return parachaintypes.CheckedSignedAvailabilityBitfield{
		Payload:        field,
		ValidatorIndex: validatorIndex,
		Signature:      *sign,
	}
}
