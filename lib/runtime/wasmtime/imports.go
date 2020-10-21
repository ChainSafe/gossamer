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

package wasmtime

import (
	"github.com/bytecodealliance/wasmtime-go"
)

func ext_logging_log_version_1(c *wasmtime.Caller, level int32, target, msg int64) {
}

func ext_sandbox_instance_teardown_version_1(c *wasmtime.Caller, a int32) {}
func ext_sandbox_instantiate_version_1(c *wasmtime.Caller, a int32, x, y int64, z int32) int32 {
	return 0
}
func ext_sandbox_invoke_version_1(c *wasmtime.Caller, a int32, x, y int64, z, d, e int32) int32 {
	return 0
}

func ImportsNodeRuntime(store *wasmtime.Store) []*wasmtime.Extern {
	lim := wasmtime.NewLimits(20, 30)
	mem := wasmtime.NewMemory(store, wasmtime.NewMemoryType(*lim))

	ext_logging_log_version_1 := wasmtime.WrapFunc(store, ext_logging_log_version_1)
	ext_sandbox_instance_teardown_version_1 := wasmtime.WrapFunc(store, ext_sandbox_instance_teardown_version_1)
	ext_sandbox_instantiate_version_1 := wasmtime.WrapFunc(store, ext_sandbox_instantiate_version_1)
	ext_sandbox_invoke_version_1 := wasmtime.WrapFunc(store, ext_sandbox_invoke_version_1)

	return []*wasmtime.Extern{
		mem.AsExtern(),
		ext_logging_log_version_1.AsExtern(),
		ext_sandbox_instance_teardown_version_1.AsExtern(),
		ext_sandbox_instantiate_version_1.AsExtern(),
		ext_sandbox_invoke_version_1.AsExtern(),
	}
}
