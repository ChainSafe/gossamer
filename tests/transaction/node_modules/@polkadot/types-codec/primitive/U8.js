// Copyright 2017-2022 @polkadot/types-codec authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { UInt } from "../base/UInt.js";
/**
 * @name u8
 * @description
 * An 8-bit unsigned integer
 */

export class u8 extends UInt.with(8) {
  // NOTE without this, we cannot properly determine extensions
  __UIntType = 'u8';
}