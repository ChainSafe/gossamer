// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { BN } from "./bn.js";
/**
 * @name BN_ZERO
 * @summary BN constant for 0.
 */

export const BN_ZERO = new BN(0);
/**
 * @name BN_ONE
 * @summary BN constant for 1.
 */

export const BN_ONE = new BN(1);
/**
 * @name BN_TWO
 * @summary BN constant for 2.
 */

export const BN_TWO = new BN(2);
/**
 * @name BN_THREE
 * @summary BN constant for 3.
 */

export const BN_THREE = new BN(3);
/**
 * @name BN_FOUR
 * @summary BN constant for 4.
 */

export const BN_FOUR = new BN(4);
/**
 * @name BN_FIVE
 * @summary BN constant for 5.
 */

export const BN_FIVE = new BN(5);
/**
 * @name BN_SIX
 * @summary BN constant for 6.
 */

export const BN_SIX = new BN(6);
/**
 * @name BN_SEVEN
 * @summary BN constant for 7.
 */

export const BN_SEVEN = new BN(7);
/**
 * @name BN_EIGHT
 * @summary BN constant for 8.
 */

export const BN_EIGHT = new BN(8);
/**
 * @name BN_NINE
 * @summary BN constant for 9.
 */

export const BN_NINE = new BN(9);
/**
 * @name BN_TEN
 * @summary BN constant for 10.
 */

export const BN_TEN = new BN(10);
/**
 * @name BN_HUNDRED
 * @summary BN constant for 100.
 */

export const BN_HUNDRED = new BN(100);
/**
 * @name BN_THOUSAND
 * @summary BN constant for 1,000.
 */

export const BN_THOUSAND = new BN(1000);
/**
 * @name BN_MILLION
 * @summary BN constant for 1,000,000.
 */

export const BN_MILLION = new BN(1000000);
/**
 * @name BN_BILLION
 * @summary BN constant for 1,000,000,000.
 */

export const BN_BILLION = new BN(1000000000);
/**
 * @name BN_QUINTILL
 * @summary BN constant for 1,000,000,000,000,000,000.
 */

export const BN_QUINTILL = BN_BILLION.mul(BN_BILLION);
/**
 * @name BN_MAX_INTEGER
 * @summary BN constant for MAX_SAFE_INTEGER
 */

export const BN_MAX_INTEGER = new BN(Number.MAX_SAFE_INTEGER);