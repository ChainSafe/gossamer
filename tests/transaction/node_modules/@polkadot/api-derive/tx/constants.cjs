"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.MORTAL_PERIOD = exports.MAX_FINALITY_LAG = exports.FALLBACK_PERIOD = exports.FALLBACK_MAX_HASH_COUNT = void 0;

var _util = require("@polkadot/util");

// Copyright 2017-2022 @polkadot/api-derive authors & contributors
// SPDX-License-Identifier: Apache-2.0
const FALLBACK_MAX_HASH_COUNT = 250; // default here to 5 min eras, adjusted based on the actual blocktime

exports.FALLBACK_MAX_HASH_COUNT = FALLBACK_MAX_HASH_COUNT;
const FALLBACK_PERIOD = new _util.BN(6 * 1000);
exports.FALLBACK_PERIOD = FALLBACK_PERIOD;
const MAX_FINALITY_LAG = new _util.BN(5);
exports.MAX_FINALITY_LAG = MAX_FINALITY_LAG;
const MORTAL_PERIOD = new _util.BN(5 * 60 * 1000);
exports.MORTAL_PERIOD = MORTAL_PERIOD;