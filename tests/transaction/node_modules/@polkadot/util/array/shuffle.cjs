"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.arrayShuffle = arrayShuffle;

// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0
function arrayShuffle(input) {
  const result = input.slice();
  let curr = result.length;

  if (curr === 1) {
    return result;
  }

  while (curr !== 0) {
    const rand = Math.floor(Math.random() * curr);
    curr--;
    [result[curr], result[rand]] = [result[rand], result[curr]];
  }

  return result;
}