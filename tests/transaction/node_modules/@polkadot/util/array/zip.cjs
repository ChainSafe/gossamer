"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.arrayZip = arrayZip;

// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0
function arrayZip(keys, values) {
  const result = new Array(keys.length);

  for (let i = 0; i < keys.length; i++) {
    result[i] = [keys[i], values[i]];
  }

  return result;
}