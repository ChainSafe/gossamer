// Copyright 2017-2022 @polkadot/api-derive authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { map, of, startWith } from 'rxjs';
import { memo } from "../util/index.js";
let indicesCache = null;

function queryAccounts(api) {
  return api.query.indices.accounts.entries().pipe(map(entries => entries.reduce((indexes, [key, idOpt]) => {
    if (idOpt.isSome) {
      indexes[idOpt.unwrap()[0].toString()] = api.registry.createType('AccountIndex', key.args[0]);
    }

    return indexes;
  }, {})));
}
/**
 * @name indexes
 * @returns Returns all the indexes on the system.
 * @description This is an unwieldly query since it loops through
 * all of the enumsets and returns all of the values found. This could be up to 32k depending
 * on the number of active accounts in the system
 * @example
 * <BR>
 *
 * ```javascript
 * api.derive.accounts.indexes((indexes) => {
 *   console.log('All existing AccountIndexes', indexes);
 * });
 * ```
 */


export function indexes(instanceId, api) {
  return memo(instanceId, () => indicesCache ? of(indicesCache) : (api.query.indices ? queryAccounts(api).pipe(startWith({})) : of({})).pipe(map(indices => {
    indicesCache = indices;
    return indices;
  })));
}