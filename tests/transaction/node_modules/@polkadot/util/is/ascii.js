// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { u8aToU8a } from "../u8a/toU8a.js";
import { isHex } from "./hex.js";
import { isString } from "./string.js";
const FORMAT = [9, 10, 13];
/** @internal */

function isAsciiByte(b) {
  return b < 127 && (b >= 32 || FORMAT.includes(b));
}

function isAsciiChar(s) {
  return isAsciiByte(s.charCodeAt(0));
}
/**
 * @name isAscii
 * @summary Tests if the input is printable ASCII
 * @description
 * Checks to see if the input string or Uint8Array is printable ASCII, 32-127 + formatters
 */


export function isAscii(value) {
  const isStringIn = isString(value);

  if (value) {
    return isStringIn && !isHex(value) ? value.toString().split('').every(isAsciiChar) : u8aToU8a(value).every(isAsciiByte);
  }

  return isStringIn;
}