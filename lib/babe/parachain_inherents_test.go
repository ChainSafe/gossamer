// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package babe

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/assert"
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
			enumValue:     explicitValidDisputeStatementKind{},
			encodingValue: []byte{0x0},
		},
		{
			name:          "BackingSeconded",
			enumValue:     backingSeconded(common.Hash{}),
			encodingValue: []byte{0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}, //nolint:lll
		},
		{
			name:          "BackingValid",
			enumValue:     backingValid(common.Hash{}),
			encodingValue: []byte{0x2, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}, //nolint:lll

		},
		{
			name:          "ApprovalChecking",
			enumValue:     approvalChecking{},
			encodingValue: []byte{0x3},
		},
	}

	for _, c := range testCases {
		t.Run(c.name, func(t *testing.T) {
			vdsKind, err := scale.NewVaryingDataType(
				explicitValidDisputeStatementKind{}, backingSeconded{}, backingValid{}, approvalChecking{})
			require.NoError(t, err)

			err = vdsKind.Set(c.enumValue)
			require.NoError(t, err)

			bytes, err := scale.Marshal(vdsKind)
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
			name:          "explicitInvalidDisputeStatementKind",
			enumValue:     explicitInvalidDisputeStatementKind{},
			encodingValue: []byte{0x0},
		},
	}

	for _, c := range testCases {
		t.Run(c.name, func(t *testing.T) {
			invalidDisputeStatementKind, err := scale.NewVaryingDataType(
				explicitInvalidDisputeStatementKind{})
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
				vdsKind, err := scale.NewVaryingDataType(
					explicitValidDisputeStatementKind{}, backingSeconded{}, backingValid{}, approvalChecking{})
				require.NoError(t, err)

				err = vdsKind.Set(explicitValidDisputeStatementKind{})
				require.NoError(t, err)

				disputeStatement := newDisputeStatement()
				err = disputeStatement.Set(validDisputeStatementKind(vdsKind))
				require.NoError(t, err)

				return disputeStatement
			}(),

			encodingValue: []byte{0x0, 0x0},
		},
		{
			name: "Valid ApprovalChecking",
			vdt: func() DisputeStatement {
				vdsKind, err := scale.NewVaryingDataType(
					explicitValidDisputeStatementKind{}, backingSeconded{}, backingValid{}, approvalChecking{},
				)
				require.NoError(t, err)

				err = vdsKind.Set(approvalChecking{})
				require.NoError(t, err)

				disputeStatement := newDisputeStatement()
				err = disputeStatement.Set(validDisputeStatementKind(vdsKind))
				require.NoError(t, err)

				return disputeStatement
			}(),
			encodingValue: []byte{0x0, 0x3},
		},
		{
			name: "Valid BackingSeconded",
			vdt: func() DisputeStatement {
				vdsKind, err := scale.NewVaryingDataType(
					explicitValidDisputeStatementKind{}, backingSeconded{}, backingValid{}, approvalChecking{},
				)
				require.NoError(t, err)

				err = vdsKind.Set(backingSeconded(common.Hash{}))
				require.NoError(t, err)

				disputeStatement := newDisputeStatement()
				err = disputeStatement.Set(validDisputeStatementKind(vdsKind))
				require.NoError(t, err)

				return disputeStatement
			}(),
			encodingValue: []byte{0x0, 0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}, //nolint:lll

		},
		{
			name: "Invalid Explicit",
			vdt: func() DisputeStatement {
				idsKind, err := scale.NewVaryingDataType(
					explicitInvalidDisputeStatementKind{},
				)
				require.NoError(t, err)

				err = idsKind.Set(explicitInvalidDisputeStatementKind{})
				require.NoError(t, err)

				disputeStatement := newDisputeStatement()
				err = disputeStatement.Set(invalidDisputeStatementKind(idsKind))
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

			require.Equal(t, c.encodingValue, bytes)

			newDst := newDisputeStatement()
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
			enumValue:     implicit(validatorSignature{}),
			encodingValue: []byte{0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}, //nolint:lll
		},
		{
			name:          "Explicit",
			enumValue:     explicit(validatorSignature{}),
			encodingValue: []byte{0x2, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}, //nolint:lll
		},
	}

	for _, c := range testCases {
		t.Run(c.name, func(t *testing.T) {
			validityAttestation := newValidityAttestation()
			err := validityAttestation.Set(c.enumValue)
			require.NoError(t, err)

			bytes, err := scale.Marshal(validityAttestation)
			require.NoError(t, err)

			require.Equal(t, c.encodingValue, bytes)

			newDst := newValidityAttestation()
			err = scale.Unmarshal(bytes, &newDst)
			require.NoError(t, err)

			if !reflect.DeepEqual(validityAttestation, newDst) {
				panic(fmt.Errorf("uh oh: \n%+v \n\n%+v", validityAttestation, newDst))
			}
		})
	}

}

func TestParachainInherents(t *testing.T) {
	expectedParaInherentsbytes := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}                                            //nolint:lll
	expectedInherentsBytes := []byte{4, 112, 97, 114, 97, 99, 104, 110, 48, 149, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0} //nolint:lll
	// corresponding rust struct
	// ----------------------------------------
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
	// ----------------------------------------
	// way to get inherents encoding from rust
	// ----------------------------------------
	// let mut inherents: sp_inherents::InherentData = sp_inherents::InherentData::new();
	// inherents.put_data(*b"parachn0", &para_int).unwrap();
	// println!("{:?}", inherents.encode());

	parachainInherent := ParachainInherentData{
		Bitfields:        []uncheckedSignedAvailabilityBitfield{},
		BackedCandidates: []backedCandidate{},
		Disputes:         multiDisputeStatementSet{},
		ParentHeader: types.Header{
			ParentHash:     common.Hash{},
			Number:         0,
			StateRoot:      common.Hash{},
			ExtrinsicsRoot: common.Hash{},
			Digest:         scale.VaryingDataTypeSlice{},
		},
	}

	actualParaInherentBytes, err := scale.Marshal(parachainInherent)
	require.NoError(t, err)

	assert.Equal(t, expectedParaInherentsbytes, actualParaInherentBytes)

	idata := types.NewInherentsData()
	err = idata.SetStructInherent(types.Parachn0, parachainInherent)
	require.NoError(t, err)

	actualInherentsBytes, err := idata.Encode()
	require.NoError(t, err)
	require.Equal(t, expectedInherentsBytes, actualInherentsBytes)

}
