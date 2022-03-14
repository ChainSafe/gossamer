// Copyright 2017-2022 @polkadot/api-derive authors & contributors
// SPDX-License-Identifier: Apache-2.0
import { map, of } from 'rxjs';
import { memo } from "../util/index.js";
export function referendumIds(instanceId, api) {
  return memo(instanceId, () => {
    var _api$query$democracy;

    return (_api$query$democracy = api.query.democracy) !== null && _api$query$democracy !== void 0 && _api$query$democracy.lowestUnbaked ? api.queryMulti([api.query.democracy.lowestUnbaked, api.query.democracy.referendumCount]).pipe(map(([first, total]) => total.gt(first) // eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
    ? [...Array(total.sub(first).toNumber())].map((_, i) => first.addn(i)) : [])) : of([]);
  });
}