"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.promisify = promisify;

// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0

/**
 * @name promisify
 * @summary Wraps an async callback into a `Promise`
 * @description
 * Wraps the supplied async function `fn` that has a standard JS callback `(error: Error, result: any)` into a `Promise`, passing the supplied parameters. When `error` is set, the Promise is rejected, else the Promise resolves with the `result` value.
 * @example
 * <BR>
 *
 * ```javascript
 * const { promisify } from '@polkadot/util';
 *
 * await promisify(null, ((a, cb) => cb(null, a), true); // resolves with `true`
 * await promisify(null, (cb) => cb(new Error('error!'))); // rejects with `error!`
 * ```
 */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
function promisify(self, fn) {
  for (var _len = arguments.length, params = new Array(_len > 2 ? _len - 2 : 0), _key = 2; _key < _len; _key++) {
    params[_key - 2] = arguments[_key];
  }

  return new Promise((resolve, reject) => {
    fn.apply(self, params.concat((error, result) => {
      if (error) {
        reject(error);
      } else {
        resolve(result);
      }
    }));
  });
}