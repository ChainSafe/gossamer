"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.hmacSha512 = exports.hmacSha256 = exports.ed25519Verify = exports.ed25519Sign = exports.ed25519KeypairFromSeed = exports.blake2b = exports.bip39Validate = exports.bip39ToSeed = exports.bip39ToMiniSecret = exports.bip39ToEntropy = exports.bip39Generate = void 0;
exports.isReady = isReady;
exports.keccak512 = exports.keccak256 = void 0;
Object.defineProperty(exports, "packageInfo", {
  enumerable: true,
  get: function () {
    return _packageInfo.packageInfo;
  }
});
exports.vrfVerify = exports.vrfSign = exports.twox = exports.sr25519Verify = exports.sr25519Sign = exports.sr25519KeypairFromSeed = exports.sr25519DerivePublicSoft = exports.sr25519DeriveKeypairSoft = exports.sr25519DeriveKeypairHard = exports.sr25519Agree = exports.sha512 = exports.sha256 = exports.secp256k1Sign = exports.secp256k1Recover = exports.secp256k1FromSeed = exports.secp256k1Expand = exports.secp256k1Compress = exports.scrypt = exports.pbkdf2 = void 0;
exports.waitReady = waitReady;

var _wasmCryptoAsmjs = require("@polkadot/wasm-crypto-asmjs");

var _wasmCryptoWasm = require("@polkadot/wasm-crypto-wasm");

var _bridge = require("./bridge.cjs");

var imports = _interopRequireWildcard(require("./imports.cjs"));

var _packageInfo = require("./packageInfo.cjs");

function _getRequireWildcardCache(nodeInterop) { if (typeof WeakMap !== "function") return null; var cacheBabelInterop = new WeakMap(); var cacheNodeInterop = new WeakMap(); return (_getRequireWildcardCache = function (nodeInterop) { return nodeInterop ? cacheNodeInterop : cacheBabelInterop; })(nodeInterop); }

function _interopRequireWildcard(obj, nodeInterop) { if (!nodeInterop && obj && obj.__esModule) { return obj; } if (obj === null || typeof obj !== "object" && typeof obj !== "function") { return { default: obj }; } var cache = _getRequireWildcardCache(nodeInterop); if (cache && cache.has(obj)) { return cache.get(obj); } var newObj = {}; var hasPropertyDescriptor = Object.defineProperty && Object.getOwnPropertyDescriptor; for (var key in obj) { if (key !== "default" && Object.prototype.hasOwnProperty.call(obj, key)) { var desc = hasPropertyDescriptor ? Object.getOwnPropertyDescriptor(obj, key) : null; if (desc && (desc.get || desc.set)) { Object.defineProperty(newObj, key, desc); } else { newObj[key] = obj[key]; } } } newObj.default = obj; if (cache) { cache.set(obj, newObj); } return newObj; }

