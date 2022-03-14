"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.HARDENED = void 0;
exports.hdValidatePath = hdValidatePath;
// Copyright 2017-2022 @polkadot/util-crypto authors & contributors
// SPDX-License-Identifier: Apache-2.0
const HARDENED = 0x80000000;
exports.HARDENED = HARDENED;

function hdValidatePath(path) {
  if (!path.startsWith('m/')) {
    return false;
  }

  const parts = path.split('/').slice(1);

  for (const p of parts) {
    const n = /^\d+'?$/.test(p) ? parseInt(p.replace(/'$/, ''), 10) : Number.NaN;

    if (isNaN(n) || n >= HARDENED || n < 0) {
      return false;
    }
  }

  return true;
}