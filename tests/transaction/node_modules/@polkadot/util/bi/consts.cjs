"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports._2pow53n = exports._1n = exports._1Qn = exports._1Mn = exports._1Bn = exports._0n = void 0;

var _xBigint = require("@polkadot/x-bigint");

// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0

/**
 * @name _0n
 * @summary BigInt constant for 0.
 */
const _0n = (0, _xBigint.BigInt)(0);
/**
 * @name _1n
 * @summary BigInt constant for 1.
 */


exports._0n = _0n;

const _1n = (0, _xBigint.BigInt)(1);
/**
 * @name _1Mn
 * @summary BigInt constant for 1,000,000.
 */


exports._1n = _1n;

const _1Mn = (0, _xBigint.BigInt)(1000000);
/**
* @name _1Bn
* @summary BigInt constant for 1,000,000,000.
*/


exports._1Mn = _1Mn;

const _1Bn = (0, _xBigint.BigInt)(1000000000);
/**
* @name _1Qn
* @summary BigInt constant for 1,000,000,000,000,000,000.
*/


exports._1Bn = _1Bn;

const _1Qn = _1Bn * _1Bn;
/**
* @name _2pow53n
* @summary BigInt constant for MAX_SAFE_INTEGER
*/


exports._1Qn = _1Qn;

const _2pow53n = (0, _xBigint.BigInt)(Number.MAX_SAFE_INTEGER);

exports._2pow53n = _2pow53n;