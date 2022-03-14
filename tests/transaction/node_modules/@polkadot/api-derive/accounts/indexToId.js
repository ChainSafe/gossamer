// Copyright 2017-2022 @polkadot/api-derive authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { map, of } from 'rxjs';
import { memo } from "../util/index.js";
/**
 * @name indexToId
 * @param {( AccountIndex | string )} accountIndex - An accounts index in different formats.
 * @returns Returns the corresponding AccountId.
 * @example
 * <BR>
 *
 * ```javascript
 * api.derive.accounts.indexToId('F7Hs', (accountId) => {
 *   console.log(`The AccountId of F7Hs is ${accountId}`);
 * });
 * ```
 */

export function indexToId(instanceId, api) {
  return memo(instanceId, accountIndex => api.query.indices ? api.query.indices.accounts(accountIndex).pipe(map(optResult => optResult.unwrapOr([])[0])) : of(undefined));
}