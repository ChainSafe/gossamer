// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSwapRemove(t *testing.T) {
	change1 := &PendingChange[string, uint]{
		canonHash: "a",
	}

	change2 := &PendingChange[string, uint]{
		canonHash: "b",
	}

	change3 := &PendingChange[string, uint]{
		canonHash: "b",
	}

	pendingChangeNode1 := &pendingChangeNode[string, uint]{
		change: change1,
	}

	pendingChangeNode2 := &pendingChangeNode[string, uint]{
		change: change2,
	}

	pendingChangeNode3 := &pendingChangeNode[string, uint]{
		change: change3,
	}

	changeNodes1 := []*pendingChangeNode[string, uint]{
		pendingChangeNode1,
		pendingChangeNode2,
	}

	changeNodes2 := []*pendingChangeNode[string, uint]{
		pendingChangeNode1,
		pendingChangeNode2,
		pendingChangeNode3,
	}
	type args struct {
		ct    ChangeTree[string, uint]
		index uint
	}
	tests := []struct {
		name string
		args args
		exp  pendingChangeNode[string, uint]
	}{
		{
			name: "TwoElemSliceDeletingLastElement",
			args: args{
				ct: ChangeTree[string, uint]{
					roots: changeNodes1,
				},
				index: 1,
			},
			exp: *pendingChangeNode2,
		},
		{
			name: "ThreeElemSliceDeletingFirstElement",
			args: args{
				ct: ChangeTree[string, uint]{
					roots: changeNodes2,
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
