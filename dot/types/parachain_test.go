// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package types

import (
	"testing"

	"github.com/ChainSafe/gossamer/pkg/scale"

	"github.com/stretchr/testify/require"

	"github.com/ChainSafe/gossamer/lib/common"
)

// Test_Validators tests the scale encoding and decoding of a Validator struct
func Test_Validators(t *testing.T) {
	t.Parallel()

	expected := []common.Address{
		"5Do5qUFfEp5CcAfdcYSj3ZyPzQcbgtTMYZozyD4VCMF4gzDb",
		"5CP9KEeF9V3eyTWEStEKQyi6grLsUgEZRqHVcTTm5PbL5CWF",
		"5DqujH7PVLVLE7czPhx1gNohVCPUQ5TfhqJ8wKLcZH3zjKd1",
		"5Fjd1oQRcLBfEjCVyqhDK2WTEhZ5srNU3LCVB8f52PLytLs8",
		"5FRz16eLxHdvCtPfZXpNW5Cn1k61tKTLVrNH36jmAeurMtzp",
		"5ExvT1sWmnt2DL5KJp7QnKXsMmx2AH5YMWmGvdkrGq2ycLHd",
		"5GjySVgQVLKiA3UG3bJrUHb2Q9PNvFuSBXpUb9vo5arg4jw8",
		"5Fxu9xhgqcxTtTQL1HJcLqQpYhj6B4a1Usn1LtYkc7qGry7K",
		"5G1BUGa4iCLLQGgypZeeUThiLk2S9Y1Yy7ddpNhPLrgYf6kj",
		"5HGRWRdLehaFbJ3zczdfgHhubCygs1bME8yLc4MWidqDwPkj",
		"5HBuwTWAzV1LEReH8ohsWRUeCx5ykb3CkCa2GNJKHozQ3hcK",
		"5Gj5kpJCbX5oQXRTv6insZeMtMCEptqJ2A5vB5gHsdBgwYJp",
		"5Fhe8H4XeYdvV31FVfxKAivKcDw9g5EMnm7oWCunnFfrxEip",
		"5CFPSw1J5S9eNyUoh1oceF3inj1zPsV8csiBzR63WZrrf6sq",
		"5GBV6ye2TkgNTn9ctjVx4dfxfRQYV1pcPwxrR9ARTTya7Bwc",
		"5DLNXbTdnVqoHK2rhskA27e9qxVQvk1GeY1rwxn5kzBf757w",
		"5EyBuSrUhLx5prt47u7R4djuhAaenABPTNVruDcVEK7K574Y",
	}
	resultHex := "0x444c8e878d7f558ce5086cc37ca0d5964bed54ddd6b15a6663a95fe42e368589360e101de266b8f5f05431dcaf63ecd936988cc348a271f42519bef19df1e9af7f4eb63e4aad805c06dc924e2f19b1dde7faf507e5bb3c1838d6a3cfc10e84fe72a262f83b46310770ae8d092147176b8b25e8855bcfbbe701d346b10db0c5385d94ef34321ca5d37a6e8953183406b76f8ebf6a4be5eefc3997d022ac6e0a050e804b9df571e2b744d65eca2d4c59eb8e4345286c00389d97bfc1d8d13aa6e57ecee3f65d78a239d7d199b100295e7a2d852ae898a6b81fd867b3471f25be7237ac837e8ca589521a83e7d9a7b307d1c41a5d9b940422488236f99646d21f3841ae40e895f46c8bfb3df63c119047d7faf21c3fe3e7a91994a3f00da6fa80f848e61d138eebd2069f1a76b3570f9de6a4b196289b198e33e6f0b59cef8837c511e2ac8f039eb02370a9577e49ffc6032e6b5bf5ff77783bdc676d1432d714fd53ce35fa64fe7a5a6fc456ed2830e64d5d1a5dba26e7a57ab458f8cedf1ec77016a0e038975cff34d01c62960828c23ec10a305fe9f5c3589c2ae40f51963e380a0825ba7677597ec9453ab5dbaa9e68bf89dc36694cb6e74cbd5a9a74b167e547b61cb85f7cf7616f9ef8f95010a51a68a4eae8afcdff715cc6a8d43da4a32a12382f17dae6b13a8ce5a7cc805056d9b592d918c8593f077db28cb14cf08a760c807fa54347a8957ff5ef6c28e2403c83947e5fad4aa805c914df0645a07aab5a"
	resultBytes, err := common.HexToBytes(resultHex)
	require.NoError(t, err)

	var validatorIDs []ValidatorID
	err = scale.Unmarshal(resultBytes, &validatorIDs)
	require.NoError(t, err)

	var validators []Validator
	validators, err = ValidatorIDToValidator(validatorIDs)
	require.NoError(t, err)

	var addresses []common.Address
	for _, v := range validators {
		addresses = append(addresses, v.Key.Address())
	}
	require.Equal(t, expected, addresses)

	encoded, err := scale.Marshal(validatorIDs)
	require.NoError(t, err)
	require.Equal(t, resultHex, common.BytesToHex(encoded))
}

