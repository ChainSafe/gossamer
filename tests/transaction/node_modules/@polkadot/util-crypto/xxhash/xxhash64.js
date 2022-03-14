// Copyright 2017-2022 @polkadot/util-crypto authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { _0n, _1n } from '@polkadot/util';
import { BigInt } from '@polkadot/x-bigint'; // Adapted from https://github.com/pierrec/js-xxhash/blob/0504e76f3d31a21ae8528a7f590c7289c9e431d2/lib/xxhash64.js
//
// xxHash64 implementation in pure Javascript
// Copyright (C) 2016, Pierre Curto
// MIT license
//
// Changes made:
//   - converted to TypeScript
//   - uses native JS BigInt (no external dependencies)
//   - support only for Uint8Array inputs
//   - no constructor function, straight fill & digest
//   - update code removed, only called once, no streams

const P64_1 = BigInt('11400714785074694791');
const P64_2 = BigInt('14029467366897019727');
const P64_3 = BigInt('1609587929392839161');
const P64_4 = BigInt('9650029242287828579');
const P64_5 = BigInt('2870177450012600261'); // mask for a u64, all bits set

const U64 = BigInt('0xffffffffffffffff'); // various constants

const _7n = BigInt(7);

const _11n = BigInt(11);

const _12n = BigInt(12);

const _16n = BigInt(16);

const _18n = BigInt(18);

const _23n = BigInt(23);

const _27n = BigInt(27);

const _29n = BigInt(29);

const _31n = BigInt(31);

const _32n = BigInt(32);

const _33n = BigInt(33);

const _64n = BigInt(64);

const _256n = BigInt(256);

function rotl(a, b) {
  const c = a & U64;
  return (c << b | c >> _64n - b) & U64;
}

function fromU8a(u8a, p, count) {
  const bigints = new Array(count);
  let offset = 0;

  for (let i = 0; i < count; i++, offset += 2) {
    bigints[i] = BigInt(u8a[p + offset] | u8a[p + 1 + offset] << 8);
  }

  let result = _0n;

  for (let i = count - 1; i >= 0; i--) {
    result = (result << _16n) + bigints[i];
  }

  return result;
}

function toU8a(h64) {
  const result = new Uint8Array(8);

  for (let i = 7; i >= 0; i--) {
    result[i] = Number(h64 % _256n);
    h64 = h64 / _256n;
  }

  return result;
}

function state(initSeed) {
  const seed = BigInt(initSeed);
  return {
    seed,
    u8a: new Uint8Array(32),
    u8asize: 0,
    v1: seed + P64_1 + P64_2,
    v2: seed + P64_2,
    v3: seed,
    v4: seed - P64_1
  };
}

function init(state, input) {
  if (input.length < 32) {
    state.u8a.set(input);
    state.u8asize = input.length;
    return state;
  }

  const limit = input.length - 32;
  let p = 0;

  if (limit >= 0) {
    const adjustV = v => P64_1 * rotl(v + P64_2 * fromU8a(input, p, 4), _31n);

    do {
      state.v1 = adjustV(state.v1);
      p += 8;
      state.v2 = adjustV(state.v2);
      p += 8;
      state.v3 = adjustV(state.v3);
      p += 8;
      state.v4 = adjustV(state.v4);
      p += 8;
    } while (p <= limit);
  }

  if (p < input.length) {
    state.u8a.set(input.subarray(p, input.length));
    state.u8asize = input.length - p;
  }

  return state;
}

export function xxhash64(input, initSeed) {
  const {
    seed,
    u8a,
    u8asize,
    v1,
    v2,
    v3,
    v4
  } = init(state(initSeed), input);
  let p = 0;
  let h64 = U64 & BigInt(input.length) + (input.length >= 32 ? ((((rotl(v1, _1n) + rotl(v2, _7n) + rotl(v3, _12n) + rotl(v4, _18n) ^ P64_1 * rotl(v1 * P64_2, _31n)) * P64_1 + P64_4 ^ P64_1 * rotl(v2 * P64_2, _31n)) * P64_1 + P64_4 ^ P64_1 * rotl(v3 * P64_2, _31n)) * P64_1 + P64_4 ^ P64_1 * rotl(v4 * P64_2, _31n)) * P64_1 + P64_4 : seed + P64_5);

  while (p <= u8asize - 8) {
    h64 = U64 & P64_4 + P64_1 * rotl(h64 ^ P64_1 * rotl(P64_2 * fromU8a(u8a, p, 4), _31n), _27n);
    p += 8;
  }

  if (p + 4 <= u8asize) {
    h64 = U64 & P64_3 + P64_2 * rotl(h64 ^ P64_1 * fromU8a(u8a, p, 2), _23n);
    p += 4;
  }

  while (p < u8asize) {
    h64 = U64 & P64_1 * rotl(h64 ^ P64_5 * BigInt(u8a[p++]), _11n);
  }

  h64 = U64 & P64_2 * (h64 ^ h64 >> _33n);
  h64 = U64 & P64_3 * (h64 ^ h64 >> _29n);
  return toU8a(U64 & (h64 ^ h64 >> _32n));
}