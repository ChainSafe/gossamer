"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.hasEq = hasEq;

var _util = require("@polkadot/util");

// Copyright 2017-2022 @polkadot/types-codec authors & contributors
// SPDX-License-Identifier: Apache-2.0
function hasEq(o) {
  // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access
  return (0, _util.isFunction)(o.eq);
}