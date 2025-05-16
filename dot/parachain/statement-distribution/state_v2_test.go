package statementdistribution

import (
	"testing"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestPeerStateUpdateView(t *testing.T) {
	ps := &peerState{
		view: parachaintypes.View{},
		implicitView: map[common.Hash]struct{}{
			{0x05}: {},
		},
	}

	newView := parachaintypes.View{
		Heads: []common.Hash{
			{0x05},
			{0x06},
		},
	}

	ctrl := gomock.NewController(t)
	localImplicitView := NewMockImplicitView(ctrl)
	localImplicitView.EXPECT().
		KnownAllowedRelayParentsUnder(common.Hash{0x05}, nil).
		Return([]common.Hash{{0x05}})
	localImplicitView.EXPECT().
		KnownAllowedRelayParentsUnder(common.Hash{0x06}, nil).
		Return([]common.Hash{{0x05}, {0x06}})

	fresh := ps.updateView(newView, localImplicitView)
	require.Equal(t, []common.Hash{{0x06}}, fresh)
	require.Equal(t, newView, ps.view)
	require.Equal(t, map[common.Hash]struct{}{
		{0x05}: {},
		{0x06}: {},
	}, ps.implicitView)
}

func TestPeerStateReconcileActiveLeaf(t *testing.T) {
	ps := &peerState{
		view: parachaintypes.View{
			Heads: []common.Hash{
				{0x05},
				{0x06},
			},
		},
		implicitView: map[common.Hash]struct{}{
			{0x05}: {},
		},
	}

	out := ps.reconcileActiveLeaf(common.Hash{0x06}, []common.Hash{{0x05}, {0x06}})
	require.Equal(t, []common.Hash{{0x06}}, out)
	require.Equal(t, map[common.Hash]struct{}{
		{0x05}: {},
		{0x06}: {},
	}, ps.implicitView)
}
