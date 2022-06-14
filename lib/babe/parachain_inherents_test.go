// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package babe

import (
	"testing"

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
			encodingValue: []uint8([]byte{0x0}),
		},
		{
			name:          "BackingSeconded",
			enumValue:     BackingSeconded(common.Hash{}),
			encodingValue: []uint8([]byte{0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}),
		},
		{
			name:          "BackingValid",
			enumValue:     BackingValid(common.Hash{}),
			encodingValue: []uint8([]byte{0x2, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}),
		},
		{
			name:          "ApprovalChecking",
			enumValue:     ApprovalChecking{},
			encodingValue: []uint8([]byte{0x3}),
		},
	}

	for _, c := range testCases {
		t.Run(c.name, func(t *testing.T) {
			validDisputeStatementKind, err := scale.NewVaryingDataType(
				ExplicitValidDisputeStatementKind{}, BackingSeconded{}, BackingValid{}, ApprovalChecking{})
			if err != nil {
				panic(err)
			}

			err = validDisputeStatementKind.Set(c.enumValue)
			if err != nil {
				panic(err)
			}

			bytes, err := scale.Marshal(validDisputeStatementKind)
			if err != nil {
				panic(err)
			}

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
			encodingValue: []uint8([]byte{0x0}),
		},
	}

	for _, c := range testCases {
		t.Run(c.name, func(t *testing.T) {
			invalidDisputeStatementKind, err := scale.NewVaryingDataType(
				ExplicitInvalidDisputeStatementKind{})
			if err != nil {
				panic(err)
			}

			err = invalidDisputeStatementKind.Set(c.enumValue)
			if err != nil {
				panic(err)
			}

			bytes, err := scale.Marshal(invalidDisputeStatementKind)
			if err != nil {
				panic(err)
			}

			require.Equal(t, c.encodingValue, bytes)
		})
	}
}

func TestDisputeStatement(t *testing.T) {

	testCases := []struct {
		name          string
		vdt           scale.VaryingDataType
		encodingValue []uint8
	}{
		{
			name: "Valid Explicit",
			vdt: func() scale.VaryingDataType {
				validDisputeStatementKind, err := scale.NewVaryingDataType(
					ExplicitValidDisputeStatementKind{}, BackingSeconded{}, BackingValid{}, ApprovalChecking{})
				if err != nil {
					panic(err)
				}
				err = validDisputeStatementKind.Set(ExplicitValidDisputeStatementKind{})
				if err != nil {
					panic(err)
				}
				disputeStatement, err := scale.NewVaryingDataType(Valid{}, Invalid{})
				if err != nil {
					panic(err)
				}

				err = disputeStatement.Set(Valid(validDisputeStatementKind))
				if err != nil {
					panic(err)
				}
				return disputeStatement
			}(),

			encodingValue: []uint8([]byte{0x0}),
		},
		{
			name: "Valid ApprovalChecking",
			vdt: func() scale.VaryingDataType {
				validDisputeStatementKind, err := scale.NewVaryingDataType(
					ExplicitValidDisputeStatementKind{}, BackingSeconded{}, BackingValid{}, ApprovalChecking{},
				)
				if err != nil {
					panic(err)
				}

				err = validDisputeStatementKind.Set(ApprovalChecking{})
				if err != nil {
					panic(err)
				}
				disputeStatement, err := scale.NewVaryingDataType(Valid{}, Invalid{})
				if err != nil {
					panic(err)
				}

				err = disputeStatement.Set(Valid(validDisputeStatementKind))
				if err != nil {
					panic(err)
				}

				return disputeStatement
			}(),
			encodingValue: []uint8([]byte{0x0}),
		},
	}

	for _, c := range testCases {
		t.Run(c.name, func(t *testing.T) {
			bytes, err := scale.Marshal(c.vdt)
			if err != nil {
				panic(err)
			}

			require.Equal(t, c.encodingValue, bytes)
		})
	}
}
