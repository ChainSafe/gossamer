"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.flattenUniq = flattenUniq;

// Copyright 2017-2022 @polkadot/types authors & contributors
// SPDX-License-Identifier: Apache-2.0

/** @internal */
function flattenUniq(list) {
  let result = arguments.length > 1 && arguments[1] !== undefined ? arguments[1] : [];

  for (let i = 0; i < list.length; i++) {
    const entry = list[i];

    if (Array.isArray(entry)) {
      flattenUniq(entry, result);
    } else {
      result.push(entry);
    }
  }

  return [...new Set(result)];
}