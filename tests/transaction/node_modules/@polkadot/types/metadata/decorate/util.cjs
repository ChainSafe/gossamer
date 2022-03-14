"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.objectNameToString = exports.objectNameToCamel = void 0;

var _util = require("@polkadot/util");

// Copyright 2017-2022 @polkadot/types authors & contributors
// SPDX-License-Identifier: Apache-2.0
function convert(fn) {
  return _ref => {
    let {
      name
    } = _ref;
    return fn(name);
  };
}

const objectNameToCamel = convert(_util.stringCamelCase);
exports.objectNameToCamel = objectNameToCamel;
const objectNameToString = convert(n => n.toString());
exports.objectNameToString = objectNameToString;