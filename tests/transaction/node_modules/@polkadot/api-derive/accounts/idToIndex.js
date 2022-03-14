// Copyright 2017-2022 @polkadot/api-derive authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { map } from 'rxjs';
import { memo } from "../util/index.js";
/**
 * @name idToIndex
 * @param {( AccountId | string )} accountId - An accounts Id in different formats.
 * @returns Returns the corresponding AccountIndex.
 * @example
 * <BR>
 *
 * ```javascript
 * const ALICE = '5GrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY';
 * api.derive.accounts.idToIndex(ALICE, (accountIndex) => {
 *   console.log(`The AccountIndex of ${ALICE} is ${accountIndex}`);
 * });
 * ```
 */

export function idToIndex(instanceId, api) {
  return memo(instanceId, accountId => api.derive.accounts.indexes().pipe(map(indexes => (indexes || {})[accountId.toString()])));
}