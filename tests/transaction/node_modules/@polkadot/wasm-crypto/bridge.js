// Copyright 2019-2021 @polkadot/wasm-crypto authors & contributors
// SPDX-License-Identifier: Apache-2.0

/* eslint-disable @typescript-eslint/no-non-null-assertion */
import { assert, stringToU8a, u8aToString } from '@polkadot/util';
let wasm = null;
let cachegetInt32 = null;
let cachegetUint8 = null;
export async function initWasm(wasmBytes, asmFn, wbg) {
  try {
    assert(typeof WebAssembly !== 'undefined' && wasmBytes && wasmBytes.length, 'WebAssembly is not available in your environment');
    const source = await WebAssembly.instantiate(wasmBytes, {
      wbg
    });
    wasm = source.instance.exports;
  } catch (error) {
    // if we have a valid supplied asm.js, return that
    if (asmFn) {
      wasm = asmFn(wbg);
    } else {
      console.error('FATAL: Unable to initialize @polkadot/wasm-crypto');
      console.error(error);
      wasm = null;
    }
  }
}
export function withWasm(fn) {
  return (...params) => {
    assert(wasm, 'The WASM interface has not been initialized. Ensure that you wait for the initialization Promise with waitReady() from @polkadot/wasm-crypto (or cryptoWaitReady() from @polkadot/util-crypto) before attempting to use WASM-only interfaces.');
    return fn(wasm, ...params);
  };
}
export function getWasm() {
  return wasm;
}
export function getInt32() {
  if (cachegetInt32 === null || cachegetInt32.buffer !== wasm.memory.buffer) {
    cachegetInt32 = new Int32Array(wasm.memory.buffer);
  }

  return cachegetInt32;
}
export function getUint8() {
  if (cachegetUint8 === null || cachegetUint8.buffer !== wasm.memory.buffer) {
    cachegetUint8 = new Uint8Array(wasm.memory.buffer);
  }

  return cachegetUint8;
}
export function getU8a(ptr, len) {
  return getUint8().subarray(ptr / 1, ptr / 1 + len);
}
export function getString(ptr, len) {
  return u8aToString(getU8a(ptr, len));
}
export function allocU8a(arg) {
  const ptr = wasm.__wbindgen_malloc(arg.length * 1);

  getUint8().set(arg, ptr / 1);
  return [ptr, arg.length];
}
export function allocString(arg) {
  return allocU8a(stringToU8a(arg));
}
export function resultU8a() {
  const r0 = getInt32()[8 / 4 + 0];
  const r1 = getInt32()[8 / 4 + 1];
  const ret = getU8a(r0, r1).slice();

  wasm.__wbindgen_free(r0, r1 * 1);

  return ret;
}
export function resultString() {
  return u8aToString(resultU8a());
}