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

package wasmer

import (
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/stretchr/testify/require"
)

func Test_ext_hashing_blake2_128_version_1(t *testing.T) {
	t.Skip()
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)
	mem := inst.inst.vm.Memory.Data()

	data := []byte("helloworld")
	ptr, err := inst.inst.ctx.Allocator.Allocate(uint32(len(data)))
	require.NoError(t, err)

	copy(mem[ptr:ptr+uint32(len(data))], data)

	testFunc, ok := inst.inst.vm.Exports["rtm_ext_hashing_blake2_128_version_1"]
	require.True(t, ok)

	out, err := testFunc(int32(ptr), int32(len(data)))
	require.NoError(t, err)

	outInt := out.ToI32()

	expected, err := common.Blake2b128(data)
	require.NoError(t, err)
	require.Equal(t, expected[:], mem[outInt:outInt+16])
}
