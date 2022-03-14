// Copyright 2017-2022 @polkadot/util-crypto authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { base58 } from '@scure/base';
import { createDecode, createEncode, createIs, createValidate } from "../base32/helpers.js";
const config = {
  chars: '123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz',
  coder: base58,
  ipfs: 'z',
  type: 'base58'
};
/**
 * @name base58Validate
 * @summary Validates a base58 value.
 * @description
 * Validates that the supplied value is valid base58, throwing exceptions if not
 */

export const base58Validate = createValidate(config);
/**
 * @name base58Decode
 * @summary Decodes a base58 value.
 * @description
 * From the provided input, decode the base58 and return the result as an `Uint8Array`.
 */

export const base58Decode = createDecode(config, base58Validate);
/**
* @name base58Encode
* @summary Creates a base58 value.
* @description
* From the provided input, create the base58 and return the result as a string.
*/

export const base58Encode = createEncode(config);
/**
* @name isBase58
* @description Checks if the input is in base58, returning true/false
*/

export const isBase58 = createIs(base58Validate);