// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSwapRemove(t *testing.T) {
	change1 := &PendingChange[common.Hash, uint]{
		canonHash: common.Hash{1},
	}

	change2 := &PendingChange[common.Hash, uint]{
		canonHash: common.Hash{2},
	}

	change3 := &PendingChange[common.Hash, uint]{
		canonHash: common.Hash{2},
	}

	pendingChangeNode1 := &pendingChangeNode[common.Hash, uint]{
		change: change1,
	}

	pendingChangeNode2 := &pendingChangeNode[common.Hash, uint]{
		change: change2,
	}

	pendingChangeNode3 := &pendingChangeNode[common.Hash, uint]{
		change: change3,
	}

	changeNodes1 := []*pendingChangeNode[common.Hash, uint]{
		pendingChangeNode1,
		pendingChangeNode2,
	}

	changeNodes2 := []*pendingChangeNode[common.Hash, uint]{
		pendingChangeNode1,
		pendingChangeNode2,
		pendingChangeNode3,
	}
	type args struct {
		ct    ChangeTree[common.Hash, uint]
		index uint
	}
	tests := []struct {
		name string
		args args
		exp  pendingChangeNode[common.Hash, uint]
	}{
		{
			name: "2 elem slice deleting last element",
			args: args{
				ct: ChangeTree[common.Hash, uint]{
					roots: changeNodes1,
				},
				index: 1,
			},
			exp: *pendingChangeNode2,
		},
		{
			name: "3 elem slice deleting first element",
			args: args{
				ct: ChangeTree[common.Hash, uint]{
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
