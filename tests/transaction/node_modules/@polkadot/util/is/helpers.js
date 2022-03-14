// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { isFunction } from "./function.js";
import { isObject } from "./object.js";
export function isOn(...fns) {
  return value => (isObject(value) || isFunction(value)) && fns.every(f => isFunction(value[f]));
}
export function isOnObject(...fns) {
  return value => isObject(value) && fns.every(f => isFunction(value[f]));
}