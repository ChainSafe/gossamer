"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.BN_LE_OPTS = exports.BN_LE_512_OPTS = exports.BN_LE_32_OPTS = exports.BN_LE_256_OPTS = exports.BN_LE_16_OPTS = exports.BN_BE_OPTS = exports.BN_BE_32_OPTS = exports.BN_BE_256_OPTS = void 0;
// Copyright 2017-2022 @polkadot/util-crypto authors & contributors
// SPDX-License-Identifier: Apache-2.0
const BN_BE_OPTS = {
  isLe: false
};
exports.BN_BE_OPTS = BN_BE_OPTS;
const BN_LE_OPTS = {
  isLe: true
};
exports.BN_LE_OPTS = BN_LE_OPTS;
const BN_LE_16_OPTS = {
  bitLength: 16,
  isLe: true
};
exports.BN_LE_16_OPTS = BN_LE_16_OPTS;
const BN_BE_32_OPTS = {
  bitLength: 32,
  isLe: false
};
exports.BN_BE_32_OPTS = BN_BE_32_OPTS;
const BN_LE_32_OPTS = {
  bitLength: 32,
  isLe: true
};
exports.BN_LE_32_OPTS = BN_LE_32_OPTS;
const BN_BE_256_OPTS = {
  bitLength: 256,
  isLe: false
};
exports.BN_BE_256_OPTS = BN_BE_256_OPTS;
const BN_LE_256_OPTS = {
  bitLength: 256,
  isLe: true
};
exports.BN_LE_256_OPTS = BN_LE_256_OPTS;
const BN_LE_512_OPTS = {
  bitLength: 512,
  isLe: true
};
exports.BN_LE_512_OPTS = BN_LE_512_OPTS;