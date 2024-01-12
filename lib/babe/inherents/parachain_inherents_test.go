// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package inherents

import (
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidDisputeStatementKind(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		enumValue     any
		encodingValue []byte
	}{
		{
			name:          "ExplicitValidDisputeStatementKind",
			enumValue:     explicitValidDisputeStatementKind{},
			encodingValue: []byte{0x0},
		},
		{
			name:          "BackingSeconded",
			enumValue:     backingSeconded(common.Hash{1}),
			encodingValue: []byte{0x1, 0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}, //nolint:lll
		},
		{
			name:          "BackingValid",
			enumValue:     backingValid(common.Hash{1}),
			encodingValue: []byte{0x2, 0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}, //nolint:lll

		},
		{
			name:          "ApprovalChecking",
			enumValue:     approvalChecking{},
			encodingValue: []byte{0x3},
		},
	}

	for _, c := range testCases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			vdsKind := validDisputeStatementKind{}

			err := vdsKind.SetValue(c.enumValue)
			require.NoError(t, err)

			bytes, err := scale.Marshal(vdsKind)
			require.NoError(t, err)

			require.Equal(t, c.encodingValue, bytes)
		})
	}
}

func TestInvalidDisputeStatementKind(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		enumValue     any
		encodingValue []byte
	}{
		{
			name:          "explicitInvalidDisputeStatementKind",
			enumValue:     explicitInvalidDisputeStatementKind{},
			encodingValue: []byte{0x0},
		},
	}

	for _, c := range testCases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			invalidDisputeStatementKind := invalidDisputeStatementKind{}

			err := invalidDisputeStatementKind.SetValue(c.enumValue)
			require.NoError(t, err)

			bytes, err := scale.Marshal(invalidDisputeStatementKind)
			require.NoError(t, err)

			require.Equal(t, c.encodingValue, bytes)
		})
	}
}

func TestDisputeStatement(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		vdtBuilder    func(t *testing.T) disputeStatement
		encodingValue []byte
	}{
		{
			name: "Valid_Explicit",
			vdtBuilder: func(t *testing.T) disputeStatement {
				vdsKind := validDisputeStatementKind{}

				err := vdsKind.SetValue(explicitValidDisputeStatementKind{})
				require.NoError(t, err)

				ds := newDisputeStatement()
				err = ds.SetValue(validDisputeStatementKind(vdsKind))
				require.NoError(t, err)

				return ds
			},

			encodingValue: []byte{0x0, 0x0},
		},
		{
			name: "Valid_ApprovalChecking",
			vdtBuilder: func(t *testing.T) disputeStatement {
				vdsKind := validDisputeStatementKind{}

				err := vdsKind.SetValue(approvalChecking{})
				require.NoError(t, err)

				ds := newDisputeStatement()
				err = ds.SetValue(validDisputeStatementKind(vdsKind))
				require.NoError(t, err)

				return ds
			},
			encodingValue: []byte{0x0, 0x3},
		},
		{
			name: "Valid_BackingSeconded",
			vdtBuilder: func(t *testing.T) disputeStatement {
				vdsKind := validDisputeStatementKind{}

				err := vdsKind.SetValue(backingSeconded(common.Hash{}))
				require.NoError(t, err)

				ds := newDisputeStatement()
				err = ds.SetValue(validDisputeStatementKind(vdsKind))
				require.NoError(t, err)

				return ds
			},
			encodingValue: []byte{0x0, 0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}, //nolint:lll

		},
		{
			name: "Invalid_Explicit",
			vdtBuilder: func(t *testing.T) disputeStatement {
				idsKind := invalidDisputeStatementKind{}

				err := idsKind.SetValue(explicitInvalidDisputeStatementKind{})
				require.NoError(t, err)

				disputeStatement := newDisputeStatement()
				err = disputeStatement.SetValue(invalidDisputeStatementKind(idsKind))
				require.NoError(t, err)

				return disputeStatement
			},
			encodingValue: []byte{0x1, 0x0},
		},
	}

	for _, c := range testCases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			disputeStatement := c.vdtBuilder(t)

			bytes, err := scale.Marshal(disputeStatement)
			require.NoError(t, err)

			require.Equal(t, c.encodingValue, bytes)

			newDst := newDisputeStatement()
			err = scale.Unmarshal(bytes, &newDst)
			require.NoError(t, err)

			assert.Equal(t, disputeStatement, newDst)
		})
	}
}

