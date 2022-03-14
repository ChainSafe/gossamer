// Copyright 2017-2022 @polkadot/util-crypto authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { base64 } from '@scure/base';
import { createDecode, createEncode, createIs, createValidate } from "../base32/helpers.js";
const config = {
  chars: 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/',
  coder: base64,
  type: 'base64'
};
/**
 * @name base64Validate
 * @summary Validates a base64 value.
 * @description
 * Validates that the supplied value is valid base64
 */

export const base64Validate = createValidate(config);
/**
 * @name isBase64
 * @description Checks if the input is in base64, returning true/false
 */

export const isBase64 = createIs(base64Validate);
/**
 * @name base64Decode
 * @summary Decodes a base64 value.
 * @description
 * From the provided input, decode the base64 and return the result as an `Uint8Array`.
 */

export const base64Decode = createDecode(config, base64Validate);
/**
 * @name base64Encode
 * @summary Creates a base64 value.
 * @description
 * From the provided input, create the base64 and return the result as a string.
 */

export const base64Encode = createEncode(config);