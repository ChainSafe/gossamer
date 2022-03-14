// Copyright 2017-2022 @polkadot/util-crypto authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { utils } from '@scure/base';
import { createDecode, createEncode, createIs, createValidate } from "./helpers.js";
const chars = 'abcdefghijklmnopqrstuvwxyz234567';
const config = {
  chars,
  coder: utils.chain( // We define our own chain, the default base32 has padding
  utils.radix2(5), utils.alphabet(chars), {
    decode: input => input.split(''),
    encode: input => input.join('')
  }),
  ipfs: 'b',
  type: 'base32'
};
/**
 * @name base32Validate
 * @summary Validates a base32 value.
 * @description
 * Validates that the supplied value is valid base32, throwing exceptions if not
 */

export const base32Validate = createValidate(config);
/**
* @name isBase32
* @description Checks if the input is in base32, returning true/false
*/

export const isBase32 = createIs(base32Validate);
/**
 * @name base32Decode
 * @summary Delookup a base32 value.
 * @description
 * From the provided input, decode the base32 and return the result as an `Uint8Array`.
 */

export const base32Decode = createDecode(config, base32Validate);
/**
* @name base32Encode
* @summary Creates a base32 value.
* @description
* From the provided input, create the base32 and return the result as a string.
*/

export const base32Encode = createEncode(config);