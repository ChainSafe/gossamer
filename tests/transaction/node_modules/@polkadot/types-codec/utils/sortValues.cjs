"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.sortAsc = sortAsc;
exports.sortMap = sortMap;
exports.sortSet = sortSet;

var _util = require("@polkadot/util");

// Copyright 2017-2022 @polkadot/types-codec authors & contributors
// SPDX-License-Identifier: Apache-2.0

/** @internal **/
function isArrayLike(arg) {
  return arg instanceof Uint8Array || Array.isArray(arg);
}
/** @internal **/


function isCodec(arg) {
  return (0, _util.isFunction)(arg && arg.toU8a);
}
/** @internal **/


function isEnum(arg) {
  return isCodec(arg) && (0, _util.isNumber)(arg.index) && isCodec(arg.value);
}
/** @internal */


function isNumberLike(arg) {
  return (0, _util.isNumber)(arg) || (0, _util.isBn)(arg) || (0, _util.isBigInt)(arg);
}
/** @internal */


function sortArray(a, b) {
  // Vec, Tuple, Bytes etc.
  let sortRes = 0;
  const minLen = Math.min(a.length, b.length);

  for (let i = 0; i < minLen; ++i) {
    sortRes = sortAsc(a[i], b[i]);

    if (sortRes !== 0) {
      return sortRes;
    }
  }

  return a.length - b.length;
}
/**
* Sort keys/values of BTreeSet/BTreeMap in ascending order for encoding compatibility with Rust's BTreeSet/BTreeMap
* (https://doc.rust-lang.org/stable/std/collections/struct.BTreeSet.html)
* (https://doc.rust-lang.org/stable/std/collections/struct.BTreeMap.html)
*/


function sortAsc(a, b) {
  if (isNumberLike(a) && isNumberLike(b)) {
    return (0, _util.bnToBn)(a).cmp((0, _util.bnToBn)(b));
  } else if (a instanceof Map && b instanceof Map) {
    return sortAsc(Array.from(a.values()), Array.from(b.values()));
  } else if (isEnum(a) && isEnum(b)) {
    return sortAsc(a.index, b.index) || sortAsc(a.value, b.value);
  } else if (isArrayLike(a) && isArrayLike(b)) {
    return sortArray(a, b);
  } else if (isCodec(a) && isCodec(b)) {
    // Text, Bool etc.
    return sortAsc(a.toU8a(true), b.toU8a(true));
  }

  throw new Error(`Attempting to sort unrecognized values: ${(0, _util.stringify)(a)} (typeof ${typeof a}) <-> ${(0, _util.stringify)(b)} (typeof ${typeof b})`);
}

function sortSet(set) {
  return new Set(Array.from(set).sort(sortAsc));
}

function sortMap(map) {
  return new Map(Array.from(map.entries()).sort((_ref, _ref2) => {
    let [keyA] = _ref;
    let [keyB] = _ref2;
    return sortAsc(keyA, keyB);
  }));
}