// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0

/**
 * @name objectClear
 * @summary Removes all the keys from the input object
 */
export function objectClear(value) {
  const keys = Object.keys(value);

  for (let i = 0; i < keys.length; i++) {
    delete value[keys[i]];
  }

  return value;
}