func TestValidityAttestation(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		enumValue     any
		encodingValue []byte
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
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			validityAttestation := newValidityAttestation()
			err := validityAttestation.SetValue(c.enumValue)
			require.NoError(t, err)

			bytes, err := scale.Marshal(validityAttestation)
			require.NoError(t, err)

			require.Equal(t, c.encodingValue, bytes)

			newDst := newValidityAttestation()
			err = scale.Unmarshal(bytes, &newDst)
			require.NoError(t, err)

			assert.Equal(t, validityAttestation, newDst)
		})
	}
}

func TestParachainInherents(t *testing.T) {
	t.Parallel()

	expectedParaInherentsbytes := []byte{0, 0, 0, 197, 243, 254, 225, 31, 117, 21, 218, 179, 213, 92, 6, 247, 164, 230, 25, 47, 166, 140, 117, 142, 159, 195, 202, 67, 196, 238, 26, 44, 18, 33, 92, 65, 31, 219, 225, 47, 12, 107, 88, 153, 146, 55, 21, 226, 186, 110, 48, 167, 187, 67, 183, 228, 232, 118, 136, 30, 254, 11, 87, 48, 112, 7, 97, 31, 82, 146, 110, 96, 87, 152, 68, 98, 162, 227, 222, 78, 14, 244, 194, 120, 154, 112, 97, 222, 144, 174, 101, 220, 44, 111, 126, 54, 34, 155, 220, 253, 124, 0}                                            //nolint:lll
	expectedInherentsBytes := []byte{4, 112, 97, 114, 97, 99, 104, 110, 48, 153, 1, 0, 0, 0, 197, 243, 254, 225, 31, 117, 21, 218, 179, 213, 92, 6, 247, 164, 230, 25, 47, 166, 140, 117, 142, 159, 195, 202, 67, 196, 238, 26, 44, 18, 33, 92, 65, 31, 219, 225, 47, 12, 107, 88, 153, 146, 55, 21, 226, 186, 110, 48, 167, 187, 67, 183, 228, 232, 118, 136, 30, 254, 11, 87, 48, 112, 7, 97, 31, 82, 146, 110, 96, 87, 152, 68, 98, 162, 227, 222, 78, 14, 244, 194, 120, 154, 112, 97, 222, 144, 174, 101, 220, 44, 111, 126, 54, 34, 155, 220, 253, 124, 0} //nolint:lll

	// corresponding rust struct
	// ----------------------------------------
	// let para_int: polkadot_primitives::v2::InherentData = polkadot_primitives::v2::InherentData {
	// 	bitfields: Vec::new(),
	// 	backed_candidates: Vec::new(),
	// 	disputes: Vec::new(),
	// 	parent_header: polkadot_core_primitives::Header{
	// 	   parent_hash: BlakeTwo256::hash(b"1000"),
	// 	   digest: Default::default(),
	// 	   number: 2000,
	// 	   state_root: BlakeTwo256::hash(b"3000"),
	// 	   extrinsics_root: BlakeTwo256::hash(b"4000"),
	//    },
	// };
	// ----------------------------------------
	// way to get inherents encoding from rust
	// ----------------------------------------
	// let mut inherents: sp_inherents::InherentData = sp_inherents::InherentData::new();
	// inherents.put_data(*b"parachn0", &para_int).unwrap();
	// println!("{:?}", inherents.encode());

	parachainInherent := ParachainInherentData{
		ParentHeader: types.Header{
			ParentHash:     common.MustBlake2bHash([]byte("1000")),
			Number:         uint(2000),
			StateRoot:      common.MustBlake2bHash([]byte("3000")),
			ExtrinsicsRoot: common.MustBlake2bHash([]byte("4000")),
		},
	}

	actualParaInherentBytes, err := scale.Marshal(parachainInherent)
	require.NoError(t, err)

	assert.Equal(t, expectedParaInherentsbytes, actualParaInherentBytes)

	idata := types.NewInherentData()
	err = idata.SetInherent(types.Parachn0, parachainInherent)
	require.NoError(t, err)

	actualInherentsBytes, err := idata.Encode()
	require.NoError(t, err)
	require.Equal(t, expectedInherentsBytes, actualInherentsBytes)
}
