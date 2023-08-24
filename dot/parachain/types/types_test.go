// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachaintypes

import (
	_ "embed"
	"fmt"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/westend.yaml
var testDataRaw string

var testData map[string]string

func init() {
	testData = make(map[string]string)
	err := yaml.Unmarshal([]byte(testDataRaw), &testData)
	if err != nil {
		fmt.Println("Error unmarshaling test data:", err)
		return
	}
}

// Test_Validators tests the scale encoding and decoding of a Validator struct
func Test_Validators(t *testing.T) {
	t.Parallel()

	resultHex := testData["validators"]
	if resultHex == "" {
		t.Fatal("cannot get test data for validators")
	}
	resultBytes, err := common.HexToBytes(resultHex)
	require.NoError(t, err)

	var validatorIDs []ValidatorID
	err = scale.Unmarshal(resultBytes, &validatorIDs)
	require.NoError(t, err)

	expected := []ValidatorID{
		mustHexTo32BArray(t, "0xa262f83b46310770ae8d092147176b8b25e8855bcfbbe701d346b10db0c5385d"),
		mustHexTo32BArray(t, "0x804b9df571e2b744d65eca2d4c59eb8e4345286c00389d97bfc1d8d13aa6e57e"),
		mustHexTo32BArray(t, "0x4eb63e4aad805c06dc924e2f19b1dde7faf507e5bb3c1838d6a3cfc10e84fe72"),
		mustHexTo32BArray(t, "0x74c337d57035cd6b7718e92a0d8ea6ef710da8ab1215a057c40c4ef792155a68"),
		mustHexTo32BArray(t, "0xe61d138eebd2069f1a76b3570f9de6a4b196289b198e33e6f0b59cef8837c511"),
		mustHexTo32BArray(t, "0x94ef34321ca5d37a6e8953183406b76f8ebf6a4be5eefc3997d022ac6e0a050e"),
		mustHexTo32BArray(t, "0xac837e8ca589521a83e7d9a7b307d1c41a5d9b940422488236f99646d21f3841"),
		mustHexTo32BArray(t, "0xb61cb85f7cf7616f9ef8f95010a51a68a4eae8afcdff715cc6a8d43da4a32a12"),
		mustHexTo32BArray(t, "0x382f17dae6b13a8ce5a7cc805056d9b592d918c8593f077db28cb14cf08a760c"),
		mustHexTo32BArray(t, "0x0825ba7677597ec9453ab5dbaa9e68bf89dc36694cb6e74cbd5a9a74b167e547"),
		mustHexTo32BArray(t, "0xcee3f65d78a239d7d199b100295e7a2d852ae898a6b81fd867b3471f25be7237"),
		mustHexTo32BArray(t, "0xe2ac8f039eb02370a9577e49ffc6032e6b5bf5ff77783bdc676d1432d714fd53"),
		mustHexTo32BArray(t, "0xce35fa64fe7a5a6fc456ed2830e64d5d1a5dba26e7a57ab458f8cedf1ec77016"),
		mustHexTo32BArray(t, "0xae40e895f46c8bfb3df63c119047d7faf21c3fe3e7a91994a3f00da6fa80f848"),
		mustHexTo32BArray(t, "0xa0e038975cff34d01c62960828c23ec10a305fe9f5c3589c2ae40f51963e380a"),
		mustHexTo32BArray(t, "0x807fa54347a8957ff5ef6c28e2403c83947e5fad4aa805c914df0645a07aab5a"),
		mustHexTo32BArray(t, "0x4c8e878d7f558ce5086cc37ca0d5964bed54ddd6b15a6663a95fe42e36858936"),
	}
	require.Equal(t, expected, validatorIDs)

	encoded, err := scale.Marshal(validatorIDs)
	require.NoError(t, err)
	require.Equal(t, resultHex, common.BytesToHex(encoded))
}

// Test_ValidatorGroup tests the validator group encoding and decoding.
func Test_ValidatorGroup(t *testing.T) {
	t.Parallel()

	result := testData["validatorGroups"]
	if result == "" {
		t.Fatal("cannot get test data for validatorGroups")
	}
	resultBytes, err := common.HexToBytes(result)
	require.NoError(t, err)

	validatorGroups := ValidatorGroups{
		Validators:        [][]ValidatorIndex{},
		GroupRotationInfo: GroupRotationInfo{},
	}
	err = scale.Unmarshal(resultBytes, &validatorGroups)
	require.NoError(t, err)

	expected := ValidatorGroups{
		Validators: [][]ValidatorIndex{{0, 1, 2, 3, 4, 5}, {6, 7, 8, 9, 10, 11}, {12, 13, 14, 15, 16}},
		GroupRotationInfo: GroupRotationInfo{
			SessionStartBlock:      15657314,
			GroupRotationFrequency: 10,
			Now:                    15657556,
		},
	}
	require.Equal(t, expected, validatorGroups)

	encoded, err := scale.Marshal(validatorGroups)
	require.NoError(t, err)

	require.Equal(t, result, common.BytesToHex(encoded))
}

// Test_AvailabilityCores tests the CoreState VDT encoding and decoding.
func Test_AvailabilityCores(t *testing.T) {
	t.Parallel()

	result := testData["availabilityCores"]
	if result == "" {
		t.Fatal("cannot get test data for availabilityCores")
	}
	resultBytes, err := common.HexToBytes(result)
	require.NoError(t, err)

	availabilityCores, err := NewAvailabilityCores()
	require.NoError(t, err)
	err = scale.Unmarshal(resultBytes, &availabilityCores)
	require.NoError(t, err)

	encoded, err := scale.Marshal(availabilityCores)
	require.NoError(t, err)
	require.Equal(t, result, common.BytesToHex(encoded))
}

// TestSessionIndex tests the SessionIndex encoding and decoding.
func TestSessionIndex(t *testing.T) {
	t.Parallel()

	result := testData["sessionIndex"]
	if result == "" {
		t.Fatal("could not find test data for session index")
	}
	resultBytes, err := common.HexToBytes(result)
	require.NoError(t, err)

	var sessionIndex SessionIndex
	err = scale.Unmarshal(resultBytes, &sessionIndex)
	require.NoError(t, err)

	require.Equal(t, SessionIndex(26895), sessionIndex)

	encoded, err := scale.Marshal(sessionIndex)
	require.NoError(t, err)
	require.Equal(t, result, common.BytesToHex(encoded))
}

// TestCommittedCandidateReceipt tests the CommittedCandidateReceipt encoding and decoding.
func TestCommittedCandidateReceipt(t *testing.T) {
	t.Parallel()

	result := testData["pendingAvailability"]
	if result == "" {
		t.Fatal("could not find test data for pending availability")
	}
	resultBytes, err := common.HexToBytes(result)
	require.NoError(t, err)

	var c *CommittedCandidateReceipt
	err = scale.Unmarshal(resultBytes, &c)
	require.NoError(t, err)

	// TODO: assert the fields

	encoded, err := scale.Marshal(c)
	require.NoError(t, err)
	require.Equal(t, result, common.BytesToHex(encoded))
}

// TestSessionInfo tests the SessionInfo encoding and decoding.
func TestSessionInfo(t *testing.T) {
	t.Parallel()

	result := testData["sessionInfo"]
	if result == "" {
		t.Fatal("could not find test data for session info")
	}
	resultBytes, err := common.HexToBytes(result)
	require.NoError(t, err)

	var sessionInfo *SessionInfo
	err = scale.Unmarshal(resultBytes, &sessionInfo)
	require.NoError(t, err)

	expected := &SessionInfo{
		ActiveValidatorIndices: []ValidatorIndex{7, 12, 14, 1, 4, 16, 3, 11, 9, 6, 13, 15, 5, 0, 8, 10, 2},
		RandomSeed: mustHexTo32BArray(t,
			"0x9a14667dcf973e46392904593e8caf2fb7a57904edbadf1547531657e7a56b5e"),
		DisputePeriod: 6,
		Validators: []ValidatorID{
			mustHexTo32BArray(t, "0xa262f83b46310770ae8d092147176b8b25e8855bcfbbe701d346b10db0c5385d"),
			mustHexTo32BArray(t, "0x804b9df571e2b744d65eca2d4c59eb8e4345286c00389d97bfc1d8d13aa6e57e"),
			mustHexTo32BArray(t, "0x4eb63e4aad805c06dc924e2f19b1dde7faf507e5bb3c1838d6a3cfc10e84fe72"),
			mustHexTo32BArray(t, "0x74c337d57035cd6b7718e92a0d8ea6ef710da8ab1215a057c40c4ef792155a68"),
			mustHexTo32BArray(t, "0xe61d138eebd2069f1a76b3570f9de6a4b196289b198e33e6f0b59cef8837c511"),
			mustHexTo32BArray(t, "0x94ef34321ca5d37a6e8953183406b76f8ebf6a4be5eefc3997d022ac6e0a050e"),
			mustHexTo32BArray(t, "0xac837e8ca589521a83e7d9a7b307d1c41a5d9b940422488236f99646d21f3841"),
			mustHexTo32BArray(t, "0xb61cb85f7cf7616f9ef8f95010a51a68a4eae8afcdff715cc6a8d43da4a32a12"),
			mustHexTo32BArray(t, "0x382f17dae6b13a8ce5a7cc805056d9b592d918c8593f077db28cb14cf08a760c"),
			mustHexTo32BArray(t, "0x0825ba7677597ec9453ab5dbaa9e68bf89dc36694cb6e74cbd5a9a74b167e547"),
			mustHexTo32BArray(t, "0xcee3f65d78a239d7d199b100295e7a2d852ae898a6b81fd867b3471f25be7237"),
			mustHexTo32BArray(t, "0xe2ac8f039eb02370a9577e49ffc6032e6b5bf5ff77783bdc676d1432d714fd53"),
			mustHexTo32BArray(t, "0xce35fa64fe7a5a6fc456ed2830e64d5d1a5dba26e7a57ab458f8cedf1ec77016"),
			mustHexTo32BArray(t, "0xae40e895f46c8bfb3df63c119047d7faf21c3fe3e7a91994a3f00da6fa80f848"),
			mustHexTo32BArray(t, "0xa0e038975cff34d01c62960828c23ec10a305fe9f5c3589c2ae40f51963e380a"),
			mustHexTo32BArray(t, "0x807fa54347a8957ff5ef6c28e2403c83947e5fad4aa805c914df0645a07aab5a"),
			mustHexTo32BArray(t, "0x4c8e878d7f558ce5086cc37ca0d5964bed54ddd6b15a6663a95fe42e36858936"),
		},
		DiscoveryKeys: []AuthorityDiscoveryID{
			mustHexTo32BArray(t, "0x407a89ac6943b9d2ef1ceb5f1299941758a6af5b8f79b89b90f95a3e38179341"),
			mustHexTo32BArray(t, "0x307744a128c608be0dff2189557715b74734359974606d96dc4d256d61b1047d"),
			mustHexTo32BArray(t, "0x74fff2667b4a2cc69198ec9d3bf41f4d001ab644b45feaf89a21ff7ef3bd2618"),
			mustHexTo32BArray(t, "0x98ab99b4b982d6a1d983ab05ac530b373043e6b7a4a7e5a7dc7ca1942196ae6c"),
			mustHexTo32BArray(t, "0x94f9e38609dd9972bfdbe4664f2063499f6233f895ee13b71793c926018a9428"),
			mustHexTo32BArray(t, "0x4ce0e8ec374f50c27948b8880628918a41b56930f1af675a5b5099d23f326763"),
			mustHexTo32BArray(t, "0x3a58b8f1f529e55fc3dac1dd81cb4547565c09f6e98d97243acb98bdda890028"),
			mustHexTo32BArray(t, "0x982bcec62ad60cf9fd00e89b7e3589adb668fcbc467127537851b5a5f3dbbb16"),
			mustHexTo32BArray(t, "0x0695b906f52a88f18bdecd811785b4299c51ebb2a2755f0b4c0d83fbef431861"),
			mustHexTo32BArray(t, "0x0ec5e1d2d044023009c63659c65a79aaf07ecbf5b9887958243aa873a63e5a1b"),
			mustHexTo32BArray(t, "0x52ef04ed449e4db577d98ad433b779c36f0d122df03e1cdc3e840a49016c5f16"),
			mustHexTo32BArray(t, "0xc2d4b5973000d0b175631dde5d1657b3e34c2f75e8a6d5414013ce4036d83355"),
			mustHexTo32BArray(t, "0xa6e01665b2d8490abf45551088021041dfb41772a9d596ed6e9f261ed1c8ae72"),
			mustHexTo32BArray(t, "0xb436c143e295617afb60353a01f2941bd33370a662c99c040984e52a072b5f22"),
			mustHexTo32BArray(t, "0x4c4c4b178f1a3d67e5f26d6b93b9a43937cd2d1d1cb2acc4650f504125df2e18"),
			mustHexTo32BArray(t, "0xca17f0edc319c140113a44722f829aa1313da1b54298a10df49ad7d67d9de85f"),
			mustHexTo32BArray(t, "0x5a6bf6911fc41d8981c7c28f87e8ed4416c65e15624f7b4e36c6a1a72c7a7819"),
		},
		AssignmentKeys: []AssignmentID{
			mustHexTo32BArray(t, "0x6acc35b896fe346adeda25c4031cf6a81e58dca091164370859828cc4456901a"),
			mustHexTo32BArray(t, "0x466627d554785807aaf50bfbdc9b8f729e8e20eb596ee5def5acd2acb72e405f"),
			mustHexTo32BArray(t, "0xc05cab9e7773ffaf045407579f9c8e16d56f119117421cd18a250c2e37fcb53a"),
			mustHexTo32BArray(t, "0xe2dca6ce9b3ebb40052c34392dc74d3cdd648399119fa470222a10956769d64f"),
			mustHexTo32BArray(t, "0x7477459916ace4f77d97d6ab5e1a2f06092282c7f0a1332628c14896e8e9be62"),
			mustHexTo32BArray(t, "0xc2574de3dc8feebfad1b3bee36a7bfe6c994e5d1459a5372ff447ac32dd46c11"),
			mustHexTo32BArray(t, "0xb0a8ed99f1e7ab160e0ac2fcfeee0d92d807c8fb4c1678e37997715578926c5c"),
			mustHexTo32BArray(t, "0x6c9bfa7c2e0f8e10a1a78bb982313c5c347a018cb3828886b99e109a8799d272"),
			mustHexTo32BArray(t, "0xe6037f1fc5b19015b7089ecf90034349e3f5c37cb50dec5356743614f94f8c33"),
			mustHexTo32BArray(t, "0x964b85f2b8e10e859e306d3670b8bdc0cea17b97dfd3edc8a9e1be1f127fee5b"),
			mustHexTo32BArray(t, "0x44d421ae62038ba15a377cad85e4ecd3c2a63b54fdbb82c47fb3e9c026405226"),
			mustHexTo32BArray(t, "0x48c51db949a58fd5f36a19888986275547b0c2fbb0b348ccb85dfc6c998dbe16"),
			mustHexTo32BArray(t, "0x0ae9425710301a9241837d624438a5d82edbbd6bf2cdbcc2694ad7db31ef9921"),
			mustHexTo32BArray(t, "0x9e47376e9af08b294901b879c7d658c41386453c6baa7c26560c5fd3b164e05d"),
			mustHexTo32BArray(t, "0x8af1a51649d44d12dffc24337f0a5424b18db9604133eafcb2639ddcdc2a7f0f"),
			mustHexTo32BArray(t, "0xae7a30d143fd125490434ca7325025a2338d0b8bb28dcd9373dfd83756191022"),
			mustHexTo32BArray(t, "0xeeba7c46f5fa1ea21e736d9ebd7a171fb2afe0a4f828a222ea0605a4ad0e6067"),
		},
		ValidatorGroups: [][]ValidatorIndex{
			{
				0, 1, 2, 3, 4, 5,
			},
			{
				6, 7, 8, 9, 10, 11,
			},
			{
				12, 13, 14, 15, 16,
			},
		},
		NCores:                  3,
		ZerothDelayTrancheWidth: 0,
		RelayVRFModuloSamples:   1,
		NDelayTranches:          40,
		NoShowSlots:             2,
		NeededApprovals:         2,
	}
	require.Equal(t, expected, sessionInfo)

	encoded, err := scale.Marshal(sessionInfo)
	require.NoError(t, err)
	require.Equal(t, result, common.BytesToHex(encoded))
}

// TestInboundHrmpMessage tests the scale encoding of an InboundHrmpMessage
func TestInboundHrmpMessage(t *testing.T) {
	t.Parallel()

	result := testData["hrmpChannelContents"]
	if result == "" {
		t.Fatal("hrmpChannelContents test data not found")
	}
	resultBytes, err := common.HexToBytes(result)
	require.NoError(t, err)

	var msg []InboundHrmpMessage
	err = scale.Unmarshal(resultBytes, &msg)
	require.NoError(t, err)

	expected := []InboundHrmpMessage{
		{
			SentAt: 1000,
			Data:   []byte{},
		},
		{
			SentAt: 2000,
			Data:   []byte{},
		},
		{
			SentAt: 2002,
			Data:   []byte{},
		},
		{
			SentAt: 2004,
			Data:   []byte{},
		},
		{
			SentAt: 2011,
			Data:   []byte{},
		},
		{
			SentAt: 2030,
			Data:   []byte{},
		},
		{
			SentAt: 2032,
			Data:   []byte{},
		},
		{
			SentAt: 2034,
			Data:   []byte{},
		},
		{
			SentAt: 2035,
			Data:   []byte{},
		},
		{
			SentAt: 2046,
			Data:   []byte{},
		},
	}
	require.Equal(t, expected, msg)

	encoded, err := scale.Marshal(msg)
	require.NoError(t, err)
	require.Equal(t, result, common.BytesToHex(encoded))
}

// TestCandidateEvent tests the scale encoding of a CandidateEvent
func TestCandidateEvent(t *testing.T) {
	t.Parallel()

	result := testData["candidateEvents"]
	if result == "" {
		t.Fatal("candidateEvents test data not found")
	}
	resultBytes, err := common.HexToBytes(result)
	require.NoError(t, err)

	candidateEvents, err := NewCandidateEvents()
	require.NoError(t, err)
	err = scale.Unmarshal(resultBytes, &candidateEvents)
	require.NoError(t, err)

	require.Greater(t, len(candidateEvents.Types), 0)

	encoded, err := scale.Marshal(candidateEvents)
	require.NoError(t, err)
	require.Equal(t, result, common.BytesToHex(encoded))
}

func mustHexTo32BArray(t *testing.T, inputHex string) (outputArray [32]byte) {
	t.Helper()
	copy(outputArray[:], common.MustHexToBytes(inputHex))
	return outputArray
}

func TestMustHexTo32BArray(t *testing.T) {
	inputHex := "0xa262f83b46310770ae8d092147176b8b25e8855bcfbbe701d346b10db0c5385d"
	expectedArray := [32]byte{0xa2, 0x62, 0xf8, 0x3b, 0x46, 0x31, 0x7, 0x70, 0xae, 0x8d, 0x9, 0x21, 0x47, 0x17, 0x6b,
		0x8b, 0x25, 0xe8, 0x85, 0x5b, 0xcf, 0xbb, 0xe7, 0x1, 0xd3, 0x46, 0xb1, 0xd, 0xb0, 0xc5, 0x38, 0x5d}
	result := mustHexTo32BArray(t, inputHex)
	require.Equal(t, expectedArray, result)
}

func TestPersistedValidationData(t *testing.T) {
	expected := []byte{12, 7, 8, 9, 10, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4, 0, 0} //nolint:lll

	pvd := PersistedValidationData{
		ParentHead:             HeadData{Data: []byte{7, 8, 9}},
		RelayParentNumber:      10,
		RelayParentStorageRoot: common.Hash{},
		MaxPovSize:             uint32(1024),
	}

	actual, err := scale.Marshal(pvd)
	require.NoError(t, err)
	require.Equal(t, expected, actual)

	newpvd := PersistedValidationData{}
	err = scale.Unmarshal(actual, &newpvd)
	require.NoError(t, err)
	require.Equal(t, pvd, newpvd)
}

func TestOccupiedCoreAssumption(t *testing.T) {

	for _, tc := range []struct {
		name string
		in   scale.VaryingDataTypeValue
		out  byte
	}{
		{
			name: "included",
			in:   IncludedOccupiedCoreAssumption{},
			out:  0,
		},
		{
			name: "timeout",
			in:   TimedOutOccupiedCoreAssumption{},
			out:  1,
		},
		{
			name: "free",
			in:   FreeOccupiedCoreAssumption{},
			out:  2,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			vdt := NewOccupiedCoreAssumption()
			err := vdt.Set(tc.in)
			require.NoError(t, err)
			res, err := scale.Marshal(vdt)
			require.NoError(t, err)
			require.Equal(t, []byte{tc.out}, res)

			vdt2 := NewOccupiedCoreAssumption()
			err = scale.Unmarshal([]byte{tc.out}, &vdt2)
			require.NoError(t, err)
			require.Equal(t, tc.in.Index(), uint(tc.out))
		})
	}
}
