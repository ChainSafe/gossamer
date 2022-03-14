// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { isBigInt } from "./is/bigInt.js";
/** @internal */

function replacer(_, v) {
  return isBigInt(v) ? v.toString() : v;
}
/**
 * @name stringify
 * @summary Performs a JSON.stringify, with BigInt handling
 * @description A wrapper for JSON.stringify that handles BigInt values transparently, converting them to string. No differences from the native JSON.stringify function otherwise.
 */


export function stringify(value, space) {
  return JSON.stringify(value, replacer, space);
}