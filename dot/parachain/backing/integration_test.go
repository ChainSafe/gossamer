package backing_test

import (
	"testing"

	availabilitystore "github.com/ChainSafe/gossamer/dot/parachain/availability-store"
	"github.com/ChainSafe/gossamer/dot/parachain/backing"
	"github.com/ChainSafe/gossamer/dot/parachain/overseer"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/erasure"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
)

/*
var tempSignature = common.MustHexToBytes("0xc67cb93bf0a36fcee3d29de8a6a69a759659680acf486475e0a2552a5fbed87e45adce5f290698d8596095722b33599227f7461f51af8617c8be74b894cf1b86") //nolint:lll

func dummyCandidateReceipt(t *testing.T) parachaintypes.CandidateReceipt {
	t.Helper()

	cr := getDummyCommittedCandidateReceipt(t).ToPlain()

	// blake2bhash of PVD in dummyPVD(t *testing.T) function
	cr.Descriptor.PersistedValidationDataHash =
		common.MustHexToHash("0x3544fbcdcb094751a5e044a30b994b2586ffc0b50e8b88c381461fe023a7242f")

	return cr
}

func getDummyCommittedCandidateReceipt(t *testing.T) parachaintypes.CommittedCandidateReceipt {
	t.Helper()
	hash5 := getDummyHash(t, 6)

	var collatorID parachaintypes.CollatorID
	tempCollatID := common.MustHexToBytes("0x48215b9d322601e5b1a95164cea0dc4626f545f98343d07f1551eb9543c4b147")
	copy(collatorID[:], tempCollatID)

	var collatorSignature parachaintypes.CollatorSignature
	copy(collatorSignature[:], tempSignature)

	ccr := parachaintypes.CommittedCandidateReceipt{
		Descriptor: parachaintypes.CandidateDescriptor{
			ParaID:                      uint32(1),
			RelayParent:                 hash5,
			Collator:                    collatorID,
			PersistedValidationDataHash: hash5,
			PovHash:                     hash5,
			ErasureRoot:                 hash5,
			Signature:                   collatorSignature,
			ParaHead:                    hash5,
			ValidationCodeHash:          parachaintypes.ValidationCodeHash(hash5),
		},
		Commitments: parachaintypes.CandidateCommitments{
			UpwardMessages:    []parachaintypes.UpwardMessage{{1, 2, 3}},
			NewValidationCode: &parachaintypes.ValidationCode{1, 2, 3},
			HeadData: parachaintypes.HeadData{
				Data: []byte{1, 2, 3},
			},
			ProcessedDownwardMessages: uint32(5),
			HrmpWatermark:             uint32(0),
		},
	}

	return ccr
}

func dummyPVD(t *testing.T) parachaintypes.PersistedValidationData {
	t.Helper()

	return parachaintypes.PersistedValidationData{
		ParentHead: parachaintypes.HeadData{
			Data: []byte{1, 2, 3},
		},
		RelayParentNumber:      5,
		RelayParentStorageRoot: getDummyHash(t, 5),
		MaxPovSize:             3,
	}
}
*/

// register the backing subsystem, run backing subsystem, start overseer
func initBackingAndOverseerMock(t *testing.T) (*backing.CandidateBacking, *overseer.MockableOverseer) {
	t.Helper()

	overseerMock := overseer.NewMockableOverseer(t)

	backing := backing.New(overseerMock.SubsystemsToOverseer)
	backing.OverseerToSubSystem = overseerMock.RegisterSubsystem(backing)
	backing.SubSystemToOverseer = overseerMock.GetSubsystemToOverseerChannel()

	overseerMock.Start()

	return backing, overseerMock
}

func getDummyHash(t *testing.T, num byte) common.Hash {
	t.Helper()

	hash := common.Hash{}
	for i := 0; i < 32; i++ {
		hash[i] = num
	}
	return hash
}

func dummyPVD(t *testing.T) parachaintypes.PersistedValidationData {
	t.Helper()

	return parachaintypes.PersistedValidationData{
		ParentHead:             parachaintypes.HeadData{Data: []byte{7, 8, 9}},
		RelayParentNumber:      0,
		RelayParentStorageRoot: getDummyHash(t, 0),
		MaxPovSize:             1024,
	}
}

