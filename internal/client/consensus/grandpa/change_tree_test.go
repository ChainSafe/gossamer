// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

// func TestSwapRemove(t *testing.T) {
// 	change1 := &PendingChange[string, uint, dummyAuthID]{
// 		CanonHash: "a",
// 	}

// 	change2 := &PendingChange[string, uint, dummyAuthID]{
// 		CanonHash: "b",
// 	}

// 	change3 := &PendingChange[string, uint, dummyAuthID]{
// 		CanonHash: "c",
// 	}

// 	pendingChangeNode1 := &PendingChangeNode[string, uint, dummyAuthID]{
// 		Change: change1,
// 	}

// 	pendingChangeNode2 := &PendingChangeNode[string, uint, dummyAuthID]{
// 		Change: change2,
// 	}

// 	pendingChangeNode3 := &PendingChangeNode[string, uint, dummyAuthID]{
// 		Change: change3,
// 	}

// 	changeNodes1 := []*PendingChangeNode[string, uint, dummyAuthID]{
// 		pendingChangeNode1,
// 		pendingChangeNode2,
// 	}

// 	changeNodes2 := []*PendingChangeNode[string, uint, dummyAuthID]{
// 		pendingChangeNode1,
// 		pendingChangeNode2,
// 		pendingChangeNode3,
// 	}
// 	type args struct {
// 		ct    ChangeTree[string, uint, dummyAuthID]
// 		index uint
// 	}
// 	tests := []struct {
// 		name string
// 		args args
// 		exp  PendingChangeNode[string, uint, dummyAuthID]
// 	}{
// 		{
// 			name: "2ElemSliceDeletingLastElement",
// 			args: args{
// 				ct: ChangeTree[string, uint, dummyAuthID]{
// 					TreeRoots: changeNodes1,
// 				},
// 				index: 1,
// 			},
// 			exp: *pendingChangeNode2,
// 		},
// 		{
// 			name: "3ElemSliceDeletingFirstElement",
// 			args: args{
// 				ct: ChangeTree[string, uint, dummyAuthID]{
// 					TreeRoots: changeNodes2,
// 				},
// 				index: 0,
// 			},
// 			exp: *pendingChangeNode1,
// 		},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			oldLength := len(tt.args.ct.Roots())
// 			removedVal := tt.args.ct.swapRemove(tt.args.ct.Roots(), tt.args.index)
// 			require.Equal(t, tt.exp, removedVal)
// 			require.Equal(t, oldLength-1, len(tt.args.ct.Roots()))
// 		})
// 	}
// }