// Copyright 2019-2021 @polkadot/wasm-crypto authors & contributors
// SPDX-License-Identifier: Apache-2.0
const wasmPromise = (0, _bridge.initWasm)(_wasmCryptoWasm.wasmBytes, _wasmCryptoAsmjs.asmJsInit, imports).catch(() => null);
const bip39Generate = (0, _bridge.withWasm)((wasm, words) => {
  wasm.ext_bip39_generate(8, words);
  return (0, _bridge.resultString)();
});
exports.bip39Generate = bip39Generate;
const bip39ToEntropy = (0, _bridge.withWasm)((wasm, phrase) => {
  wasm.ext_bip39_to_entropy(8, ...(0, _bridge.allocString)(phrase));
  return (0, _bridge.resultU8a)();
});
exports.bip39ToEntropy = bip39ToEntropy;
const bip39ToMiniSecret = (0, _bridge.withWasm)((wasm, phrase, password) => {
  wasm.ext_bip39_to_mini_secret(8, ...(0, _bridge.allocString)(phrase), ...(0, _bridge.allocString)(password));
  return (0, _bridge.resultU8a)();
});
exports.bip39ToMiniSecret = bip39ToMiniSecret;
const bip39ToSeed = (0, _bridge.withWasm)((wasm, phrase, password) => {
  wasm.ext_bip39_to_seed(8, ...(0, _bridge.allocString)(phrase), ...(0, _bridge.allocString)(password));
  return (0, _bridge.resultU8a)();
});
exports.bip39ToSeed = bip39ToSeed;
const bip39Validate = (0, _bridge.withWasm)((wasm, phrase) => {
  const ret = wasm.ext_bip39_validate(...(0, _bridge.allocString)(phrase));
  return ret !== 0;
});
exports.bip39Validate = bip39Validate;
const ed25519KeypairFromSeed = (0, _bridge.withWasm)((wasm, seed) => {
  wasm.ext_ed_from_seed(8, ...(0, _bridge.allocU8a)(seed));
  return (0, _bridge.resultU8a)();
});
exports.ed25519KeypairFromSeed = ed25519KeypairFromSeed;
const ed25519Sign = (0, _bridge.withWasm)((wasm, pubkey, seckey, message) => {
  wasm.ext_ed_sign(8, ...(0, _bridge.allocU8a)(pubkey), ...(0, _bridge.allocU8a)(seckey), ...(0, _bridge.allocU8a)(message));
  return (0, _bridge.resultU8a)();
});
exports.ed25519Sign = ed25519Sign;
const ed25519Verify = (0, _bridge.withWasm)((wasm, signature, message, pubkey) => {
  const ret = wasm.ext_ed_verify(...(0, _bridge.allocU8a)(signature), ...(0, _bridge.allocU8a)(message), ...(0, _bridge.allocU8a)(pubkey));
  return ret !== 0;
});
exports.ed25519Verify = ed25519Verify;
const secp256k1FromSeed = (0, _bridge.withWasm)((wasm, seckey) => {
  wasm.ext_secp_from_seed(8, ...(0, _bridge.allocU8a)(seckey));
  return (0, _bridge.resultU8a)();
});
exports.secp256k1FromSeed = secp256k1FromSeed;
const secp256k1Compress = (0, _bridge.withWasm)((wasm, pubkey) => {
  wasm.ext_secp_pub_compress(8, ...(0, _bridge.allocU8a)(pubkey));
  return (0, _bridge.resultU8a)();
});
exports.secp256k1Compress = secp256k1Compress;
const secp256k1Expand = (0, _bridge.withWasm)((wasm, pubkey) => {
  wasm.ext_secp_pub_expand(8, ...(0, _bridge.allocU8a)(pubkey));
  return (0, _bridge.resultU8a)();
});
exports.secp256k1Expand = secp256k1Expand;
const secp256k1Recover = (0, _bridge.withWasm)((wasm, msgHash, sig, recovery) => {
  wasm.ext_secp_recover(8, ...(0, _bridge.allocU8a)(msgHash), ...(0, _bridge.allocU8a)(sig), recovery);
  return (0, _bridge.resultU8a)();
});
exports.secp256k1Recover = secp256k1Recover;
const secp256k1Sign = (0, _bridge.withWasm)((wasm, msgHash, seckey) => {
  wasm.ext_secp_sign(8, ...(0, _bridge.allocU8a)(msgHash), ...(0, _bridge.allocU8a)(seckey));
  return (0, _bridge.resultU8a)();
});
exports.secp256k1Sign = secp256k1Sign;
const sr25519DeriveKeypairHard = (0, _bridge.withWasm)((wasm, pair, cc) => {
  wasm.ext_sr_derive_keypair_hard(8, ...(0, _bridge.allocU8a)(pair), ...(0, _bridge.allocU8a)(cc));
  return (0, _bridge.resultU8a)();
});
exports.sr25519DeriveKeypairHard = sr25519DeriveKeypairHard;
const sr25519DeriveKeypairSoft = (0, _bridge.withWasm)((wasm, pair, cc) => {
  wasm.ext_sr_derive_keypair_soft(8, ...(0, _bridge.allocU8a)(pair), ...(0, _bridge.allocU8a)(cc));
  return (0, _bridge.resultU8a)();
});
exports.sr25519DeriveKeypairSoft = sr25519DeriveKeypairSoft;
const sr25519DerivePublicSoft = (0, _bridge.withWasm)((wasm, pubkey, cc) => {
  wasm.ext_sr_derive_public_soft(8, ...(0, _bridge.allocU8a)(pubkey), ...(0, _bridge.allocU8a)(cc));
  return (0, _bridge.resultU8a)();
});
exports.sr25519DerivePublicSoft = sr25519DerivePublicSoft;
const sr25519KeypairFromSeed = (0, _bridge.withWasm)((wasm, seed) => {
  wasm.ext_sr_from_seed(8, ...(0, _bridge.allocU8a)(seed));
  return (0, _bridge.resultU8a)();
});
exports.sr25519KeypairFromSeed = sr25519KeypairFromSeed;
const sr25519Sign = (0, _bridge.withWasm)((wasm, pubkey, secret, message) => {
  wasm.ext_sr_sign(8, ...(0, _bridge.allocU8a)(pubkey), ...(0, _bridge.allocU8a)(secret), ...(0, _bridge.allocU8a)(message));
  return (0, _bridge.resultU8a)();
});
exports.sr25519Sign = sr25519Sign;
const sr25519Verify = (0, _bridge.withWasm)((wasm, signature, message, pubkey) => {
  const ret = wasm.ext_sr_verify(...(0, _bridge.allocU8a)(signature), ...(0, _bridge.allocU8a)(message), ...(0, _bridge.allocU8a)(pubkey));
  return ret !== 0;
});
exports.sr25519Verify = sr25519Verify;
const sr25519Agree = (0, _bridge.withWasm)((wasm, pubkey, secret) => {
  wasm.ext_sr_agree(8, ...(0, _bridge.allocU8a)(pubkey), ...(0, _bridge.allocU8a)(secret));
  return (0, _bridge.resultU8a)();
});
exports.sr25519Agree = sr25519Agree;
const vrfSign = (0, _bridge.withWasm)((wasm, secret, context, message, extra) => {
  wasm.ext_vrf_sign(8, ...(0, _bridge.allocU8a)(secret), ...(0, _bridge.allocU8a)(context), ...(0, _bridge.allocU8a)(message), ...(0, _bridge.allocU8a)(extra));
  return (0, _bridge.resultU8a)();
});
exports.vrfSign = vrfSign;
const vrfVerify = (0, _bridge.withWasm)((wasm, pubkey, context, message, extra, outAndProof) => {
  const ret = wasm.ext_vrf_verify(...(0, _bridge.allocU8a)(pubkey), ...(0, _bridge.allocU8a)(context), ...(0, _bridge.allocU8a)(message), ...(0, _bridge.allocU8a)(extra), ...(0, _bridge.allocU8a)(outAndProof));
  return ret !== 0;
});
exports.vrfVerify = vrfVerify;
const blake2b = (0, _bridge.withWasm)((wasm, data, key, size) => {
  wasm.ext_blake2b(8, ...(0, _bridge.allocU8a)(data), ...(0, _bridge.allocU8a)(key), size);
  return (0, _bridge.resultU8a)();
});
exports.blake2b = blake2b;
const hmacSha256 = (0, _bridge.withWasm)((wasm, key, data) => {
  wasm.ext_hmac_sha256(8, ...(0, _bridge.allocU8a)(key), ...(0, _bridge.allocU8a)(data));
  return (0, _bridge.resultU8a)();
});
exports.hmacSha256 = hmacSha256;
const hmacSha512 = (0, _bridge.withWasm)((wasm, key, data) => {
  wasm.ext_hmac_sha512(8, ...(0, _bridge.allocU8a)(key), ...(0, _bridge.allocU8a)(data));
  return (0, _bridge.resultU8a)();
});
exports.hmacSha512 = hmacSha512;
const keccak256 = (0, _bridge.withWasm)((wasm, data) => {
  wasm.ext_keccak256(8, ...(0, _bridge.allocU8a)(data));
  return (0, _bridge.resultU8a)();
});
exports.keccak256 = keccak256;
const keccak512 = (0, _bridge.withWasm)((wasm, data) => {
  wasm.ext_keccak512(8, ...(0, _bridge.allocU8a)(data));
  return (0, _bridge.resultU8a)();
});
exports.keccak512 = keccak512;
const pbkdf2 = (0, _bridge.withWasm)((wasm, data, salt, rounds) => {
  wasm.ext_pbkdf2(8, ...(0, _bridge.allocU8a)(data), ...(0, _bridge.allocU8a)(salt), rounds);
  return (0, _bridge.resultU8a)();
});
exports.pbkdf2 = pbkdf2;
const scrypt = (0, _bridge.withWasm)((wasm, password, salt, log2n, r, p) => {
  wasm.ext_scrypt(8, ...(0, _bridge.allocU8a)(password), ...(0, _bridge.allocU8a)(salt), log2n, r, p);
  return (0, _bridge.resultU8a)();
});
exports.scrypt = scrypt;
const sha256 = (0, _bridge.withWasm)((wasm, data) => {
  wasm.ext_sha256(8, ...(0, _bridge.allocU8a)(data));
  return (0, _bridge.resultU8a)();
});
exports.sha256 = sha256;
const sha512 = (0, _bridge.withWasm)((wasm, data) => {
  wasm.ext_sha512(8, ...(0, _bridge.allocU8a)(data));
  return (0, _bridge.resultU8a)();
});
exports.sha512 = sha512;
const twox = (0, _bridge.withWasm)((wasm, data, rounds) => {
  wasm.ext_twox(8, ...(0, _bridge.allocU8a)(data), rounds);
  return (0, _bridge.resultU8a)();
});
exports.twox = twox;

function isReady() {
  return !!(0, _bridge.getWasm)();
}

function waitReady() {
  return wasmPromise.then(() => isReady());
}