func makeErasureRoot(
	t *testing.T,
	numOfValidators uint,
	pov parachaintypes.PoV,
	pvd parachaintypes.PersistedValidationData,
) common.Hash {
	t.Helper()

	availableData := availabilitystore.AvailableData{
		PoV:            pov,
		ValidationData: pvd,
	}

	dataBytes, err := scale.Marshal(availableData)
	require.NoError(t, err)

	chunks, err := erasure.ObtainChunks(numOfValidators, dataBytes)
	require.NoError(t, err)

	trie, err := erasure.ChunksToTrie(chunks)
	root, err := trie.Hash()
	require.NoError(t, err)

	return root
}

// newCommittedCandidate creates a new committed candidate receipt for testing purposes.
func newCommittedCandidate(
	t *testing.T,
	paraID uint32,
	headData parachaintypes.HeadData,
	povHash, relayParent, erasureRoot, pvdHash common.Hash,
	validationCode parachaintypes.ValidationCode,
) parachaintypes.CommittedCandidateReceipt {
	t.Helper()

	var collatorID parachaintypes.CollatorID
	var collatorSignature parachaintypes.CollatorSignature

	headDataHash, err := headData.Hash()
	require.NoError(t, err)

	validationCodeHash, err := common.Blake2bHash(validationCode)
	require.NoError(t, err)

	return parachaintypes.CommittedCandidateReceipt{
		Descriptor: parachaintypes.CandidateDescriptor{
			ParaID:                      paraID,
			RelayParent:                 relayParent,
			Collator:                    collatorID,
			PersistedValidationDataHash: pvdHash,
			PovHash:                     povHash,
			ErasureRoot:                 erasureRoot,
			Signature:                   collatorSignature,
			ParaHead:                    headDataHash,
			ValidationCodeHash:          parachaintypes.ValidationCodeHash(validationCodeHash),
		},
		Commitments: parachaintypes.CandidateCommitments{
			UpwardMessages:            []parachaintypes.UpwardMessage{},
			HorizontalMessages:        []parachaintypes.OutboundHrmpMessage{},
			NewValidationCode:         nil,
			HeadData:                  headData,
			ProcessedDownwardMessages: 0,
			HrmpWatermark:             0,
		},
	}
}

/*
func TestSecondsValidCandidate(t *testing.T) {
	candidateBacking, _ := initBackingAndOverseerMock(t)

	pov1 := parachaintypes.PoV{BlockData: []byte{42, 43, 44}}
	pvd1 := dummyPVD(t)
	validationCode1 := parachaintypes.ValidationCode{1, 2, 3}

	pov1Hash, err := pov1.Hash()
	require.NoError(t, err)

	pvd1Hash, err := pvd1.Hash()
	require.NoError(t, err)

	// pov2 := parachaintypes.PoV{BlockData: []byte{45, 46, 47}}
	// pvd2 := dummyPVD(t)

	// pvd2.ParentHead.Data = []byte{14, 15, 16}
	// pvd2.MaxPovSize = pvd2.MaxPovSize / 2
	// validationCode2 := parachaintypes.ValidationCode{4, 5, 6}

	// not sure
	var (
		paraID          uint32      = 1
		relayParent     common.Hash = getDummyHash(t, 5)
		numOfValidators uint        = 6
	)

	// default values
	var (
		headData parachaintypes.HeadData
	)

	candidate1 := newCommittedCandidate(
		t,
		paraID,
		headData,
		pov1Hash,
		relayParent,
		makeErasureRoot(t, numOfValidators, pov1, pvd1),
		pvd1Hash,
		validationCode1,
	)

	second1 := backing.SecondMessage{
		RelayParent:             relayParent,
		CandidateReceipt:        candidate1.ToPlain(),
		PersistedValidationData: pvd1,
		PoV:                     pov1,
	}

	candidateBacking.SubSystemToOverseer <- second1

	time.Sleep(10 * time.Minute)
}
*/
