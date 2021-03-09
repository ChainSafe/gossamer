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
	"github.com/ChainSafe/gossamer/lib/runtime"
	wasm "github.com/wasmerio/wasmer-go/wasmer"
)

// ImportsNodeRuntime returns the imported objects needed for v0.8 of the runtime API
func ImportsNodeRuntime(store *wasm.Store, memory *wasm.Memory, ctx *runtime.Context) *wasm.ImportObject {
	importsMap := make(map[string]wasm.IntoExtern)

	importsMap["memory"] = memory
	importsMap["ext_allocator_malloc_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I32),
		wasm.NewValueTypes(wasm.I32),
	), ctx, ext_allocator_malloc_version_1)

	importsMap["ext_logging_log_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I32, wasm.I64, wasm.I64),
		wasm.NewValueTypes(),
	), ctx, ext_logging_log_version_1)

	importsMap["ext_crypto_ed25519_generate_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I32, wasm.I64),
		wasm.NewValueTypes(wasm.I32),
	), ctx, ext_crypto_ed25519_generate_version_1)
	importsMap["ext_crypto_ed25519_public_keys_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I32),
		wasm.NewValueTypes(wasm.I64),
	), ctx, ext_crypto_ed25519_public_keys_version_1)
	importsMap["ext_crypto_ed25519_sign_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I32, wasm.I32, wasm.I64),
		wasm.NewValueTypes(wasm.I64),
	), ctx, ext_crypto_ed25519_sign_version_1)

	imports := wasm.NewImportObject()
	imports.Register("env", importsMap)
	return imports
}
