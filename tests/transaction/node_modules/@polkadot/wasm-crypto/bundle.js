// Copyright 2019-2021 @polkadot/wasm-crypto authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { asmJsInit } from '@polkadot/wasm-crypto-asmjs';
import { wasmBytes } from '@polkadot/wasm-crypto-wasm';
import { allocString, allocU8a, getWasm, initWasm, resultString, resultU8a, withWasm } from "./bridge.js";
import * as imports from "./imports.js";
export { packageInfo } from "./packageInfo.js";
const wasmPromise = initWasm(wasmBytes, asmJsInit, imports).catch(() => null);
export const bip39Generate = withWasm((wasm, words) => {
  wasm.ext_bip39_generate(8, words);
  return resultString();
});
export const bip39ToEntropy = withWasm((wasm, phrase) => {
  wasm.ext_bip39_to_entropy(8, ...allocString(phrase));
  return resultU8a();
});
export const bip39ToMiniSecret = withWasm((wasm, phrase, password) => {
  wasm.ext_bip39_to_mini_secret(8, ...allocString(phrase), ...allocString(password));
  return resultU8a();
});
export const bip39ToSeed = withWasm((wasm, phrase, password) => {
  wasm.ext_bip39_to_seed(8, ...allocString(phrase), ...allocString(password));
  return resultU8a();
});
export const bip39Validate = withWasm((wasm, phrase) => {
  const ret = wasm.ext_bip39_validate(...allocString(phrase));
  return ret !== 0;
});
export const ed25519KeypairFromSeed = withWasm((wasm, seed) => {
  wasm.ext_ed_from_seed(8, ...allocU8a(seed));
  return resultU8a();
});
export const ed25519Sign = withWasm((wasm, pubkey, seckey, message) => {
  wasm.ext_ed_sign(8, ...allocU8a(pubkey), ...allocU8a(seckey), ...allocU8a(message));
  return resultU8a();
});
export const ed25519Verify = withWasm((wasm, signature, message, pubkey) => {
  const ret = wasm.ext_ed_verify(...allocU8a(signature), ...allocU8a(message), ...allocU8a(pubkey));
  return ret !== 0;
});
export const secp256k1FromSeed = withWasm((wasm, seckey) => {
  wasm.ext_secp_from_seed(8, ...allocU8a(seckey));
  return resultU8a();
});
export const secp256k1Compress = withWasm((wasm, pubkey) => {
  wasm.ext_secp_pub_compress(8, ...allocU8a(pubkey));
  return resultU8a();
});
export const secp256k1Expand = withWasm((wasm, pubkey) => {
  wasm.ext_secp_pub_expand(8, ...allocU8a(pubkey));
  return resultU8a();
});
export const secp256k1Recover = withWasm((wasm, msgHash, sig, recovery) => {
  wasm.ext_secp_recover(8, ...allocU8a(msgHash), ...allocU8a(sig), recovery);
  return resultU8a();
});
export const secp256k1Sign = withWasm((wasm, msgHash, seckey) => {
  wasm.ext_secp_sign(8, ...allocU8a(msgHash), ...allocU8a(seckey));
  return resultU8a();
});
export const sr25519DeriveKeypairHard = withWasm((wasm, pair, cc) => {
  wasm.ext_sr_derive_keypair_hard(8, ...allocU8a(pair), ...allocU8a(cc));
  return resultU8a();
});
export const sr25519DeriveKeypairSoft = withWasm((wasm, pair, cc) => {
  wasm.ext_sr_derive_keypair_soft(8, ...allocU8a(pair), ...allocU8a(cc));
  return resultU8a();
});
export const sr25519DerivePublicSoft = withWasm((wasm, pubkey, cc) => {
  wasm.ext_sr_derive_public_soft(8, ...allocU8a(pubkey), ...allocU8a(cc));
  return resultU8a();
});
export const sr25519KeypairFromSeed = withWasm((wasm, seed) => {
  wasm.ext_sr_from_seed(8, ...allocU8a(seed));
  return resultU8a();
});
export const sr25519Sign = withWasm((wasm, pubkey, secret, message) => {
  wasm.ext_sr_sign(8, ...allocU8a(pubkey), ...allocU8a(secret), ...allocU8a(message));
  return resultU8a();
});
export const sr25519Verify = withWasm((wasm, signature, message, pubkey) => {
  const ret = wasm.ext_sr_verify(...allocU8a(signature), ...allocU8a(message), ...allocU8a(pubkey));
  return ret !== 0;
});
export const sr25519Agree = withWasm((wasm, pubkey, secret) => {
  wasm.ext_sr_agree(8, ...allocU8a(pubkey), ...allocU8a(secret));
  return resultU8a();
});
export const vrfSign = withWasm((wasm, secret, context, message, extra) => {
  wasm.ext_vrf_sign(8, ...allocU8a(secret), ...allocU8a(context), ...allocU8a(message), ...allocU8a(extra));
  return resultU8a();
});
export const vrfVerify = withWasm((wasm, pubkey, context, message, extra, outAndProof) => {
  const ret = wasm.ext_vrf_verify(...allocU8a(pubkey), ...allocU8a(context), ...allocU8a(message), ...allocU8a(extra), ...allocU8a(outAndProof));
  return ret !== 0;
});
export const blake2b = withWasm((wasm, data, key, size) => {
  wasm.ext_blake2b(8, ...allocU8a(data), ...allocU8a(key), size);
  return resultU8a();
});
export const hmacSha256 = withWasm((wasm, key, data) => {
  wasm.ext_hmac_sha256(8, ...allocU8a(key), ...allocU8a(data));
  return resultU8a();
});
export const hmacSha512 = withWasm((wasm, key, data) => {
  wasm.ext_hmac_sha512(8, ...allocU8a(key), ...allocU8a(data));
  return resultU8a();
});
export const keccak256 = withWasm((wasm, data) => {
  wasm.ext_keccak256(8, ...allocU8a(data));
  return resultU8a();
});
export const keccak512 = withWasm((wasm, data) => {
  wasm.ext_keccak512(8, ...allocU8a(data));
  return resultU8a();
});
export const pbkdf2 = withWasm((wasm, data, salt, rounds) => {
  wasm.ext_pbkdf2(8, ...allocU8a(data), ...allocU8a(salt), rounds);
  return resultU8a();
});
export const scrypt = withWasm((wasm, password, salt, log2n, r, p) => {
  wasm.ext_scrypt(8, ...allocU8a(password), ...allocU8a(salt), log2n, r, p);
  return resultU8a();
});
export const sha256 = withWasm((wasm, data) => {
  wasm.ext_sha256(8, ...allocU8a(data));
  return resultU8a();
});
export const sha512 = withWasm((wasm, data) => {
  wasm.ext_sha512(8, ...allocU8a(data));
  return resultU8a();
});
export const twox = withWasm((wasm, data, rounds) => {
  wasm.ext_twox(8, ...allocU8a(data), rounds);
  return resultU8a();
});
export function isReady() {
  return !!getWasm();
}
export function waitReady() {
  return wasmPromise.then(() => isReady());
}