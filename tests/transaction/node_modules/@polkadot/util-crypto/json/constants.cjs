"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.SCRYPT_LENGTH = exports.NONCE_LENGTH = exports.ENCODING_VERSION = exports.ENCODING_NONE = exports.ENCODING = void 0;
// Copyright 2017-2022 @polkadot/util-crypto authors & contributors
// SPDX-License-Identifier: Apache-2.0
const ENCODING = ['scrypt', 'xsalsa20-poly1305'];
exports.ENCODING = ENCODING;
const ENCODING_NONE = ['none'];
exports.ENCODING_NONE = ENCODING_NONE;
const ENCODING_VERSION = '3';
exports.ENCODING_VERSION = ENCODING_VERSION;
const NONCE_LENGTH = 24;
exports.NONCE_LENGTH = NONCE_LENGTH;
const SCRYPT_LENGTH = 32 + 3 * 4;
exports.SCRYPT_LENGTH = SCRYPT_LENGTH;