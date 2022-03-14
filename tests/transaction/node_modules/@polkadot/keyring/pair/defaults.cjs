"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.SEED_LENGTH = exports.SEC_LENGTH = exports.SALT_LENGTH = exports.PUB_LENGTH = exports.PKCS8_HEADER = exports.PKCS8_DIVIDER = void 0;
// Copyright 2017-2022 @polkadot/keyring authors & contributors
// SPDX-License-Identifier: Apache-2.0
const PKCS8_DIVIDER = new Uint8Array([161, 35, 3, 33, 0]);
exports.PKCS8_DIVIDER = PKCS8_DIVIDER;
const PKCS8_HEADER = new Uint8Array([48, 83, 2, 1, 1, 48, 5, 6, 3, 43, 101, 112, 4, 34, 4, 32]);
exports.PKCS8_HEADER = PKCS8_HEADER;
const PUB_LENGTH = 32;
exports.PUB_LENGTH = PUB_LENGTH;
const SALT_LENGTH = 32;
exports.SALT_LENGTH = SALT_LENGTH;
const SEC_LENGTH = 64;
exports.SEC_LENGTH = SEC_LENGTH;
const SEED_LENGTH = 32;
exports.SEED_LENGTH = SEED_LENGTH;