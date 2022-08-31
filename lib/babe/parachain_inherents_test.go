// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package babe

import (
	"bytes"
	"fmt"
	reflect "reflect"
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
)

func TestValidDisputeStatementKind(t *testing.T) {

	testCases := []struct {
		name          string
		enumValue     scale.VaryingDataTypeValue
		encodingValue []uint8
	}{
		{
			name:          "ExplicitValidDisputeStatementKind",
			enumValue:     ExplicitValidDisputeStatementKind{},
			encodingValue: []byte{0x0},
		},
		{
			name:          "BackingSeconded",
			enumValue:     BackingSeconded(common.Hash{}),
			encodingValue: []byte{0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}, //nolint:lll
		},
		{
			name:          "BackingValid",
			enumValue:     BackingValid(common.Hash{}),
			encodingValue: []byte{0x2, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}, //nolint:lll

		},
		{
			name:          "ApprovalChecking",
			enumValue:     ApprovalChecking{},
			encodingValue: []byte{0x3},
		},
	}

	for _, c := range testCases {
		t.Run(c.name, func(t *testing.T) {
			validDisputeStatementKind, err := scale.NewVaryingDataType(
				ExplicitValidDisputeStatementKind{}, BackingSeconded{}, BackingValid{}, ApprovalChecking{})
			require.NoError(t, err)

			err = validDisputeStatementKind.Set(c.enumValue)
			require.NoError(t, err)

			bytes, err := scale.Marshal(validDisputeStatementKind)
			require.NoError(t, err)

			require.Equal(t, c.encodingValue, bytes)
		})
	}
}

func TestInvalidDisputeStatementKind(t *testing.T) {

	testCases := []struct {
		name          string
		enumValue     scale.VaryingDataTypeValue
		encodingValue []uint8
	}{
		{
			name:          "ExplicitInvalidDisputeStatementKind",
			enumValue:     ExplicitInvalidDisputeStatementKind{},
			encodingValue: []byte{0x0},
		},
	}

	for _, c := range testCases {
		t.Run(c.name, func(t *testing.T) {
			invalidDisputeStatementKind, err := scale.NewVaryingDataType(
				ExplicitInvalidDisputeStatementKind{})
			require.NoError(t, err)

			err = invalidDisputeStatementKind.Set(c.enumValue)
			require.NoError(t, err)

			bytes, err := scale.Marshal(invalidDisputeStatementKind)
			require.NoError(t, err)

			require.Equal(t, c.encodingValue, bytes)
		})
	}
}

func TestDisputeStatement(t *testing.T) {

	testCases := []struct {
		name          string
		vdt           DisputeStatement
		encodingValue []uint8
	}{
		{
			name: "Valid Explicit",
			vdt: func() DisputeStatement {
				validDisputeStatementKind, err := scale.NewVaryingDataType(
					ExplicitValidDisputeStatementKind{}, BackingSeconded{}, BackingValid{}, ApprovalChecking{})
				require.NoError(t, err)

				err = validDisputeStatementKind.Set(ExplicitValidDisputeStatementKind{})
				require.NoError(t, err)

				disputeStatement := NewDisputeStatement()
				err = disputeStatement.Set(ValidDisputeStatementKind(validDisputeStatementKind))
				require.NoError(t, err)

				return disputeStatement
			}(),

			encodingValue: []byte{0x0, 0x0},
		},
		{
			name: "Valid ApprovalChecking",
			vdt: func() DisputeStatement {
				validDisputeStatementKind, err := scale.NewVaryingDataType(
					ExplicitValidDisputeStatementKind{}, BackingSeconded{}, BackingValid{}, ApprovalChecking{},
				)
				require.NoError(t, err)

				err = validDisputeStatementKind.Set(ApprovalChecking{})
				require.NoError(t, err)

				disputeStatement := NewDisputeStatement()
				err = disputeStatement.Set(ValidDisputeStatementKind(validDisputeStatementKind))
				require.NoError(t, err)

				return disputeStatement
			}(),
			encodingValue: []byte{0x0, 0x3},
		},
		{
			name: "Valid BackingSeconded",
			vdt: func() DisputeStatement {
				validDisputeStatementKind, err := scale.NewVaryingDataType(
					ExplicitValidDisputeStatementKind{}, BackingSeconded{}, BackingValid{}, ApprovalChecking{},
				)
				require.NoError(t, err)

				err = validDisputeStatementKind.Set(BackingSeconded(common.Hash{}))
				require.NoError(t, err)

				disputeStatement := NewDisputeStatement()
				err = disputeStatement.Set(ValidDisputeStatementKind(validDisputeStatementKind))
				require.NoError(t, err)

				return disputeStatement
			}(),
			encodingValue: []byte{0x0, 0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}, //nolint:lll

		},
		{
			name: "Invalid Explicit",
			vdt: func() DisputeStatement {
				invalidDisputeStatementKind, err := scale.NewVaryingDataType(
					ExplicitInvalidDisputeStatementKind{},
				)
				require.NoError(t, err)

				err = invalidDisputeStatementKind.Set(ExplicitInvalidDisputeStatementKind{})
				require.NoError(t, err)

				disputeStatement := NewDisputeStatement()
				err = disputeStatement.Set(InvalidDisputeStatementKind(invalidDisputeStatementKind))
				require.NoError(t, err)

				return disputeStatement
			}(),
			encodingValue: []byte{0x1, 0x0},
		},
	}

	for _, c := range testCases {
		t.Run(c.name, func(t *testing.T) {
			bytes, err := scale.Marshal(c.vdt)
			require.NoError(t, err)

			fmt.Printf("0x%x\n", c.encodingValue)
			require.Equal(t, c.encodingValue, bytes)

			newDst := NewDisputeStatement()
			err = scale.Unmarshal(bytes, &newDst)
			require.NoError(t, err)

			if !reflect.DeepEqual(c.vdt, newDst) {
				panic(fmt.Errorf("uh oh: \n%+v \n\n%+v", c.vdt, newDst))
			}
		})
	}
}

