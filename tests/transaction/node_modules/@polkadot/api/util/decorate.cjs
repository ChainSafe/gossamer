"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.decorateDeriveSections = decorateDeriveSections;

var _apiDerive = require("@polkadot/api-derive");

// Copyright 2017-2022 @polkadot/api authors & contributors
// SPDX-License-Identifier: Apache-2.0

/**
 * This is a section decorator which keeps all type information.
 */
function decorateDeriveSections(decorateMethod, derives) {
  const getKeys = s => Object.keys(derives[s]);

  const creator = (s, m) => decorateMethod(derives[s][m]);

  const result = {};
  const names = Object.keys(derives);

  for (let i = 0; i < names.length; i++) {
    (0, _apiDerive.lazyDeriveSection)(result, names[i], getKeys, creator);
  }

  return result;
}