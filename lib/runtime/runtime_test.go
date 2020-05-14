// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package runtime

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExecVersion(t *testing.T) {
	// https://github.com/paritytech/substrate/blob/7b1d822446982013fa5b7ad5caff35ca84f8b7d0/core/test-runtime/src/lib.rs#L73
	expected := &Version{
		Spec_name:         []byte("test"),
		Impl_name:         []byte("parity-test"),
		Authoring_version: 1,
		Spec_version:      1,
		Impl_version:      1,
	}

	runtime := NewTestRuntime(t, POLKADOT_RUNTIME_c768a7e4c70e)

	ret, err := runtime.Exec(CoreVersion, []byte{})
	require.Nil(t, err)

	version := &VersionAPI{
		RuntimeVersion: &Version{},
		API:            nil,
	}
	version.Decode(ret)
	require.Nil(t, err)

	t.Logf("Spec_name: %s\n", version.RuntimeVersion.Spec_name)
	t.Logf("Impl_name: %s\n", version.RuntimeVersion.Impl_name)
	t.Logf("Authoring_version: %d\n", version.RuntimeVersion.Authoring_version)
	t.Logf("Spec_version: %d\n", version.RuntimeVersion.Spec_version)
	t.Logf("Impl_version: %d\n", version.RuntimeVersion.Impl_version)

	require.Equal(t, expected, version.RuntimeVersion)
}

// test used for ensuring runtime Exec calls can me made concurrently
func TestConcurrentRuntimeCalls(t *testing.T) {
	runtime := NewTestRuntime(t, TEST_RUNTIME)

	// Execute 2 concurrent calls to the runtime
	go func() {
		_, _ = runtime.Exec(CoreVersion, []byte{})
	}()
	go func() {
		_, _ = runtime.Exec(CoreVersion, []byte{})
	}()
}

func TestRuntime_Exec_Metadata(t *testing.T) {
	// expected results based on results from previous runs
	expected := []byte{16, 116, 101, 115, 116, 44, 112, 97, 114, 105, 116, 121, 45, 116, 101, 115, 116, 1, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 36, 223, 106, 203, 104, 153, 7, 96, 155, 2, 0, 0, 0, 55, 227, 151, 252, 124, 145, 245, 228, 1, 0, 0, 0, 210, 188, 152, 151, 238, 208, 143, 21, 1, 0, 0, 0, 64, 254, 58, 212, 1, 248, 149, 154, 3, 0, 0, 0, 198, 233, 167, 99, 9, 243, 155, 9, 1, 0, 0, 0, 221, 113, 141, 92, 197, 50, 98, 212, 1, 0, 0, 0, 203, 202, 37, 227, 159, 20, 35, 135, 1, 0, 0, 0, 247, 139, 39, 139, 229, 63, 69, 76, 1, 0, 0, 0, 171, 60, 5, 114, 41, 31, 235, 139, 1, 0, 0, 0}
	runtime := NewTestRuntime(t, POLKADOT_RUNTIME_c768a7e4c70e)

	ret, err := runtime.Exec(CoreVersion, []byte{})
	require.NoError(t, err)

	require.Equal(t, expected, ret)
}