func TestValidityAttestation(t *testing.T) {
	testCases := []struct {
		name          string
		enumValue     scale.VaryingDataTypeValue
		encodingValue []uint8
	}{
		{
			name:          "Implicit",
			enumValue:     Implicit(ValidatorSignature{}),
			encodingValue: []byte{0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}, //nolint:lll
		},
		{
			name:          "Explicit",
			enumValue:     Explicit(ValidatorSignature{}),
			encodingValue: []byte{0x2, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}, //nolint:lll
		},
	}

	for _, c := range testCases {
		t.Run(c.name, func(t *testing.T) {
			validityAttestation := NewValidityAttestation()
			err := validityAttestation.Set(c.enumValue)
			require.NoError(t, err)

			bytes, err := scale.Marshal(validityAttestation)
			require.NoError(t, err)

			fmt.Printf("bytes 0x%x\n", bytes)
			require.Equal(t, c.encodingValue, bytes)

			newDst := NewValidityAttestation()
			err = scale.Unmarshal(bytes, &newDst)
			require.NoError(t, err)

			if !reflect.DeepEqual(validityAttestation, newDst) {
				panic(fmt.Errorf("uh oh: \n%+v \n\n%+v", validityAttestation, newDst))
			}
		})
	}

}

func TestParachainInherents(t *testing.T) {
	expectedbytes := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

	// corresponding rust struct
	//
	// let para_int: polkadot_primitives::v2::InherentData = polkadot_primitives::v2::InherentData {
	// 	bitfields: Vec::new(),
	// 	backed_candidates: Vec::new(),
	// 	disputes: Vec::new(),
	// 	parent_header: polkadot_core_primitives::Header {
	// 	   number: 0,
	// 	   digest: Default::default(),
	// 	   extrinsics_root: Default::default(),
	// 	   parent_hash: Default::default(),
	// 	   state_root : Default::default(),
	//    },
	// };

	parachainInherent := ParachainInherentData{
		Bitfields:        []UncheckedSignedAvailabilityBitfield{},
		BackedCandidates: []BackedCandidate{},
		Disputes:         MultiDisputeStatementSet{},
		ParentHeader: types.Header{
			ParentHash:     common.Hash{},
			Number:         0,
			StateRoot:      common.Hash{},
			ExtrinsicsRoot: common.Hash{},
			Digest:         scale.VaryingDataTypeSlice{},
		},
	}

	encoded, err := scale.Marshal(parachainInherent)
	require.NoError(t, err)

	require.Equal(t, len(expectedbytes), len(encoded))
	require.True(t, bytes.Equal(encoded, expectedbytes))
}
