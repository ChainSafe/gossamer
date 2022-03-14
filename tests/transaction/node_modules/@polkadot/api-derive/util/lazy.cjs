"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.lazyDeriveSection = lazyDeriveSection;

var _util = require("@polkadot/util");

// Copyright 2017-2022 @polkadot/api-derive authors & contributors
// SPDX-License-Identifier: Apache-2.0
function lazyDeriveSection(result, section, getKeys, creator) {
  (0, _util.lazyMethod)(result, section, () => (0, _util.lazyMethods)({}, getKeys(section), method => creator(section, method)));
}