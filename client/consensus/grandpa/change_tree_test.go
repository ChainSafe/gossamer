// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSwapRemove(t *testing.T) {
	change1 := &PendingChange[string, uint, uint]{
		CanonHash: "a",
	}

	change2 := &PendingChange[string, uint, uint]{
		CanonHash: "b",
	}

	change3 := &PendingChange[string, uint, uint]{
		CanonHash: "c",
	}

	pendingChangeNode1 := &PendingChangeNode[string, uint, uint]{
		Change: change1,
	}

	pendingChangeNode2 := &PendingChangeNode[string, uint, uint]{
		Change: change2,
	}

	pendingChangeNode3 := &PendingChangeNode[string, uint, uint]{
		Change: change3,
	}

	changeNodes1 := []*PendingChangeNode[string, uint, uint]{
		pendingChangeNode1,
		pendingChangeNode2,
	}

	changeNodes2 := []*PendingChangeNode[string, uint, uint]{
		pendingChangeNode1,
		pendingChangeNode2,
		pendingChangeNode3,
	}
	type args struct {
		ct    ChangeTree[string, uint, uint]
		index uint
	}
	tests := []struct {
		name string
		args args
		exp  PendingChangeNode[string, uint, uint]
	}{
		{
			name: "2 elem slice deleting last element",
			args: args{
				ct: ChangeTree[string, uint, uint]{
					TreeRoots: changeNodes1,
				},
				index: 1,
			},
			exp: *pendingChangeNode2,
		},
		{
			name: "3 elem slice deleting first element",
			args: args{
				ct: ChangeTree[string, uint, uint]{
					TreeRoots: changeNodes2,
				},
				index: 0,
			},
			exp: *pendingChangeNode1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldLength := len(tt.args.ct.Roots())
			removedVal := tt.args.ct.swapRemove(tt.args.ct.Roots(), tt.args.index)
			require.Equal(t, tt.exp, removedVal)
			require.Equal(t, oldLength-1, len(tt.args.ct.Roots()))
		})
	}
}