// Test_ValidatorGroup tests the validator group encoding and decoding.
func Test_ValidatorGroup(t *testing.T) {
	t.Parallel()

	expected := ValidatorGroups{
		Validators: [][]ValidatorIndex{{0, 1, 2, 3, 4, 5}, {6, 7, 8, 9, 10, 11}, {12, 13, 14, 15, 16}},
		GroupRotationInfo: GroupRotationInfo{
			SessionStartBlock:      15657314,
			GroupRotationFrequency: 10,
			Now:                    15657556,
		},
	}

	result := "0x0c1800000000010000000200000003000000040000000500000018060000000700000008000000090000000a0000000b000000140c0000000d0000000e0000000f0000001000000062e9ee000a00000054eaee00"
	resultBytes, err := common.HexToBytes(result)
	require.NoError(t, err)

	validatorGroups := ValidatorGroups{
		Validators:        [][]ValidatorIndex{},
		GroupRotationInfo: GroupRotationInfo{},
	}
	err = scale.Unmarshal(resultBytes, &validatorGroups)
	require.NoError(t, err)

	require.Equal(t, expected, validatorGroups)

	encoded, err := scale.Marshal(validatorGroups)
	require.NoError(t, err)

	require.Equal(t, result, common.BytesToHex(encoded))
}

// Test_AvailabilityCoresScheduled tests the CoreState VDT with ScheduledCore encoding and decoding.
// TODO: cover it for other CoreState variants
func Test_AvailabilityCoresScheduled(t *testing.T) {
	t.Parallel()

	result := "0x0c01e80300000001e90300000001ea03000000"
	resultBytes, err := common.HexToBytes(result)
	require.NoError(t, err)

	availabilityCoreVDT, err := NewCoreStateVDT()
	require.NoError(t, err)
	availabilityCores := scale.NewVaryingDataTypeSlice(availabilityCoreVDT)
	err = scale.Unmarshal(resultBytes, &availabilityCores)
	require.NoError(t, err)

	vdt, err := NewCoreStateVDT()
	require.NoError(t, err)
	expected := scale.NewVaryingDataTypeSlice(vdt)
	err = expected.Add(
		ScheduledCore{
			ParaID:   1000,
			Collator: nil,
		},
		ScheduledCore{
			ParaID:   1001,
			Collator: nil,
		},
		ScheduledCore{
			ParaID:   1002,
			Collator: nil,
		},
	)
	require.NoError(t, err)
	require.Equal(t, expected, availabilityCores)

	encoded, err := scale.Marshal(availabilityCores)
	require.NoError(t, err)
	require.Equal(t, result, common.BytesToHex(encoded))
}

// TestSessionIndex tests the SessionIndex encoding and decoding.
func TestSessionIndex(t *testing.T) {
	t.Parallel()

	result := "0x0f690000"
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
