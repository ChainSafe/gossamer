// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { TextEncoder } from '@polkadot/x-textencoder';
const encoder = new TextEncoder();
/**
 * @name stringToU8a
 * @summary Creates a Uint8Array object from a utf-8 string.
 * @description
 * String input values return the actual encoded `UInt8Array`. `null` or `undefined` values returns an empty encoded array.
 * @example
 * <BR>
 *
 * ```javascript
 * import { stringToU8a } from '@polkadot/util';
 *
 * stringToU8a('hello'); // [0x68, 0x65, 0x6c, 0x6c, 0x6f]
 * ```
 */
// eslint-disable-next-line @typescript-eslint/ban-types

export function stringToU8a(value) {
  return value ? encoder.encode(value.toString()) : new Uint8Array();
}