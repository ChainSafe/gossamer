// Copyright 2019-2021 @polkadot/wasm-crypto authors & contributors
// SPDX-License-Identifier: Apache-2.0

/* eslint-disable camelcase */
import { getRandomValues } from '@polkadot/x-randomvalues';
import { getString, getU8a } from "./bridge.js";
const DEFAULT_CRYPTO = {
  getRandomValues
};
const DEFAULT_SELF = {
  crypto: DEFAULT_CRYPTO
};
const heap = new Array(32).fill(undefined).concat(undefined, null, true, false);
let heapNext = heap.length;

function getObject(idx) {
  return heap[idx];
}

function dropObject(idx) {
  if (idx < 36) {
    return;
  }

  heap[idx] = heapNext;
  heapNext = idx;
}

function takeObject(idx) {
  const ret = getObject(idx);
  dropObject(idx);
  return ret;
}

function addObject(obj) {
  if (heapNext === heap.length) {
    heap.push(heap.length + 1);
  }

  const idx = heapNext;
  heapNext = heap[idx];
  heap[idx] = obj;
  return idx;
}

export function __wbindgen_is_undefined(idx) {
  return getObject(idx) === undefined;
}
export function __wbindgen_throw(ptr, len) {
  throw new Error(getString(ptr, len));
}
export function __wbg_self_1b7a39e3a92c949c() {
  return addObject(DEFAULT_SELF);
}
export function __wbg_require_604837428532a733(ptr, len) {
  throw new Error(`Unable to require ${getString(ptr, len)}`);
} // eslint-disable-next-line @typescript-eslint/no-unused-vars

export function __wbg_crypto_968f1772287e2df0(_idx) {
  return addObject(DEFAULT_CRYPTO);
} // eslint-disable-next-line @typescript-eslint/no-unused-vars

export function __wbg_getRandomValues_a3d34b4fee3c2869(_idx) {
  return addObject(DEFAULT_CRYPTO.getRandomValues);
} // eslint-disable-next-line @typescript-eslint/no-unused-vars

export function __wbg_getRandomValues_f5e14ab7ac8e995d(_arg0, ptr, len) {
  DEFAULT_CRYPTO.getRandomValues(getU8a(ptr, len));
} // eslint-disable-next-line @typescript-eslint/no-unused-vars

export function __wbg_randomFillSync_d5bd2d655fdf256a(_idx, _ptr, _len) {
  throw new Error('randomFillsync is not available'); // getObject(idx).randomFillSync(getU8a(ptr, len));
}
export function __wbindgen_object_drop_ref(idx) {
  takeObject(idx);
}
export function abort() {
  throw new Error('abort');
}