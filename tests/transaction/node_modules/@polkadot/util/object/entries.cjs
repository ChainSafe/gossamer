"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.objectEntries = objectEntries;

// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0

/**
 * @name objectEntries
 * @summary A version of Object.entries that is typed for TS
 */
function objectEntries(obj) {
  return Object.entries(obj);
}