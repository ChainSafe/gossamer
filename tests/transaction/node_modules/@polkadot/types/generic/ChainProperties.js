// Copyright 2017-2022 @polkadot/types authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { Json } from '@polkadot/types-codec';
import { isFunction, isNull, isUndefined } from '@polkadot/util';

function createValue(registry, type, value, asArray = true) {
  // We detect codec here as well - when found, generally this is constructed from itself
  if (value && isFunction(value.unwrapOrDefault)) {
    return value;
  }

  return registry.createTypeUnsafe(type, [asArray ? isNull(value) || isUndefined(value) ? null : Array.isArray(value) ? value : [value] : value]);
}

function decodeValue(registry, key, value) {
  return key === 'ss58Format' ? createValue(registry, 'Option<u32>', value, false) : key === 'tokenDecimals' ? createValue(registry, 'Option<Vec<u32>>', value) : key === 'tokenSymbol' ? createValue(registry, 'Option<Vec<Text>>', value) : value;
}

function decode(registry, value) {
  return ( // allow decoding from a map as well (ourselves)
  value && isFunction(value.entries) ? [...value.entries()] : Object.entries(value || {})).reduce((all, [key, value]) => {
    all[key] = decodeValue(registry, key, value);
    return all;
  }, {
    ss58Format: registry.createTypeUnsafe('Option<u32>', []),
    tokenDecimals: registry.createTypeUnsafe('Option<Vec<u32>>', []),
    tokenSymbol: registry.createTypeUnsafe('Option<Vec<Text>>', [])
  });
}

export class GenericChainProperties extends Json {
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