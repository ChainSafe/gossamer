package babe

import (
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/scale"

	"github.com/stretchr/testify/require"
)

func TestInherentExtrinsics(t *testing.T) {
	rt := runtime.NewTestRuntime(t, runtime.NODE_RUNTIME)

	idata := NewInherentsData()
	err := idata.SetInt64Inherent(Timstap0, uint64(time.Now().Unix()))
	require.NoError(t, err)

	tmp := [8]byte{}
	copy(tmp[:], Timstap0)
	t.Log(idata.data[tmp])

	// err = idata.SetInt64Inherent(Babeslot, 1)
	// require.NoError(t, err)

	ienc, err := idata.Encode()
	require.NoError(t, err)

	t.Log(ienc)
	t.Log(len(ienc))

	ie, err := rt.InherentExtrinsics(ienc)
	require.NoError(t, err)

	t.Log(ie)

	exts := make([][]byte, 2)
	inherents, err := scale.Decode(ie, exts)
	require.NoError(t, err)

	t.Log(inherents)

	for _, in := range inherents.([][]byte) {
		t.Log(in)
		ret, err := rt.ApplyExtrinsic(in)
		require.NoError(t, err)

		t.Log(ret)
	}
}
