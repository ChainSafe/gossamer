// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachaintypes

import (
	"testing"

	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
)

var testCasesExecutorParam = []struct {
	name          string
	enumValue     scale.VaryingDataTypeValue
	encodingValue []byte
	expectedErr   error
}{
	{
		name:          "MaxMemoryPages",
		enumValue:     MaxMemoryPages(9),
		encodingValue: []byte{1, 9, 0, 0, 0},
		expectedErr:   nil,
	},
	{
		name:          "StackLogicalMax",
		enumValue:     StackLogicalMax(8),
		encodingValue: []byte{2, 8, 0, 0, 0},
		expectedErr:   nil,
	},
	{
		name:          "StackNativeMax",
		enumValue:     StackNativeMax(7),
		encodingValue: []byte{3, 7, 0, 0, 0},
		expectedErr:   nil,
	},
	{
		name:          "PrecheckingMaxMemory",
		enumValue:     PrecheckingMaxMemory(6),
		encodingValue: []byte{4, 6, 0, 0, 0, 0, 0, 0, 0},
		expectedErr:   nil,
	},
	{
		name: "PvfPrepTimeout",
		enumValue: PvfPrepTimeout{
			PvfPrepTimeoutKind: func() PvfPrepTimeoutKind {
				kind := NewPvfPrepTimeoutKind()
				if err := kind.Set(Lenient{}); err != nil {
					panic(err)
				}

				return kind
			}(),
			Millisec: 5,
		},
		encodingValue: []byte{5, 1, 5, 0, 0, 0, 0, 0, 0, 0},
		expectedErr:   nil,
	},
	{
		name: "PvfExecTimeout",
		enumValue: PvfExecTimeout{
			PvfExecTimeoutKind: func() PvfExecTimeoutKind {
				kind := NewPvfExecTimeoutKind()
				if err := kind.Set(Approval{}); err != nil {
					panic(err)
				}

				return kind
			}(),
			Millisec: 4,
		},
		encodingValue: []byte{6, 1, 4, 0, 0, 0, 0, 0, 0, 0},
		expectedErr:   nil,
	},
	{
		name:        "invalid_struct",
		enumValue:   invalidVayingDataTypeValue{},
		expectedErr: scale.ErrUnsupportedVaryingDataTypeValue,
	},
}

func TestExecutorParam(t *testing.T) {
	t.Parallel()

	for _, c := range testCasesExecutorParam {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			t.Run("marshal", func(t *testing.T) {
				t.Parallel()

				vdt := NewExecutorParam()
				err := vdt.Set(c.enumValue)

				if c.expectedErr != nil {
					require.ErrorContains(t, err, c.expectedErr.Error())
					return
				}

				require.NoError(t, err)
				bytes, err := scale.Marshal(vdt)
				require.NoError(t, err)

				require.Equal(t, c.encodingValue, bytes)
			})

			t.Run("unmarshal", func(t *testing.T) {
				t.Parallel()

				if c.expectedErr != nil {
					return
				}

				vdt := NewExecutorParam()
				err := scale.Unmarshal(c.encodingValue, &vdt)
				require.NoError(t, err)

				actualData, err := vdt.Value()
				require.NoError(t, err)

				require.EqualValues(t, c.enumValue, actualData)
			})
		})
	}
}

func TestExecutorParams(t *testing.T) {
	t.Parallel()

	for _, c := range testCasesExecutorParam {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			params := NewExecutorParams()
			err := params.Add(c.enumValue)

			if c.expectedErr == nil {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, c.expectedErr.Error())
			}
		})
	}
}
