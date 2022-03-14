"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.GenericChainProperties = void 0;

var _typesCodec = require("@polkadot/types-codec");

var _util = require("@polkadot/util");

// Copyright 2017-2022 @polkadot/types authors & contributors
// SPDX-License-Identifier: Apache-2.0
function createValue(registry, type, value) {
  let asArray = arguments.length > 3 && arguments[3] !== undefined ? arguments[3] : true;

  // We detect codec here as well - when found, generally this is constructed from itself
  if (value && (0, _util.isFunction)(value.unwrapOrDefault)) {
    return value;
  }

  return registry.createTypeUnsafe(type, [asArray ? (0, _util.isNull)(value) || (0, _util.isUndefined)(value) ? null : Array.isArray(value) ? value : [value] : value]);
}

function decodeValue(registry, key, value) {
  return key === 'ss58Format' ? createValue(registry, 'Option<u32>', value, false) : key === 'tokenDecimals' ? createValue(registry, 'Option<Vec<u32>>', value) : key === 'tokenSymbol' ? createValue(registry, 'Option<Vec<Text>>', value) : value;
}

function decode(registry, value) {
  return ( // allow decoding from a map as well (ourselves)
  value && (0, _util.isFunction)(value.entries) ? [...value.entries()] : Object.entries(value || {})).reduce((all, _ref) => {
    let [key, value] = _ref;
    all[key] = decodeValue(registry, key, value);
    return all;
  }, {
    ss58Format: registry.createTypeUnsafe('Option<u32>', []),
    tokenDecimals: registry.createTypeUnsafe('Option<Vec<u32>>', []),
    tokenSymbol: registry.createTypeUnsafe('Option<Vec<Text>>', [])
  });
}

class GenericChainProperties extends _typesCodec.Json {
  constructor(registry, value) {
    super(registry, decode(registry, value));
  }
  /**
   * @description The chain ss58Format
   */


  get ss58Format() {
    return this.getT('ss58Format');
  }
  /**
   * @description The decimals for each of the tokens
   */


  get tokenDecimals() {
    return this.getT('tokenDecimals');
  }
  /**
   * @description The symbols for the tokens
   */


  get tokenSymbol() {
    return this.getT('tokenSymbol');
  }

}

exports.GenericChainProperties = GenericChainProperties;