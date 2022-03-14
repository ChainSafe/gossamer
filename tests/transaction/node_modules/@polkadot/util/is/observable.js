// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { isOn } from "./helpers.js";
/**
 * @name isBObservable
 * @summary Tests for a `Observable` object instance.
 * @description
 * Checks to see if the input object is an instance of `BN` (bn.js).
 * @example
 * <BR>
 *
 * ```javascript
 * import { isObservable } from '@polkadot/util';
 *
 * console.log('isObservable', isObservable(...));
 * ```
 */

export const isObservable = isOn('next');