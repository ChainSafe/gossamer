"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.__wbg_crypto_968f1772287e2df0 = __wbg_crypto_968f1772287e2df0;
exports.__wbg_getRandomValues_a3d34b4fee3c2869 = __wbg_getRandomValues_a3d34b4fee3c2869;
exports.__wbg_getRandomValues_f5e14ab7ac8e995d = __wbg_getRandomValues_f5e14ab7ac8e995d;
exports.__wbg_randomFillSync_d5bd2d655fdf256a = __wbg_randomFillSync_d5bd2d655fdf256a;
exports.__wbg_require_604837428532a733 = __wbg_require_604837428532a733;
exports.__wbg_self_1b7a39e3a92c949c = __wbg_self_1b7a39e3a92c949c;
exports.__wbindgen_is_undefined = __wbindgen_is_undefined;
exports.__wbindgen_object_drop_ref = __wbindgen_object_drop_ref;
exports.__wbindgen_throw = __wbindgen_throw;
exports.abort = abort;

var _xRandomvalues = require("@polkadot/x-randomvalues");

var _bridge = require("./bridge.cjs");

// Copyright 2019-2021 @polkadot/wasm-crypto authors & contributors
// SPDX-License-Identifier: Apache-2.0

/* eslint-disable camelcase */
const DEFAULT_CRYPTO = {
  getRandomValues: _xRandomvalues.getRandomValues
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

function __wbindgen_is_undefined(idx) {
  return getObject(idx) === undefined;
}

function __wbindgen_throw(ptr, len) {
  throw new Error((0, _bridge.getString)(ptr, len));
}

function __wbg_self_1b7a39e3a92c949c() {
  return addObject(DEFAULT_SELF);
}

function __wbg_require_604837428532a733(ptr, len) {
  throw new Error(`Unable to require ${(0, _bridge.getString)(ptr, len)}`);
} // eslint-disable-next-line @typescript-eslint/no-unused-vars


function __wbg_crypto_968f1772287e2df0(_idx) {
  return addObject(DEFAULT_CRYPTO);
} // eslint-disable-next-line @typescript-eslint/no-unused-vars


function __wbg_getRandomValues_a3d34b4fee3c2869(_idx) {
  return addObject(DEFAULT_CRYPTO.getRandomValues);
} // eslint-disable-next-line @typescript-eslint/no-unused-vars


function __wbg_getRandomValues_f5e14ab7ac8e995d(_arg0, ptr, len) {
  DEFAULT_CRYPTO.getRandomValues((0, _bridge.getU8a)(ptr, len));
} // eslint-disable-next-line @typescript-eslint/no-unused-vars


function __wbg_randomFillSync_d5bd2d655fdf256a(_idx, _ptr, _len) {
  throw new Error('randomFillsync is not available'); // getObject(idx).randomFillSync(getU8a(ptr, len));
}

function __wbindgen_object_drop_ref(idx) {
  takeObject(idx);
}

function abort() {
  throw new Error('abort');
}