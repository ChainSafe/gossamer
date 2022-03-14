// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { assert } from "../assert.js";

function gt(a, b) {
  return a > b;
}

function lt(a, b) {
  return a < b;
}

function find(items, cmp) {
  assert(items.length >= 1, 'Must provide one or more bigint arguments');
  let result = items[0];

  for (let i = 1; i < items.length; i++) {
    if (cmp(items[i], result)) {
      result = items[i];
    }
  }

  return result;
}
/**
 * @name nMax
 * @summary Finds and returns the highest value in an array of bigint.
 */


export function nMax(...items) {
  return find(items, gt);
}
/**
 * @name nMin
 * @summary Finds and returns the lowest value in an array of bigint.
 */

export function nMin(...items) {
  return find(items, lt);
}