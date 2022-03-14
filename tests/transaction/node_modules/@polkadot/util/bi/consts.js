// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { BigInt } from '@polkadot/x-bigint';
/**
 * @name _0n
 * @summary BigInt constant for 0.
 */

export const _0n = BigInt(0);
/**
 * @name _1n
 * @summary BigInt constant for 1.
 */

export const _1n = BigInt(1);
/**
 * @name _1Mn
 * @summary BigInt constant for 1,000,000.
 */

export const _1Mn = BigInt(1000000);
/**
* @name _1Bn
* @summary BigInt constant for 1,000,000,000.
*/

export const _1Bn = BigInt(1000000000);
/**
* @name _1Qn
* @summary BigInt constant for 1,000,000,000,000,000,000.
*/

export const _1Qn = _1Bn * _1Bn;
/**
* @name _2pow53n
* @summary BigInt constant for MAX_SAFE_INTEGER
*/

export const _2pow53n = BigInt(Number.MAX_SAFE_INTEGER);