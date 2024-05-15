package wazero_runtime

import (
	"context"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

var RuntimeConfig wazero.RuntimeConfig
var HostRuntimeModule wazero.CompiledModule

func init() {
	RuntimeConfig = wazero.NewRuntimeConfig().WithCompilationCache(wazero.NewCompilationCache())
	rt := wazero.NewRuntimeWithConfig(context.Background(), RuntimeConfig)

	var err error
	HostRuntimeModule, err = rt.NewHostModuleBuilder("env").
		// values from newer kusama/polkadot runtimes
		ExportMemory("memory", 1080).
		NewFunctionBuilder().
		WithFunc(ext_logging_log_version_1).
		Export("ext_logging_log_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			stack[0] = api.EncodeU32(4)
		}), []api.ValueType{}, []api.ValueType{api.ValueTypeI32}).
		Export("ext_logging_max_level_version_1").
		NewFunctionBuilder().
		WithFunc(func(a int32, b int32, c int32) {
			panic("unimplemented")
		}).
		Export("ext_transaction_index_index_version_1").
		NewFunctionBuilder().
		WithFunc(func(a int32, b int32) {
			panic("unimplemented")
		}).
		Export("ext_transaction_index_renew_version_1").
		NewFunctionBuilder().
		WithFunc(func(a int32) {
			panic("unimplemented")
		}).
		Export("ext_sandbox_instance_teardown_version_1").
		NewFunctionBuilder().
		WithFunc(func(a int32, b int64, c int64, d int32) int32 {
			panic("unimplemented")
		}).
		Export("ext_sandbox_instantiate_version_1").
		NewFunctionBuilder().
		WithFunc(func(a int32, b int64, c int64, d int32, e int32, f int32) int32 {
			panic("unimplemented")
		}).
		Export("ext_sandbox_invoke_version_1").
		NewFunctionBuilder().
		WithFunc(func(a int32, b int32, c int32, d int32) int32 {
			panic("unimplemented")
		}).
		Export("ext_sandbox_memory_get_version_1").
		NewFunctionBuilder().
		WithFunc(func(a int32, b int32, c int32, d int32) int32 {
			panic("unimplemented")
		}).
		Export("ext_sandbox_memory_set_version_1").
		NewFunctionBuilder().
		WithFunc(func(a int32) {
			panic("unimplemented")
		}).
		Export("ext_sandbox_memory_teardown_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			stack[0] = api.EncodeU32(
				ext_crypto_ed25519_generate_version_1(ctx, mod, api.DecodeU32(stack[0]), stack[1]))
		}), []api.ValueType{api.ValueTypeI32, api.ValueTypeI64}, []api.ValueType{api.ValueTypeI32}).
		Export("ext_crypto_ed25519_generate_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			stack[0] = ext_crypto_ed25519_public_keys_version_1(ctx, mod, api.DecodeU32(stack[0]))
		}), []api.ValueType{api.ValueTypeI32}, []api.ValueType{api.ValueTypeI64}).
		Export("ext_crypto_ed25519_public_keys_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			stack[0] = ext_crypto_ed25519_sign_version_1(ctx, mod,
				api.DecodeU32(stack[0]), api.DecodeU32(stack[1]), stack[2])
		}), []api.ValueType{api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI64}, []api.ValueType{api.ValueTypeI64}).
		Export("ext_crypto_ed25519_sign_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			stack[0] = api.EncodeU32(
				ext_crypto_ed25519_verify_version_1(ctx, mod,
					api.DecodeU32(stack[0]), stack[1], api.DecodeU32(stack[2])))
		}), []api.ValueType{api.ValueTypeI32, api.ValueTypeI64, api.ValueTypeI32}, []api.ValueType{api.ValueTypeI32}).
		Export("ext_crypto_ed25519_verify_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			stack[0] = ext_crypto_secp256k1_ecdsa_recover_version_1(ctx, mod,
				api.DecodeU32(stack[0]), api.DecodeU32(stack[1]))
		}), []api.ValueType{api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{api.ValueTypeI64}).
		Export("ext_crypto_secp256k1_ecdsa_recover_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			stack[0] = ext_crypto_secp256k1_ecdsa_recover_version_2(ctx, mod,
				api.DecodeU32(stack[0]), api.DecodeU32(stack[1]))
		}), []api.ValueType{api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{api.ValueTypeI64}).
		Export("ext_crypto_secp256k1_ecdsa_recover_version_2").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			stack[0] = api.EncodeU32(
				ext_crypto_ecdsa_verify_version_2(ctx, mod,
					api.DecodeU32(stack[0]), stack[1], api.DecodeU32(stack[2])))
		}), []api.ValueType{api.ValueTypeI32, api.ValueTypeI64, api.ValueTypeI32}, []api.ValueType{api.ValueTypeI32}).
		Export("ext_crypto_ecdsa_verify_version_2").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			stack[0] = ext_crypto_secp256k1_ecdsa_recover_compressed_version_1(ctx, mod,
				api.DecodeU32(stack[0]), api.DecodeU32(stack[1]))
		}), []api.ValueType{api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{api.ValueTypeI64}).
		Export("ext_crypto_secp256k1_ecdsa_recover_compressed_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			stack[0] = ext_crypto_secp256k1_ecdsa_recover_compressed_version_2(ctx, mod,
				api.DecodeU32(stack[0]), api.DecodeU32(stack[1]))
		}), []api.ValueType{api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{api.ValueTypeI64}).
		Export("ext_crypto_secp256k1_ecdsa_recover_compressed_version_2").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			stack[0] = api.EncodeU32(ext_crypto_sr25519_generate_version_1(ctx, mod,
				api.DecodeU32(stack[0]), stack[1]))
		}), []api.ValueType{api.ValueTypeI32, api.ValueTypeI64}, []api.ValueType{api.ValueTypeI32}).
		Export("ext_crypto_sr25519_generate_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			stack[0] = ext_crypto_sr25519_public_keys_version_1(ctx, mod, api.DecodeU32(stack[0]))
		}), []api.ValueType{api.ValueTypeI32}, []api.ValueType{api.ValueTypeI64}).
		Export("ext_crypto_sr25519_public_keys_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			stack[0] = ext_crypto_sr25519_sign_version_1(ctx, mod,
				api.DecodeU32(stack[0]), api.DecodeU32(stack[1]), stack[2])
		}), []api.ValueType{api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI64}, []api.ValueType{api.ValueTypeI64}).
		Export("ext_crypto_sr25519_sign_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			stack[0] = api.EncodeU32(ext_crypto_sr25519_verify_version_1(ctx, mod,
				api.DecodeU32(stack[0]), stack[1], api.DecodeU32(stack[2])))
		}), []api.ValueType{api.ValueTypeI32, api.ValueTypeI64, api.ValueTypeI32}, []api.ValueType{api.ValueTypeI32}).
		Export("ext_crypto_sr25519_verify_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			stack[0] = api.EncodeU32(ext_crypto_sr25519_verify_version_2(ctx, mod,
				api.DecodeU32(stack[0]), stack[1], api.DecodeU32(stack[2])))
		}), []api.ValueType{api.ValueTypeI32, api.ValueTypeI64, api.ValueTypeI32}, []api.ValueType{api.ValueTypeI32}).
		Export("ext_crypto_sr25519_verify_version_2").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			ext_crypto_start_batch_verify_version_1(ctx, mod)
		}), []api.ValueType{}, []api.ValueType{}).
		Export("ext_crypto_start_batch_verify_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			stack[0] = api.EncodeU32(ext_crypto_finish_batch_verify_version_1(ctx, mod))
		}), []api.ValueType{}, []api.ValueType{api.ValueTypeI32}).
		Export("ext_crypto_finish_batch_verify_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			stack[0] = api.EncodeU32(ext_trie_blake2_256_root_version_1(ctx, mod, stack[0]))
		}), []api.ValueType{api.ValueTypeI64}, []api.ValueType{api.ValueTypeI32}).
		Export("ext_trie_blake2_256_root_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			stack[0] = api.EncodeU32(ext_trie_blake2_256_root_version_2(ctx, mod, stack[0], api.DecodeU32(stack[1])))
		}), []api.ValueType{api.ValueTypeI64, api.ValueTypeI32}, []api.ValueType{api.ValueTypeI32}).
		Export("ext_trie_blake2_256_root_version_2").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			stack[0] = api.EncodeU32(ext_trie_blake2_256_ordered_root_version_1(ctx, mod, stack[0]))
		}), []api.ValueType{api.ValueTypeI64}, []api.ValueType{api.ValueTypeI32}).
		Export("ext_trie_blake2_256_ordered_root_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			stack[0] = api.EncodeU32(ext_trie_blake2_256_ordered_root_version_2(
				ctx, mod, stack[0], api.DecodeU32(stack[1])))
		}), []api.ValueType{api.ValueTypeI64, api.ValueTypeI32}, []api.ValueType{api.ValueTypeI32}).
		Export("ext_trie_blake2_256_ordered_root_version_2").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			stack[0] = api.EncodeU32(
				ext_trie_blake2_256_verify_proof_version_1(ctx, mod,
					api.DecodeU32(stack[0]), stack[1], stack[2], stack[3]))
		}), []api.ValueType{api.ValueTypeI32, api.ValueTypeI64, api.ValueTypeI64, api.ValueTypeI64},
			[]api.ValueType{api.ValueTypeI32}).
		Export("ext_trie_blake2_256_verify_proof_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			stack[0] = api.EncodeU32(ext_trie_blake2_256_verify_proof_version_2(ctx, mod,
				api.DecodeU32(stack[0]), stack[1], stack[2], stack[3], api.DecodeU32(stack[4])))
		}), []api.ValueType{api.ValueTypeI32, api.ValueTypeI64, api.ValueTypeI64, api.ValueTypeI64, api.ValueTypeI32},
			[]api.ValueType{api.ValueTypeI32}).
		Export("ext_trie_blake2_256_verify_proof_version_2").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			ext_misc_print_hex_version_1(ctx, mod, stack[0])
		}), []api.ValueType{api.ValueTypeI64}, []api.ValueType{}).
		Export("ext_misc_print_hex_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			ext_misc_print_num_version_1(ctx, mod, stack[0])
		}), []api.ValueType{api.ValueTypeI64}, []api.ValueType{}).
		Export("ext_misc_print_num_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			ext_misc_print_utf8_version_1(ctx, mod, stack[0])
		}), []api.ValueType{api.ValueTypeI64}, []api.ValueType{}).
		Export("ext_misc_print_utf8_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			stack[0] = ext_misc_runtime_version_version_1(ctx, mod, stack[0])
		}), []api.ValueType{api.ValueTypeI64}, []api.ValueType{api.ValueTypeI64}).
		Export("ext_misc_runtime_version_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			ext_default_child_storage_set_version_1(ctx, mod, stack[0], stack[1], stack[3])
		}), []api.ValueType{api.ValueTypeI64, api.ValueTypeI64, api.ValueTypeI64}, []api.ValueType{}).
		Export("ext_default_child_storage_set_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			stack[0] = ext_default_child_storage_read_version_1(ctx, mod, stack[0], stack[1], stack[2], api.DecodeU32(stack[3]))
		}), []api.ValueType{api.ValueTypeI64, api.ValueTypeI64, api.ValueTypeI64, api.ValueTypeI32},
			[]api.ValueType{api.ValueTypeI64}).
		Export("ext_default_child_storage_read_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			ext_default_child_storage_clear_version_1(ctx, mod, stack[0], stack[1])
		}), []api.ValueType{api.ValueTypeI64, api.ValueTypeI64}, []api.ValueType{}).
		Export("ext_default_child_storage_clear_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			ext_default_child_storage_clear_prefix_version_1(ctx, mod, stack[0], stack[1])
		}), []api.ValueType{api.ValueTypeI64, api.ValueTypeI64}, []api.ValueType{}).
		Export("ext_default_child_storage_clear_prefix_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			stack[0] = ext_default_child_storage_clear_prefix_version_2(ctx, mod, stack[0], stack[1], stack[2])
		}), []api.ValueType{api.ValueTypeI64, api.ValueTypeI64, api.ValueTypeI64}, []api.ValueType{api.ValueTypeI64}).
		Export("ext_default_child_storage_clear_prefix_version_2").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			stack[0] = api.EncodeU32(ext_default_child_storage_exists_version_1(ctx, mod, stack[0], stack[1]))
		}), []api.ValueType{api.ValueTypeI64, api.ValueTypeI64}, []api.ValueType{api.ValueTypeI32}).
		Export("ext_default_child_storage_exists_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			stack[0] = ext_default_child_storage_get_version_1(ctx, mod, stack[0], stack[1])
		}), []api.ValueType{api.ValueTypeI64, api.ValueTypeI64}, []api.ValueType{api.ValueTypeI64}).
		Export("ext_default_child_storage_get_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			stack[0] = ext_default_child_storage_next_key_version_1(ctx, mod, stack[0], stack[1])
		}), []api.ValueType{api.ValueTypeI64, api.ValueTypeI64}, []api.ValueType{api.ValueTypeI64}).
		Export("ext_default_child_storage_next_key_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			stack[0] = ext_default_child_storage_root_version_1(ctx, mod, stack[0])
		}), []api.ValueType{api.ValueTypeI64}, []api.ValueType{api.ValueTypeI64}).
		Export("ext_default_child_storage_root_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			stack[0] = ext_default_child_storage_root_version_2(ctx, mod, stack[0], api.DecodeU32(stack[1]))
		}), []api.ValueType{api.ValueTypeI64, api.ValueTypeI32}, []api.ValueType{api.ValueTypeI64}).
		Export("ext_default_child_storage_root_version_2").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			ext_default_child_storage_storage_kill_version_1(ctx, mod, stack[0])
		}), []api.ValueType{api.ValueTypeI64}, []api.ValueType{}).
		Export("ext_default_child_storage_storage_kill_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			stack[0] = api.EncodeU32(ext_default_child_storage_storage_kill_version_2(ctx, mod, stack[0], stack[1]))
		}), []api.ValueType{api.ValueTypeI64, api.ValueTypeI64}, []api.ValueType{api.ValueTypeI32}).
		Export("ext_default_child_storage_storage_kill_version_2").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			stack[0] = ext_default_child_storage_storage_kill_version_3(ctx, mod, stack[0], stack[1])
		}), []api.ValueType{api.ValueTypeI64, api.ValueTypeI64}, []api.ValueType{api.ValueTypeI64}).
		Export("ext_default_child_storage_storage_kill_version_3").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			addr := api.DecodeU32(stack[0])
			ext_allocator_free_version_1(ctx, mod, addr)
		}), []api.ValueType{api.ValueTypeI32}, []api.ValueType{}).
		Export("ext_allocator_free_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			size := api.DecodeU32(stack[0])
			stack[0] = api.EncodeU32(ext_allocator_malloc_version_1(ctx, mod, size))
		}), []api.ValueType{api.ValueTypeI32}, []api.ValueType{api.ValueTypeI32}).
		Export("ext_allocator_malloc_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			stack[0] = api.EncodeU32(ext_hashing_blake2_128_version_1(ctx, mod, stack[0]))
		}), []api.ValueType{api.ValueTypeI64}, []api.ValueType{api.ValueTypeI32}).
		Export("ext_hashing_blake2_128_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			stack[0] = api.EncodeU32(ext_hashing_blake2_256_version_1(ctx, mod, stack[0]))
		}), []api.ValueType{api.ValueTypeI64}, []api.ValueType{api.ValueTypeI32}).
		Export("ext_hashing_blake2_256_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			stack[0] = api.EncodeU32(ext_hashing_keccak_256_version_1(ctx, mod, stack[0]))
		}), []api.ValueType{api.ValueTypeI64}, []api.ValueType{api.ValueTypeI32}).
		Export("ext_hashing_keccak_256_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			stack[0] = api.EncodeU32(ext_hashing_sha2_256_version_1(ctx, mod, stack[0]))
		}), []api.ValueType{api.ValueTypeI64}, []api.ValueType{api.ValueTypeI32}).
		Export("ext_hashing_sha2_256_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			stack[0] = api.EncodeU32(ext_hashing_twox_256_version_1(ctx, mod, stack[0]))
		}), []api.ValueType{api.ValueTypeI64}, []api.ValueType{api.ValueTypeI32}).
		Export("ext_hashing_twox_256_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			stack[0] = api.EncodeU32(ext_hashing_twox_128_version_1(ctx, mod, stack[0]))
		}), []api.ValueType{api.ValueTypeI64}, []api.ValueType{api.ValueTypeI32}).
		Export("ext_hashing_twox_128_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			stack[0] = api.EncodeU32(ext_hashing_twox_64_version_1(ctx, mod, stack[0]))
		}), []api.ValueType{api.ValueTypeI64}, []api.ValueType{api.ValueTypeI32}).
		Export("ext_hashing_twox_64_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			ext_offchain_index_set_version_1(ctx, mod, stack[0], stack[1])
		}), []api.ValueType{api.ValueTypeI64, api.ValueTypeI64}, []api.ValueType{}).
		Export("ext_offchain_index_set_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			ext_offchain_index_clear_version_1(ctx, mod, stack[0])
		}), []api.ValueType{api.ValueTypeI64}, []api.ValueType{}).
		Export("ext_offchain_index_clear_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			ext_offchain_local_storage_clear_version_1(ctx, mod, api.DecodeU32(stack[0]), stack[1])
		}), []api.ValueType{api.ValueTypeI32, api.ValueTypeI64}, []api.ValueType{}).
		Export("ext_offchain_local_storage_clear_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			stack[0] = api.EncodeU32(ext_offchain_is_validator_version_1(ctx, mod))
		}), []api.ValueType{}, []api.ValueType{api.ValueTypeI32}).
		Export("ext_offchain_is_validator_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			stack[0] = api.EncodeU32(ext_offchain_local_storage_compare_and_set_version_1(ctx, mod,
				api.DecodeU32(stack[0]), stack[1], stack[2], stack[3]))
		}), []api.ValueType{api.ValueTypeI32, api.ValueTypeI64, api.ValueTypeI64, api.ValueTypeI64},
			[]api.ValueType{api.ValueTypeI32}).
		Export("ext_offchain_local_storage_compare_and_set_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			stack[0] = ext_offchain_local_storage_get_version_1(ctx, mod, api.DecodeU32(stack[0]), stack[1])
		}), []api.ValueType{api.ValueTypeI32, api.ValueTypeI64}, []api.ValueType{api.ValueTypeI64}).
		Export("ext_offchain_local_storage_get_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			ext_offchain_local_storage_set_version_1(ctx, mod, api.DecodeU32(stack[0]), stack[1], stack[2])
		}), []api.ValueType{api.ValueTypeI32, api.ValueTypeI64, api.ValueTypeI64}, []api.ValueType{}).
		Export("ext_offchain_local_storage_set_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			stack[0] = ext_offchain_network_state_version_1(ctx, mod)
		}), []api.ValueType{}, []api.ValueType{api.ValueTypeI64}).
		Export("ext_offchain_network_state_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			stack[0] = api.EncodeU32(ext_offchain_random_seed_version_1(ctx, mod))
		}), []api.ValueType{}, []api.ValueType{api.ValueTypeI32}).
		Export("ext_offchain_random_seed_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			stack[0] = ext_offchain_submit_transaction_version_1(ctx, mod, stack[0])
		}), []api.ValueType{api.ValueTypeI64}, []api.ValueType{api.ValueTypeI64}).
		Export("ext_offchain_submit_transaction_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			stack[0] = ext_offchain_timestamp_version_1(ctx, mod)
		}), []api.ValueType{}, []api.ValueType{api.ValueTypeI64}).
		Export("ext_offchain_timestamp_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			ext_offchain_sleep_until_version_1(ctx, mod, stack[0])
		}), []api.ValueType{api.ValueTypeI64}, []api.ValueType{}).
		Export("ext_offchain_sleep_until_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			stack[0] = ext_offchain_http_request_start_version_1(ctx, mod, stack[0], stack[1], stack[2])
		}), []api.ValueType{api.ValueTypeI64, api.ValueTypeI64, api.ValueTypeI64}, []api.ValueType{api.ValueTypeI64}).
		Export("ext_offchain_http_request_start_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			stack[0] = ext_offchain_http_request_add_header_version_1(ctx, mod, api.DecodeU32(stack[0]), stack[1], stack[2])
		}), []api.ValueType{api.ValueTypeI32, api.ValueTypeI64, api.ValueTypeI64}, []api.ValueType{api.ValueTypeI64}).
		Export("ext_offchain_http_request_add_header_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			ext_storage_append_version_1(ctx, mod, stack[0], stack[1])
		}), []api.ValueType{api.ValueTypeI64, api.ValueTypeI64}, []api.ValueType{}).
		Export("ext_storage_append_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			stack[0] = ext_storage_changes_root_version_1(ctx, mod, stack[0])
		}), []api.ValueType{api.ValueTypeI64}, []api.ValueType{api.ValueTypeI64}).
		Export("ext_storage_changes_root_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			ext_storage_clear_version_1(ctx, mod, stack[0])
		}), []api.ValueType{api.ValueTypeI64}, []api.ValueType{}).
		Export("ext_storage_clear_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			ext_storage_clear_prefix_version_1(ctx, mod, stack[0])
		}), []api.ValueType{api.ValueTypeI64}, []api.ValueType{}).
		Export("ext_storage_clear_prefix_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			stack[0] = ext_storage_clear_prefix_version_2(ctx, mod, stack[0], stack[1])
		}), []api.ValueType{api.ValueTypeI64, api.ValueTypeI64}, []api.ValueType{api.ValueTypeI64}).
		Export("ext_storage_clear_prefix_version_2").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			stack[0] = api.EncodeU32(ext_storage_exists_version_1(ctx, mod, stack[0]))
		}), []api.ValueType{api.ValueTypeI64}, []api.ValueType{api.ValueTypeI32}).
		Export("ext_storage_exists_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			stack[0] = ext_storage_get_version_1(ctx, mod, stack[0])
		}), []api.ValueType{api.ValueTypeI64}, []api.ValueType{api.ValueTypeI64}).
		Export("ext_storage_get_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			stack[0] = ext_storage_next_key_version_1(ctx, mod, stack[0])
		}), []api.ValueType{api.ValueTypeI64}, []api.ValueType{api.ValueTypeI64}).
		Export("ext_storage_next_key_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			stack[0] = ext_storage_read_version_1(ctx, mod, stack[0], stack[1], api.DecodeU32(stack[2]))
		}), []api.ValueType{api.ValueTypeI64, api.ValueTypeI64, api.ValueTypeI32}, []api.ValueType{api.ValueTypeI64}).
		Export("ext_storage_read_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			stack[0] = ext_storage_root_version_1(ctx, mod)
		}), []api.ValueType{}, []api.ValueType{api.ValueTypeI64}).
		Export("ext_storage_root_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			stack[0] = ext_storage_root_version_2(ctx, mod, api.DecodeU32(stack[0]))
		}), []api.ValueType{api.ValueTypeI32}, []api.ValueType{api.ValueTypeI64}).
		Export("ext_storage_root_version_2").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			ext_storage_set_version_1(ctx, mod, stack[0], stack[1])
		}), []api.ValueType{api.ValueTypeI64, api.ValueTypeI64}, []api.ValueType{}).
		Export("ext_storage_set_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			ext_storage_start_transaction_version_1(ctx, mod)
		}), []api.ValueType{}, []api.ValueType{}).
		Export("ext_storage_start_transaction_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			ext_storage_rollback_transaction_version_1(ctx, mod)
		}), []api.ValueType{}, []api.ValueType{}).
		Export("ext_storage_rollback_transaction_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			ext_storage_commit_transaction_version_1(ctx, mod)
		}), []api.ValueType{}, []api.ValueType{}).
		Export("ext_storage_commit_transaction_version_1").
		NewFunctionBuilder().
		WithGoModuleFunction(api.GoModuleFunc(func(ctx context.Context, mod api.Module, stack []uint64) {
			stack[0] = api.EncodeU32(ext_crypto_ecdsa_generate_version_1(ctx, mod, api.DecodeU32(stack[0]), stack[1]))
		}), []api.ValueType{api.ValueTypeI32, api.ValueTypeI64}, []api.ValueType{api.ValueTypeI32}).
		Export("ext_crypto_ecdsa_generate_version_1").
		Compile(context.Background())

	if err != nil {
		panic("failed to compile host runtime module")
	}
}
