// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { u8aEq } from "../u8a/eq.js";
import { isU8a } from "./u8a.js";
const WASM_MAGIC = new Uint8Array([0, 97, 115, 109]); // \0asm

/**
 * @name isWasm
 * @summary Tests if the input has a WASM header
 * @description
 * Checks to see if the input Uint8Array contains a valid WASM header
 */

export function isWasm(value) {
  return isU8a(value) && u8aEq(value.subarray(0, 4), WASM_MAGIC);
}