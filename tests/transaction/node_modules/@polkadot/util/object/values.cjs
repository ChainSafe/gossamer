"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.objectValues = objectValues;

// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0

/**
 * @name objectValues
 * @summary A version of Object.values that is typed for TS
 */
function objectValues(obj) {
  return Object.values(obj);
}