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
	wasm "github.com/wasmerio/wasmer-go/wasmer"

	"github.com/ChainSafe/gossamer/lib/runtime"
)

// ImportsNodeRuntime returns the imported objects needed for v0.8 of the runtime API
func ImportsNodeRuntime(store *wasm.Store, memory *wasm.Memory, ctx *runtime.Context) *wasm.ImportObject {
	importsMap := make(map[string]wasm.IntoExtern)

	if memory != nil {
		importsMap["memory"] = memory
	}

	importsMap["ext_logging_log_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I32, wasm.I64, wasm.I64),
		wasm.NewValueTypes(),
	), ctx, ext_logging_log_version_1)

	importsMap["ext_sandbox_instance_teardown_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I32),
		wasm.NewValueTypes(),
	), ctx, ext_sandbox_instance_teardown_version_1)
	importsMap["ext_sandbox_instantiate_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I32, wasm.I64, wasm.I64, wasm.I32),
		wasm.NewValueTypes(wasm.I32),
	), ctx, ext_sandbox_instantiate_version_1)
	importsMap["ext_sandbox_invoke_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I32, wasm.I64, wasm.I64, wasm.I32, wasm.I32, wasm.I32),
		wasm.NewValueTypes(wasm.I32),
	), ctx, ext_sandbox_invoke_version_1)
	importsMap["ext_sandbox_memory_get_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I32, wasm.I32, wasm.I32, wasm.I32),
		wasm.NewValueTypes(wasm.I32),
	), ctx, ext_sandbox_memory_get_version_1)
	importsMap["ext_sandbox_memory_new_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I32, wasm.I32),
		wasm.NewValueTypes(wasm.I32),
	), ctx, ext_sandbox_memory_new_version_1)
	importsMap["ext_sandbox_memory_set_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I32, wasm.I32, wasm.I32, wasm.I32),
		wasm.NewValueTypes(wasm.I32),
	), ctx, ext_sandbox_memory_set_version_1)
	importsMap["ext_sandbox_memory_teardown_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I32),
		wasm.NewValueTypes(),
	), ctx, ext_sandbox_memory_teardown_version_1)

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
	importsMap["ext_crypto_ed25519_verify_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I32, wasm.I64, wasm.I32),
		wasm.NewValueTypes(wasm.I32),
	), ctx, ext_crypto_ed25519_verify_version_1)
	importsMap["ext_crypto_secp256k1_ecdsa_recover_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I32, wasm.I32),
		wasm.NewValueTypes(wasm.I64),
	), ctx, ext_crypto_secp256k1_ecdsa_recover_version_1)
	importsMap["ext_crypto_secp256k1_ecdsa_recover_compressed_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I32, wasm.I32),
		wasm.NewValueTypes(wasm.I64),
	), ctx, ext_crypto_secp256k1_ecdsa_recover_compressed_version_1)
	importsMap["ext_crypto_sr25519_generate_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I32, wasm.I64),
		wasm.NewValueTypes(wasm.I32),
	), ctx, ext_crypto_sr25519_generate_version_1)
	importsMap["ext_crypto_sr25519_public_keys_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I32),
		wasm.NewValueTypes(wasm.I64),
	), ctx, ext_crypto_sr25519_public_keys_version_1)
	importsMap["ext_crypto_sr25519_sign_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I32, wasm.I32, wasm.I64),
		wasm.NewValueTypes(wasm.I64),
	), ctx, ext_crypto_sr25519_sign_version_1)
	importsMap["ext_crypto_sr25519_verify_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I32, wasm.I64, wasm.I32),
		wasm.NewValueTypes(wasm.I32),
	), ctx, ext_crypto_sr25519_verify_version_1)
	importsMap["ext_crypto_sr25519_verify_version_2"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I32, wasm.I64, wasm.I32),
		wasm.NewValueTypes(wasm.I32),
	), ctx, ext_crypto_sr25519_verify_version_2)
	importsMap["ext_crypto_start_batch_verify_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(),
		wasm.NewValueTypes(),
	), ctx, ext_crypto_start_batch_verify_version_1)
	importsMap["ext_crypto_finish_batch_verify_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(),
		wasm.NewValueTypes(wasm.I32),
	), ctx, ext_crypto_finish_batch_verify_version_1)

	importsMap["ext_trie_blake2_256_root_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I64),
		wasm.NewValueTypes(wasm.I32),
	), ctx, ext_trie_blake2_256_root_version_1)
	importsMap["ext_trie_blake2_256_ordered_root_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I64),
		wasm.NewValueTypes(wasm.I32),
	), ctx, ext_trie_blake2_256_ordered_root_version_1)

	importsMap["ext_misc_print_hex_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I64),
		wasm.NewValueTypes(),
	), ctx, ext_misc_print_hex_version_1)
	importsMap["ext_misc_print_num_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I64),
		wasm.NewValueTypes(),
	), ctx, ext_misc_print_num_version_1)
	importsMap["ext_misc_print_utf8_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I64),
		wasm.NewValueTypes(),
	), ctx, ext_misc_print_utf8_version_1)
	importsMap["ext_misc_runtime_version_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I64),
		wasm.NewValueTypes(wasm.I64),
	), ctx, ext_misc_runtime_version_version_1)

	importsMap["ext_default_child_storage_read_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I64, wasm.I64, wasm.I64, wasm.I32),
		wasm.NewValueTypes(wasm.I64),
	), ctx, ext_default_child_storage_read_version_1)
	importsMap["ext_default_child_storage_clear_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I64, wasm.I64),
		wasm.NewValueTypes(),
	), ctx, ext_default_child_storage_clear_version_1)
	importsMap["ext_default_child_storage_clear_prefix_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I64, wasm.I64),
		wasm.NewValueTypes(),
	), ctx, ext_default_child_storage_clear_prefix_version_1)
	importsMap["ext_default_child_storage_exists_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I64, wasm.I64),
		wasm.NewValueTypes(wasm.I32),
	), ctx, ext_default_child_storage_exists_version_1)
	importsMap["ext_default_child_storage_get_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I64, wasm.I64),
		wasm.NewValueTypes(wasm.I64),
	), ctx, ext_default_child_storage_get_version_1)
	importsMap["ext_default_child_storage_next_key_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I64, wasm.I64),
		wasm.NewValueTypes(wasm.I64),
	), ctx, ext_default_child_storage_next_key_version_1)
	importsMap["ext_default_child_storage_root_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I64),
		wasm.NewValueTypes(wasm.I64),
	), ctx, ext_default_child_storage_root_version_1)
	importsMap["ext_default_child_storage_set_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I64, wasm.I64, wasm.I64),
		wasm.NewValueTypes(),
	), ctx, ext_default_child_storage_set_version_1)
	importsMap["ext_default_child_storage_storage_kill_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I64),
		wasm.NewValueTypes(),
	), ctx, ext_default_child_storage_storage_kill_version_1)

	importsMap["ext_allocator_free_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I32),
		wasm.NewValueTypes(),
	), ctx, ext_allocator_free_version_1)
	importsMap["ext_allocator_malloc_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I32),
		wasm.NewValueTypes(wasm.I32),
	), ctx, ext_allocator_malloc_version_1)

	importsMap["ext_hashing_blake2_128_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I64),
		wasm.NewValueTypes(wasm.I32),
	), ctx, ext_hashing_blake2_128_version_1)
	importsMap["ext_hashing_blake2_256_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I64),
		wasm.NewValueTypes(wasm.I32),
	), ctx, ext_hashing_blake2_256_version_1)
	importsMap["ext_hashing_keccak_256_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I64),
		wasm.NewValueTypes(wasm.I32),
	), ctx, ext_hashing_keccak_256_version_1)
	importsMap["ext_hashing_sha2_256_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I64),
		wasm.NewValueTypes(wasm.I32),
	), ctx, ext_hashing_sha2_256_version_1)
	importsMap["ext_hashing_twox_256_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I64),
		wasm.NewValueTypes(wasm.I32),
	), ctx, ext_hashing_twox_256_version_1)
	importsMap["ext_hashing_twox_128_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I64),
		wasm.NewValueTypes(wasm.I32),
	), ctx, ext_hashing_twox_128_version_1)
	importsMap["ext_hashing_twox_64_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I64),
		wasm.NewValueTypes(wasm.I32),
	), ctx, ext_hashing_twox_64_version_1)

	importsMap["ext_offchain_index_set_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I64, wasm.I64),
		wasm.NewValueTypes(),
	), ctx, ext_offchain_index_set_version_1)
	importsMap["ext_offchain_is_validator_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(),
		wasm.NewValueTypes(wasm.I32),
	), ctx, ext_offchain_is_validator_version_1)
	importsMap["ext_offchain_local_storage_compare_and_set_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I32, wasm.I64, wasm.I64, wasm.I64),
		wasm.NewValueTypes(wasm.I32),
	), ctx, ext_offchain_local_storage_compare_and_set_version_1)
	importsMap["ext_offchain_local_storage_get_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I32, wasm.I64),
		wasm.NewValueTypes(wasm.I64),
	), ctx, ext_offchain_local_storage_get_version_1)
	importsMap["ext_offchain_local_storage_set_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I32, wasm.I64, wasm.I64),
		wasm.NewValueTypes(),
	), ctx, ext_offchain_local_storage_set_version_1)
	importsMap["ext_offchain_network_state_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(),
		wasm.NewValueTypes(wasm.I64),
	), ctx, ext_offchain_network_state_version_1)
	importsMap["ext_offchain_random_seed_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(),
		wasm.NewValueTypes(wasm.I32),
	), ctx, ext_offchain_random_seed_version_1)
	importsMap["ext_offchain_submit_transaction_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I64),
		wasm.NewValueTypes(wasm.I64),
	), ctx, ext_offchain_submit_transaction_version_1)

	importsMap["ext_storage_append_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I64, wasm.I64),
		wasm.NewValueTypes(),
	), ctx, ext_storage_append_version_1)
	importsMap["ext_storage_changes_root_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I64),
		wasm.NewValueTypes(wasm.I64),
	), ctx, ext_storage_changes_root_version_1)
	importsMap["ext_storage_clear_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I64),
		wasm.NewValueTypes(),
	), ctx, ext_storage_clear_version_1)
	importsMap["ext_storage_clear_prefix_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I64),
		wasm.NewValueTypes(),
	), ctx, ext_storage_clear_prefix_version_1)
	importsMap["ext_storage_exists_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I64),
		wasm.NewValueTypes(wasm.I32),
	), ctx, ext_storage_exists_version_1)
	importsMap["ext_storage_get_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I64),
		wasm.NewValueTypes(wasm.I64),
	), ctx, ext_storage_get_version_1)
	importsMap["ext_storage_next_key_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I64),
		wasm.NewValueTypes(wasm.I64),
	), ctx, ext_storage_next_key_version_1)
	importsMap["ext_storage_read_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I64, wasm.I64, wasm.I32),
		wasm.NewValueTypes(wasm.I64),
	), ctx, ext_storage_read_version_1)
	importsMap["ext_storage_root_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(),
		wasm.NewValueTypes(wasm.I64),
	), ctx, ext_storage_root_version_1)
	importsMap["ext_storage_set_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I64, wasm.I64),
		wasm.NewValueTypes(),
	), ctx, ext_storage_set_version_1)
	importsMap["ext_storage_start_transaction_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(),
		wasm.NewValueTypes(),
	), ctx, ext_storage_start_transaction_version_1)
	importsMap["ext_storage_rollback_transaction_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(),
		wasm.NewValueTypes(),
	), ctx, ext_storage_rollback_transaction_version_1)
	importsMap["ext_storage_commit_transaction_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(),
		wasm.NewValueTypes(),
	), ctx, ext_storage_commit_transaction_version_1)

	importsMap["ext_offchain_timestamp_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(),
		wasm.NewValueTypes(wasm.I64),
	), ctx, ext_offchain_timestamp_version_1)

	importsMap["ext_offchain_sleep_until_version_1"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I64),
		wasm.NewValueTypes(),
	), ctx, ext_offchain_sleep_until_version_1)

	importsMap["ext_default_child_storage_storage_kill_version_2"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I64, wasm.I64),
		wasm.NewValueTypes(wasm.I32),
	), ctx, ext_default_child_storage_storage_kill_version_2)

	importsMap["ext_default_child_storage_storage_kill_version_3"] = wasm.NewFunctionWithEnvironment(store, wasm.NewFunctionType(
		wasm.NewValueTypes(wasm.I64, wasm.I64),
		wasm.NewValueTypes(wasm.I32),
	), ctx, ext_default_child_storage_storage_kill_version_3)

	imports := wasm.NewImportObject()
	imports.Register("env", importsMap)
	return imports
}
